# /s1/video/generations 接口传参与计费分析

> 分析日期: 2026-04-25
> 分析原因: 排查客户端调用报错（invalid_json 和 invalid resolution）

---

## 一、接口概览

| 维度 | 值 |
|---|---|
| 路由 | `POST /s1/video/generations` |
| 查询路由 | `GET /s1/video/generations/:task_id` |
| 鉴权 | Bearer Token（`middleware.TokenAuth()`） |
| 渠道分发 | `middleware.Distribute()` — 根据 model 字段选择渠道 |
| 控制器 | `controller.RelayTask` / `controller.RelayTaskFetch` |
| 协议类型 | Seedance 2.0 官方 API 格式（content 数组） |
| 上游渠道类型 | `DoubaoVideo` (54) 或 `VolcEngine` (45) |
| 上游地址 | `{channel_base_url}/api/v3/contents/generations/tasks` |

### 路由注册

```go
// router/video-router.go:36-42
videoS1Router := router.Group("/s1")
videoS1Router.Use(middleware.RouteTag("relay"))
videoS1Router.Use(middleware.TokenAuth(), middleware.Distribute())
{
    videoS1Router.POST("/video/generations", controller.RelayTask)
    videoS1Router.GET("/video/generations/:task_id", controller.RelayTaskFetch)
}
```

---

## 二、请求体结构（SeedanceSubmitReq）

定义在 `relay/common/relay_info.go:778-799`。

```json
{
  "model": "string",                           // 必填，模型名称
  "content": [                                 // 必填，数组不能为空
    {
      "type": "text",                          // 类型: text|image_url|video_url|audio_url|draft_task
      "text": "prompt文本",                    // type=text 时必填
      "image_url": {"url": "..."},            // type=image_url 时
      "video_url": {"url": "..."},            // type=video_url 时
      "audio_url": {"url": "..."},            // type=audio_url 时
      "draft_task": {"id": "..."},            // type=draft_task 时
      "role": "string"
    }
  ],
  "callback_url": "string",                   // 异步回调URL
  "return_last_frame": true/false,            // 是否返回最后一帧
  "service_tier": "string",                   // 服务层级 (default/flex)
  "execution_expires_after": 300,             // 执行超时(秒)
  "generate_audio": true/false,               // 是否生成音频
  "draft": true/false,                        // 是否草稿模式
  "tools": [{"type": "web_search"}],          // 工具列表
  "safety_identifier": "string",              // 安全标识符
  "resolution": "720p",                       // 分辨率 (480p/720p/1080p)
  "ratio": "16:9",                            // 宽高比
  "duration": 5,                              // 视频时长(秒)，整数
  "frames": 120,                              // 帧数
  "seed": 42,                                 // 随机种子
  "camera_fixed": true/false,                 // 是否固定相机
  "watermark": true/false,                    // 是否加水印
  "metadata": {"key": "value"}               // 元数据(支持字符串或对象)
}
```

### 校验规则（ValidateSeedanceTaskRequest）

定义在 `relay/common/relay_utils.go:226-262`：

1. `model` 必填且不能为空白字符串
2. `content` 数组必填且不能为空
3. `content` 中 `type` 只能是 `text`、`image_url`、`video_url`、`audio_url`、`draft_task`
4. `type=text` 时 `text` 不能为空
5. `content` 必须包含至少一个 `text` 类型的条目

### 已排查的两个常见错误

**错误 1：invalid_json**
```
"message": "json: cannot unmarshal string into Go struct field .Alias.content of type []common.SeedanceContentItem"
```
原因：`content` 被传成了 JSON 字符串 `"[{...}]"` 而非 JSON 数组 `[{}]`。

**错误 2：invalid resolution**
```
"message": "the parameter resolution specified in the request is not valid for model doubao-seedance-2-0"
```
原因：`resolution: "480"` 不是合法值，应为 `"480p"`、`"720p"` 或 `"1080p"`。这是上游字节跳动 API 返回的错误。

