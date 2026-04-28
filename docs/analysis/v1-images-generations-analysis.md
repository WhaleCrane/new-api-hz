# `/v1/images/generations` 接口详细分析

## 1. 路由注册

**文件:** `router/relay-router.go:112-114`

```go
httpRouter.POST("/images/generations", func(c *gin.Context) {
    controller.Relay(c, types.RelayFormatOpenAIImage)
})
```

路由归属于 `/v1` 路由组，完整路径为 `/v1/images/generations`。该路由组在请求到达时依次经过以下中间件：

| 顺序 | 中间件 | 作用 |
|------|--------|------|
| 1 | `middleware.RouteTag("relay")` | 标记路由标签 |
| 2 | `middleware.SystemPerformanceCheck()` | 系统性能检查 |
| 3 | `middleware.TokenAuth()` | Token 认证 |
| 4 | `middleware.ModelRequestRateLimit()` | 模型请求限流 |
| 5 | `middleware.Distribute()` | 渠道选择（核心） |

最终交由 `controller.Relay` 统一处理，传入 `RelayFormatOpenAIImage` 格式标识。

---

## 2. 中间件层：Distribute（渠道分发）

**文件:** `middleware/distributor.go`

### 2.1 模型名解析

在 `getModelRequest()` 函数中（第 310-311 行），对 `/v1/images/generations` 路径有特殊处理：

```go
if strings.HasPrefix(c.Request.URL.Path, "/v1/images/generations") {
    modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "dall-e")
}
```

**关键行为：** 如果请求体 JSON 中没有 `model` 字段（为空字符串），系统会**自动填充默认值 `"dall-e"`**。这意味着：
- 客户端不传 `model` → 系统在 abilities 表中查找 `"dall-e"` 模型的渠道 → 找不到则报错
- 客户端传了 `model`（如 `"doubao-seedream-4-0-250828"`）→ 使用该值进行渠道查找

### 2.2 渠道匹配

系统通过 `service.CacheGetRandomSatisfiedChannel()` 查询 `abilities` 表，寻找同时满足以下条件的渠道：
- 在用户的分组（group）下
- 启用了请求中指定的模型
- 渠道状态为 `enabled`

如果找不到匹配渠道，返回错误：
```
No available channel for model dall-e under group default (distributor)
```

---

## 3. Controller 层：Relay（中继处理）

**文件:** `controller/relay.go:67`

`Relay` 函数是整个中继系统的入口，处理流程如下：

### 3.1 请求解析与验证

```go
request, err := helper.GetAndValidateRequest(c, relayFormat)
```

调用 `helper.GetAndValidOpenAIImageRequest()` 进行验证（详见第 5 节）。

### 3.2 生成 RelayInfo

```go
relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, request, ws)
```

将请求信息、渠道信息、模型信息等聚合到 `RelayInfo` 结构体，后续所有处理都依赖此对象。

### 3.3 敏感词检查 & Token 估算

```go
if needSensitiveCheck && meta != nil {
    contains, words := service.CheckSensitiveText(meta.CombineText)
}
tokens, err := service.EstimateRequestToken(c, meta, relayInfo)
```

### 3.4 计费预扣

```go
priceData, err := helper.ModelPriceHelper(c, relayInfo, tokens, meta)
newAPIError = service.PreConsumeBilling(c, priceData.QuotaToPreConsume, relayInfo)
```

根据模型价格计算预扣费金额，从用户余额中扣除。

### 3.5 中继执行（支持重试）

```go
for ; retryParam.GetRetry() <= common.RetryTimes; retryParam.IncreaseRetry() {
    channel, channelErr := getChannel(c, relayInfo, retryParam)
    // ...
    newAPIError = relayHandler(c, relayInfo)
}
```

- 支持多次重试（默认 `common.RetryTimes` 次）
- 失败时根据状态码判断是否可重试
- 重试时可能切换不同渠道

### 3.6 错误处理与配额返还

