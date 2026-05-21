# `/s1/video/generations` 接口新旧计费策略对比

## 一、计费公式

```
预扣额度(quota) = 基础价格 × 分组倍率(groupRatio) × OtherRatios乘积
```

- `基础价格`：由后台管理面板的 `modelPrice` / `defaultModelPrice` / `modelRatio` 配置确定
- `groupRatio`：分组倍率，默认为 1.0
- `OtherRatios`：**代码层计算**，根据请求特征返回的额外倍率系数

---

## 二、旧策略（仅检测 video_url）

### 代码逻辑

```go
// adaptor.go（旧，已注释保留）
if req.HasVideo() {
    ratio := GetVideoInputRatio(modelName)  // 仅根据模型名返回固定折扣
    return map[string]float64{"video_input": ratio}
}
return nil  // 无视频 → 不应用任何倍率
```

### constants.go 旧映射

```go
var videoInputRatioMap = map[string]float64{
    "doubao-seedance-2-0-260128":      28.0 / 46.0,  // ≈0.6087
    "doubao-seedance-2-0-fast-260128": 22.0 / 37.0,  // ≈0.5946
}
```

### 实际计费结果

| 场景 | OtherRatio | 说明 |
|------|------------|------|
| 纯文生视频（任何分辨率） | 不返回（×1.0） | 按后台配置的基础价格扣费 |
| 视频生视频（任何分辨率） | `video_input: 0.6087` | 基础价格 × 0.6087 |

**核心问题：** 分辨率不参与计费判断，1080p 和 480p/720p 扣费相同。

---

## 三、新策略（video × resolution 四档）

### 代码逻辑

```go
// adaptor.go（新）
hasVideo := req.HasVideo()
tier := resolveResolutionTier(req.Resolution)  // 1080p → high, 其余 → low
key := tierKey(hasVideo, tier)                 // 生成四档 key

if key == "low" {
    return nil  // 基准档，不返回倍率
}
ratio := ratioMap[key]
return map[string]float64{"video_resolution_input": ratio}
```

### constants.go 新映射

```go
var resolutionVideoRatioMap = map[string]map[string]float64{
    "doubao-seedance-2-0-260128": {
        "low_video":  28.0 / 46.0,  // ≈0.6087 (480p/720p + 视频)
        "high_video": 31.0 / 46.0,  // ≈0.6739 (1080p + 视频)
        "high":       51.0 / 46.0,  // ≈1.1087 (1080p 无视频)
    },
    "doubao-seedance-2-0-fast-260128": {
        "low_video":  22.0 / 37.0,  // ≈0.5946 (480p/720p + 视频)
        "high_video": 31.0 / 46.0,  // ≈0.6739 (1080p + 视频)
        "high":       51.0 / 46.0,  // ≈1.1087 (1080p 无视频)
    },
}
```

### 辅助函数

| 函数 | 作用 |
|------|------|
| `resolveResolutionTier(resolution)` | `1080p` → `tierHigh`，其余/未传 → `tierLow` |
| `tierKey(hasVideo, tier)` | 组合生成 `"low"` / `"high"` / `"low_video"` / `"high_video"` |

### 实际计费结果

| 场景 | resolution | video_url | OtherRatio | 扣费 |
|------|-----------|-----------|------------|------|
| 文生视频（低清） | 480p / 720p / 未传 | 无 | 不返回（×1.0） | 基础价格 |
| 文生视频（高清） | 1080p | 无 | `video_resolution_input: 1.1087` | 基础价格 × 1.1087 |
| 视频生视频（低清） | 480p / 720p | 有 | `video_resolution_input: 0.6087` | 基础价格 × 0.6087 |
| 视频生视频（高清） | 1080p | 有 | `video_resolution_input: 0.6739` | 基础价格 × 0.6739 |

---

## 四、具体数值对比

假设后台配置的基础价格为 **1.0**（对应 480p 无视频 = 46 元/百万tokens），`QuotaPerUnit = 500,000`：

| 场景 | 旧策略 quota | 新策略 quota | 差异 |
|------|-------------|-------------|------|
| 480p 文生视频 | 500,000 | 500,000 | **不变** |
| **1080p 文生视频** | 500,000 | **554,350** | **+10.87%** |
| 480p 视频生视频 | 304,350 | 304,350 | **不变** |
| **1080p 视频生视频** | 304,350 | **336,950** | **+10.73%** |

### 逐场景分析

#### 场景 1：480p 文生视频（不变）

- **旧策略**：无 video_url → 不返回 OtherRatio → 扣 500,000 quota
- **新策略**：无 video + tierLow → key="low"（基准档）→ 不返回 OtherRatio → 扣 500,000 quota
- **结论**：完全一致

#### 场景 2：1080p 文生视频（修正）

- **旧策略**：无 video_url → 不返回 OtherRatio → 扣 500,000 quota
- **新策略**：无 video + tierHigh → key="high" → 返回 `1.1087` → 扣 554,350 quota
- **问题**：旧策略按 46 元扣费，但官方 1080p 无视频是 51 元，少扣了 10.87%
- **结论**：新策略对齐官方价格

#### 场景 3：480p 视频生视频（不变）

- **旧策略**：有 video_url → 返回 `video_input: 0.6087` → 扣 304,350 quota
- **新策略**：有 video + tierLow → key="low_video" → 返回 `0.6087` → 扣 304,350 quota
- **结论**：数值完全一致（比率未变）

#### 场景 4：1080p 视频生视频（修正）

- **旧策略**：有 video_url → 返回 `video_input: 0.6087` → 扣 304,350 quota
- **新策略**：有 video + tierHigh → key="high_video" → 返回 `0.6739` → 扣 336,950 quota
- **问题**：旧策略用了 480p 的折扣 28/46，但官方 1080p 有视频是 31 元，应使用 31/46 ≈ 0.6739
- **结论**：新策略对齐官方价格

---

## 五、官方价格基准

| 分辨率 | 含视频输入 | 不含视频输入 |
|--------|-----------|-------------|
| 480p | 28 元/百万tokens | 46 元/百万tokens |
| 720p | 28 元/百万tokens | 46 元/百万tokens |
| 1080p | 31 元/百万tokens | 51 元/百万tokens |

新策略以 **480p/720p 不含视频输入（46 元）** 为基准价，其余三档通过 OtherRatio 叠加。

---

## 六、旧代码保留说明

旧策略代码已注释保留，位于：

| 文件 | 注释内容 |
|------|----------|
| `relay/channel/task/doubao/constants.go` | `videoInputRatioMap` 和 `GetVideoInputRatio` 函数 |
| `relay/channel/task/doubao/adaptor.go` | 旧 `EstimateBilling` 函数实现 |

如需回滚到旧策略，取消注释新代码并恢复旧代码即可。

---

## 七、文件变更清单

| 文件 | 变更 |
|------|------|
| `relay/channel/task/doubao/constants.go` | 注释旧映射 + 旧函数，新增四档映射 + 新函数 |
| `relay/channel/task/doubao/adaptor.go` | 注释旧 `EstimateBilling`，新增四档逻辑 + 辅助函数 |
| `docs/change-logs/s1-video-resolution-billing.md` | 变更文档（四档矩阵、代码改动说明） |
| `docs/change-logs/s1-video-billing-comparison.md` | 本文档（新旧策略详细对比） |