---

## 三、调用链路

```
POST /s1/video/generations
  → middleware.Distribute()
     - 识别路径，设置 RelayMode = RelayModeSeedanceSubmit
     - 从请求 body 读取 model 字段
     - 根据 model 选择匹配的渠道（DoubaoVideo/VolcEngine 类型）
  → controller.RelayTask(c, relayInfo)
     - 确定 platform → GetTaskAdaptor("54") → taskdoubao.TaskAdaptor{}
     - ValidateRequestAndSetAction → ValidateSeedanceTaskRequest
     - ModelPriceHelperPerCall → 计算预扣费
     - EstimateBilling → 检测是否含视频输入，应用折扣
     - PreConsumeBilling → 预扣费
     - BuildRequestBody → 透传 SeedanceSubmitReq 到上游
     - DoRequest → 发送请求到上游
     - DoResponse → 返回 task_id
     - SettleBilling → 结算
```

### 渠道选择逻辑

`GetTaskAdaptor` 根据渠道类型选择适配器（`relay/relay_adaptor.go:154`）：

```go
case constant.ChannelTypeDoubaoVideo, constant.ChannelTypeVolcEngine:
    return &taskdoubao.TaskAdaptor{}
```

- `ChannelTypeDoubaoVideo` = 54
- `ChannelTypeVolcEngine` = 45

---

## 四、计费规则详解

### 4.1 计费公式

```
预扣额度(quota) = 基础价格 × 分组倍率 × OtherRatios乘积
```

其中 `QuotaPerUnit = 500,000`（即 ¥0.002/1K tokens 的换算基准）。

### 4.2 基础价格确定（ModelPriceHelperPerCall）

定义在 `relay/helper/price.go:166-225`，优先级从高到低：

| 优先级 | 配置位置 | 计算公式 |
|---|---|---|
| 1 | `modelPrice` 后台配置 | `quota = modelPrice × 500000 × groupRatio` |
| 2 | `defaultModelPrice` 默认映射 | `quota = defaultPrice × 500000 × groupRatio` |
| 3 | `modelRatio` 倍率模式 | `quota = (modelRatio / 2) × 500000 × groupRatio` |
| 4 | 未配置 | 报错（除非 `AcceptUnsetRatioModel` 开启） |

- **按次计费**：走优先级 1/2 时，直接以固定价格扣费
- **按量计费**：走优先级 3 时，预扣费按倍率的一半计算

### 4.3 OtherRatios 倍率叠加（EstimateBilling）

Doubao Adaptor 的 `EstimateBilling`（`adaptor.go:140-162`）检测 content 中是否包含**视频输入**：

```go
if req.HasVideo() {   // content 中有 video_url 类型
    ratio = 28.0 / 46.0   // ≈ 0.6087  (doubao-seedance-2-0-260128)
    ratio = 22.0 / 37.0   // ≈ 0.5946  (doubao-seedance-2-0-fast-260128)
}
```

| 场景 | video_input OtherRatio | 说明 |
|---|---|---|
| 文生视频（纯 text） | 1.0 | 无折扣 |
| 视频生视频（含 video_url） | ≈0.6087 | 折扣后的价格 |

> 如果模型名在 `TaskPricePatches` 列表中（按次计费模型），则**不应用** OtherRatios。

### 4.4 分组倍率（GroupRatio）

| 场景 | groupRatio |
|---|---|
| 默认 | 1.0 |
| 使用 auto_group | 根据用户组配置的特殊倍率 |

### 4.5 预扣费机制

定义在 `relay/relay_task.go:144-211`：

1. **提交时预扣费**：首次请求成功返回 `task_id` 后，立即从用户账户预扣 `quota`
2. **失败退款**：如果请求失败（非 200），通过 `Billing.Refund()` 全额退还
3. **重试复用**：如果渠道支持重试，每次重试复用同一个 `info.Billing`，不会重复预扣
4. **免费模型**：如果 `modelPrice=0` 且 `groupRatio=0`，则 `freeModel=true`，不扣费

