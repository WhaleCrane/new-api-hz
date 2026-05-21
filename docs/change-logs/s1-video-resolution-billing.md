# `/s1/video/generations` 接口按分辨率+视频输入分档计费

## 变更原因

火山方舟 Seedance 2.0 对不同分辨率有不同单价，项目原有计费仅检测 `video_url` 是否存在来应用单一折扣，未区分分辨率。1080p 和 480p/720p 官方价格差异明显，需要支持四档综合计费。

## 官方价格表

| 分辨率 | 含视频输入 | 不含视频输入 |
|--------|-----------|-------------|
| 480p | 28 元/百万tokens | 46 元/百万tokens |
| 720p | 28 元/百万tokens | 46 元/百万tokens |
| 1080p | 31 元/百万tokens | 51 元/百万tokens |

## 计费矩阵

以 480p/720p 不含视频输入（46 元/百万tokens）为基准价：

| 档位 | 条件 | OtherRatio | 计算 |
|------|------|------------|------|
| `low`（基准） | 无视频 + 480p/720p | 不返回（×1.0） | 46/46 |
| `high` | 无视频 + 1080p | ≈ 1.1087 | 51/46 |
| `low_video` | 有视频 + 480p/720p | ≈ 0.6087 | 28/46 |
| `high_video` | 有视频 + 1080p | ≈ 0.6739 | 31/46 |

## 旧计费方式（已注释保留）

**旧逻辑：** 仅检测 content 中是否存在 `video_url` 类型项。

| 条件 | OtherRatio |
|------|------------|
| 无 video_url | 不返回（基准价 × 1.0） |
| 有 video_url | `video_input: 0.6087`（2.0）或 `0.5946`（2.0 Fast） |

旧代码保留为注释块，位于以下函数旁：
- `constants.go`：`videoInputRatioMap` 和 `GetVideoInputRatio`
- `adaptor.go`：旧 `EstimateBilling` 函数

## 新计费方式

### 1. 比率映射 — `relay/channel/task/doubao/constants.go`

```go
// resolutionVideoRatioMap 视频输入×分辨率四档折扣比率
var resolutionVideoRatioMap = map[string]map[string]float64{
    "doubao-seedance-2-0-260128": {
        "low_video":  28.0 / 46.0, // ≈0.6087
        "high_video": 31.0 / 46.0, // ≈0.6739
        "high":       51.0 / 46.0, // ≈1.1087
    },
    "doubao-seedance-2-0-fast-260128": {
        "low_video":  22.0 / 37.0, // ≈0.5946
        "high_video": 31.0 / 46.0, // ≈0.6739
        "high":       51.0 / 46.0, // ≈1.1087
    },
}
```

新增 `GetResolutionVideoRatio(modelName)` 函数查询比率映射。

### 2. EstimateBilling 重构 — `relay/channel/task/doubao/adaptor.go`

**核心流程：**

```
EstimateBilling
    │
    ├── Seedance 路径 (info.RelayMode == RelayModeSeedanceSubmit)
    │   └── estimateSeedanceBilling
    │       ├── req.HasVideo() → hasVideo
    │       ├── resolveResolutionTier(req.Resolution) → tierLow/tierHigh
    │       ├── tierKey(hasVideo, tier) → "low"/"high"/"low_video"/"high_video"
    │       └── 查 ratioMap，基准档不返回，其余返回 {"video_resolution_input": ratio}
    │
    └── 标准格式路径
        └── 检测 metadata 中 video_url → 有则返回 low_video 档
```

**辅助函数：**

| 函数 | 作用 |
|------|------|
| `resolveResolutionTier(resolution)` | 1080p → `tierHigh`，其余/未传 → `tierLow` |
| `tierKey(hasVideo, tier)` | 组合生成 `low`/`high`/`low_video`/`high_video` |

## 文件变更清单

| 文件 | 变更 |
|------|------|
| `relay/channel/task/doubao/constants.go` | 注释旧 `videoInputRatioMap`/`GetVideoInputRatio`，新增 `resolutionVideoRatioMap`/`GetResolutionVideoRatio` |
| `relay/channel/task/doubao/adaptor.go` | 注释旧 `EstimateBilling`，新增四档逻辑 + `estimateSeedanceBilling`/`resolveResolutionTier`/`tierKey` |

## 验证方式

1. `go build ./...` 编译通过
2. 请求 `resolution=480p`，无 `video_url` → 日志中 `OtherRatios` 为空（基准价）
3. 请求 `resolution=1080p`，无 `video_url` → 日志中 `video_resolution_input: 1.1087`
4. 请求 `resolution=480p`，有 `video_url` → 日志中 `video_resolution_input: 0.6087`
5. 请求 `resolution=1080p`，有 `video_url` → 日志中 `video_resolution_input: 0.6739`
6. 请求未传 `resolution` → 按基准档 `low` 处理（×1.0）