```go
defer func() {
    if newAPIError != nil {
        if relayInfo.Billing != nil {
            relayInfo.Billing.Refund(c)
        }
    }
}()
```

下游失败时自动返还预扣配额。

---

## 4. relayHandler 分发

**文件:** `controller/relay.go:34-55`

```go
func relayHandler(c *gin.Context, info *relaycommon.RelayInfo) *types.NewAPIError {
    switch info.RelayMode {
    case relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits:
        err = relay.ImageHelper(c, info)
    // ... 其他分支
    }
}
```

`RelayModeImagesGenerations` 对应 `/v1/images/generations` 路径，由 `relay.ImageHelper` 处理。

---

## 5. 请求验证：GetAndValidOpenAIImageRequest

**文件:** `relay/helper/valid_request.go:142-228`

验证逻辑：

1. **解析请求体** → `dto.ImageRequest`
2. **model 为空** → 返回错误 `"model is required"`
3. **size 参数含 `×`（乘号）** → 返回错误，要求用 `x` 代替
4. **模型特定的 size 校验**：
   - `dall-e-2` / `dall-e`：仅允许 `256x256`、`512x512`、`1024x1024`
   - `dall-e-3`：仅允许 `1024x1024`、`1024x1792`、`1792x1024`
   - 其他模型（如 seedream）：**不校验 size**
5. **N 默认为 1**：`imageRequest.N = common.GetPointer(uint(1))`

注意：验证函数本身不设置默认 model，默认 `"dall-e"` 的填充发生在 `Distribute` 中间件阶段。如果 `Distribute` 没设置，这里会报 `"model is required"`。

---

## 6. ImageHelper 核心处理

**文件:** `relay/image_handler.go:23-157`

### 6.1 模型映射

```go
err = helper.ModelMappedHelper(c, info, request)
```

如果渠道配置了模型映射（如将 `doubao-seedream-4-0-250828` 映射为其他名称），在此阶段应用。

### 6.2 Adaptor 选择

```go
adaptor := GetAdaptor(info.ApiType)
```

根据渠道类型选择对应的适配器。例如 `ChannelTypeVolcEngine` (45) → `volcengine.Adaptor`。

### 6.3 请求体构建

```go
if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
    // 透模式：直接使用原始请求体
    requestBody = common.readerOnly(storage)
} else {
    // 转换模式：通过 adaptor 转换
    convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
    jsonData, err := common.Marshal(convertedRequest)
    // 应用 ParamOverride
    jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
    requestBody = bytes.NewBuffer(jsonData)
}
```

两种模式：
- **透传模式**：直接转发原始请求体，不做任何修改
- **转换模式**：通过渠道适配器转换为上游格式

### 6.4 发起请求

```go
resp, err := adaptor.DoRequest(c, info, requestBody)
```

### 6.5 响应处理

```go
usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
```

VolcEngine 等大多数渠道的 `DoResponse` 内部委托给 `openai.Adaptor` 处理。

### 6.6 计费结算

```go
service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), logContent)
```

记录消费日志，更新用户配额。

---

## 7. VolcEngine 渠道适配器（以 Doubao Seedream 为例）

**文件:** `relay/channel/volcengine/adaptor.go`

### 7.1 ConvertImageRequest

```go
func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
    switch info.RelayMode {
    case constant.RelayModeImagesGenerations:
        return request, nil  // 直接透传，不做转换
    }
}
```

`doubao-seedream-4-0-250828` 的请求体不做转换，直接转发。

### 7.2 GetRequestURL

```go
case constant.RelayModeImagesGenerations, constant.RelayModeImagesEdits:
    return fmt.Sprintf("%s/api/v3/images/generations", baseUrl), nil
```

URL 映射为 `https://{baseUrl}/api/v3/images/generations`。

### 7.3 SetupRequestHeader

```go
req.Set("Authorization", "Bearer "+info.ApiKey)
req.Set("Content-Type", gin.MIMEJSON)
```

设置 Bearer Token 认证。