### 4.6 任务完成后的差额结算

通过轮询获取任务结果后（`service/task_polling.go:538-560`），有三个优先级：

| 优先级 | 条件 | 处理方式 |
|---|---|---|
| 1 | `AdjustBillingOnComplete` 返回 > 0 | 使用 adaptor 计算的额度（Doubao 返回 0，不触发） |
| 2 | 上游返回 `usage.total_tokens > 0` | 按 token 重算：`actualQuota = totalTokens × modelRatio × groupRatio × otherMultiplier` |
| 3 | 以上都不满足 | 保持预扣额度不变 |

**按次计费模式**（`PerCallBilling=true`）：任务完成后**不做差额结算**，预扣多少就是多少。

### 4.7 计费日志

消耗日志记录在 `service/task_billing.go:17-65`，包含：

| 日志字段 | 说明 |
|---|---|
| `model_price` | 模型固定价格 |
| `model_ratio` | 模型倍率 |
| `group_ratio` | 分组倍率 |
| `OtherRatios` | 额外倍率（如 `video_input: 0.61`） |
| `request_path` | `/s1/video/generations` |
| `is_task` | true |
| `upstream_model_name` | 模型映射后的上游模型名（如有映射） |

### 4.8 计费流程时序图

```
时间轴:
  T0  请求到达 → 渠道选择
  T1  ModelPriceHelperPerCall → 计算基础额度
  T2  EstimateBilling → 检测视频输入，添加 OtherRatio
  T3  合并 OtherRatio 到 quota
  T4  PreConsumeBilling → 预扣费（扣用户余额）
  T5  转发上游 → 返回 task_id
  T6  SettleBilling → 结算（记录消费）
  T7  插入 task 记录（含 BillingContext）

  轮询周期:
  T8  轮询上游获取任务结果
  T9  settleTaskBillingOnComplete → 差额结算
      - adaptor.AdjustBillingOnComplete (Doubao 返回 0)
      - totalTokens > 0 → RecalculateTaskQuotaByTokens
      - 都不满足 → 保持预扣额度
```

---

## 五、Doubao 模型列表

定义在 `relay/channel/task/doubao/constants.go`：

| 模型 ID | 视频输入折扣 |
|---|---|
| `doubao-seedance-1-0-pro-250528` | 无 |
| `doubao-seedance-1-0-lite-t2v` | 无 |
| `doubao-seedance-1-0-lite-i2v` | 无 |
| `doubao-seedance-1-5-pro-251215` | 无 |
| `doubao-seedance-2-0-260128` | 28.0/46.0 ≈ 0.6087 |
| `doubao-seedance-2-0-fast-260128` | 22.0/37.0 ≈ 0.5946 |

---

## 六、关键文件索引

| 文件 | 作用 |
|---|---|
| `router/video-router.go` | 路由注册（line 36-42） |
| `middleware/distributor.go` | 路径识别、渠道分发（line 265-280） |
| `relay/common/relay_info.go` | SeedanceSubmitReq 结构体定义（line 778-799） |
| `relay/common/relay_utils.go` | ValidateSeedanceTaskRequest 校验逻辑（line 226-262） |
| `relay/channel/task/doubao/adaptor.go` | Doubao 适配器：计费估算、请求转换、响应处理 |
| `relay/channel/task/doubao/constants.go` | 模型列表、视频输入折扣比率 |
| `relay/relay_task.go` | RelayTaskSubmit 主流程（line 144-240） |
| `relay/helper/price.go` | ModelPriceHelperPerCall 价格计算（line 166-225） |
| `service/billing.go` | PreConsumeBilling、SettleBilling |
| `service/task_billing.go` | LogTaskConsumption、差额结算 |
| `service/task_polling.go` | settleTaskBillingOnComplete 完成结算 |