### 7.4 DoResponse

```go
adaptor := openai.Adaptor{}
usage, err = adaptor.DoResponse(c, resp, info)
```

委托给 OpenAI 适配器处理响应，因为 VolcEngine 的图像生成接口返回格式与 OpenAI 兼容。

---

## 8. 数据模型：ImageRequest

**文件:** `dto/openai_image.go`（推断）

关键字段：
| 字段 | 类型 | 说明 |
|------|------|------|
| `Model` | `string` | 模型名称 |
| `Prompt` | `string` | 提示词 |
| `N` | `*uint` | 生成图片数量 |
| `Quality` | `string` | 品质（`standard` / `hd`） |
| `Size` | `string` | 图片尺寸（如 `1024x1024`） |
| `ResponseFormat` | `string` | 响应格式（`url` / `b64_json`） |

---

## 9. 完整请求生命周期

```
客户端请求 POST /v1/images/generations
    │
    ▼
┌─────────────────────────────────┐
│ middleware.TokenAuth()          │  ← 验证 Token
└────────────┬────────────────────┘
             ▼
┌─────────────────────────────────┐
│ middleware.Distribute()         │  ← 渠道选择
│   1. 解析请求体获取 model       │
│   2. model 为空 → 默认 "dall-e" │
│   3. 查询 abilities 表找渠道     │
│   4. 设置渠道上下文             │
└────────────┬────────────────────┘
             ▼
┌─────────────────────────────────┐
│ controller.Relay()              │  ← 中继入口
│   1. GetAndValidateRequest()    │     验证请求
│   2. GenRelayInfo()             │     构建中继信息
│   3. CheckSensitiveText()       │     敏感词检查
│   4. EstimateToken()            │     估算 Token
│   5. PreConsumeBilling()        │     预扣费
└────────────┬────────────────────┘
             ▼
┌─────────────────────────────────┐
│ relayHandler()                  │
│   → relay.ImageHelper()         │  ← 图像中继
└────────────┬────────────────────┘
             ▼
┌─────────────────────────────────┐
│ ImageHelper()                   │
│   1. ModelMappedHelper()        │     模型映射
│   2. GetAdaptor()               │     选择适配器
│   3. ConvertImageRequest()      │     请求转换
│   4. DoRequest()                │     发起上游请求
│   5. DoResponse()               │     处理响应
│   6. PostTextConsumeQuota()     │     计费结算
└────────────┬────────────────────┘
             ▼
┌─────────────────────────────────┐
│ 返回 OpenAI 兼容的 JSON 响应     │
│ {"data": [{"url": "..."}]}      │
└─────────────────────────────────┘
```

---

## 10. 已支持的图像生成模型

**文件:** `common/model.go:12-19`

```go
ImageGenerationModels = []string{
    "dall-e-3",
    "dall-e-2",
    "gpt-image-1",
    "prefix:imagen-",
    "flux-",
    "flux.1-",
}
```

这是**信息性列表**（用于 `IsImageGenerationModel()` 判断），不影响实际路由。实际能否使用取决于渠道的 `abilities` 表中是否配置了该模型。

**VolcEngine 渠道已注册的模型：**
- `doubao-seedream-4-0-250828`
- `seedream-4-0-250828`

---

## 11. 常见问题排查

### 11.1 "No available channel for model dall-e"

**原因：** 请求体中没有 `model` 字段，系统默认填充了 `"dall-e"`，但 abilities 表中没有 `dall-e` 的渠道。

**解决：** 确保请求体中包含 `"model": "doubao-seedream-4-0-250828"`。

### 11.2 绘图日志不记录

当前绘图日志（`MjLogs` 表）**仅针对 Midjourney 渠道**设计，走 `/v1/images/generations` 的请求不会被记录到该表中，而是走正常的消费日志（`Log` 表）。

### 11.3 请求体太大

如果启用了请求体存储且请求体超过限制，会返回 413 错误。可通过 `PassThroughBodyEnabled` 开启透传模式避免。
