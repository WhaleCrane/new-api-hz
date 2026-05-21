# /s1/video/generations 接口定义过程

> 创建日期: 2026-04-25
> 触发原因: 现有 `/v1/video/generations` 使用简化统一格式 `{prompt, images[], metadata:{}}`，
> Doubao adaptor 的 `convertToRequestPayload` 仅支持文本+图片，无法利用 Seedance 2.0
> 官方完整的 `content` 数组能力（参考视频、参考音频、role 首帧/尾帧区分、样片模式等）。

---

## 一、设计目标

新增 `/s1/video/generations` 接口，直接接受 Seedance 2.0 官方 API 格式：
- 复用现有中间件链、控制器、适配器、计费与持久化基础设施
- 请求体直接透传到上游，无需格式转换
- 现有 `/v1/video/generations` 接口完全不受影响

---

## 二、架构对比

### 现有 `/v1/video/generations` 请求格式

```json
{
  "model": "doubao-seedance-2-0-260128",
  "prompt": "一只猫在草地上跑",
  "images": ["https://example.com/cat.jpg"],
  "metadata": {"resolution": "720p", "ratio": "16:9", "duration": 5}
}
```

**转换逻辑**：Doubao adaptor 的 `convertToRequestPayload` 将上述格式拼接为官方格式，
但仅处理 `prompt`（转为 text）和 `images`（转为 image_url），**不处理 role、video_url、audio_url 等**。

### 新 `/s1/video/generations` 请求格式

```json
{
  "model": "doubao-seedance-2-0-260128",
  "content": [
    {"type": "text", "text": "一只猫在草地上跑"},
    {"type": "image_url", "image_url": {"url": "https://example.com/cat.jpg"}, "role": "first_frame"},
    {"type": "video_url", "video_url": {"url": "https://example.com/ref.mp4"}, "role": "reference_video"}
  ],
  "resolution": "720p",
  "ratio": "16:9",
  "duration": 5,
  "generate_audio": true
}
```

**无转换逻辑**：`buildSeedanceRequestBody` 直接将请求体序列化后发送到上游。

---

## 三、实现步骤

### Step 1: 新增 relay mode 常量

**文件:** `relay/constant/relay_mode.go`

```go
RelayModeSeedanceSubmit     // /s1/video/generations POST
RelayModeSeedanceFetchByID  // /s1/video/generations/:task_id GET
```

这两个常量用于标识请求来自 `/s1/` 路径，Doubao adaptor 据此选择对应的验证和构建逻辑。

### Step 2: 注册路由

**文件:** `router/video-router.go`

在现有 `videoV1Router` 之后新增 `/s1` 路由组：

```go
videoS1Router := router.Group("/s1")
videoS1Router.Use(middleware.RouteTag("relay"))
videoS1Router.Use(middleware.TokenAuth(), middleware.Distribute())
{
    videoS1Router.POST("/video/generations", controller.RelayTask)
    videoS1Router.GET("/video/generations/:task_id", controller.RelayTaskFetch)
}
```

复用 `controller.RelayTask` 和 `controller.RelayTaskFetch`，路由区分由 relay mode 控制。

### Step 3: Distributor 识别 `/s1/` 路径

**文件:** `middleware/distributor.go`

在 `getModelRequest()` 中，在 `/v1/video/generations` 分支之后添加 `/s1/video/generations` 分支：
- POST → 提取 model，设置 `RelayModeSeedanceSubmit`
- GET → 设置 `RelayModeSeedanceFetchByID`，不选渠道

逻辑与 `/v1/` 完全一致，仅 relay mode 不同。

### Step 4: 定义 SeedanceSubmitReq 结构体

**文件:** `relay/common/relay_info.go`

新增三个结构体：

| 结构体 | 作用 |
|---|---|
| `SeedanceMediaURL` | 封装 `image_url`/`video_url`/`audio_url` 中的 `{url: "..."}` 对象 |
| `SeedanceContentItem` | 对应官方 `content` 数组中的每一项，含 `type`、`text`、`image_url`、`video_url`、`audio_url`、`role`、`draft_task` |
| `SeedanceSubmitReq` | 完整请求体，含 `model`、`content[]`、`callback_url`、`resolution`、`ratio`、`duration`、`seed`、`generate_audio`、`draft`、`tools`、`safety_identifier`、`watermark` 等全部官方字段 |

附加方法：
- `GetPrompt()` — 从 content 数组中提取第一个 text 类型的文本
- `HasVideo()` — 检查 content 数组中是否包含 video_url 类型（用于计费折扣检测）
- `UnmarshalJSON()` — 支持 metadata 字段的字符串/对象双格式

### Step 5: 新增验证函数

**文件:** `relay/common/relay_utils.go`

| 函数 | 作用 |
|---|---|
| `ValidateSeedanceTaskRequest` | 校验 model 非空、content 数组非空、至少含一个 text 类型条目、type 合法；校验通过后存入 `c.Set("seedance_request", req)` |
| `GetSeedanceTaskRequest` | 从 gin context 中获取 `SeedanceSubmitReq` |

与 `ValidateBasicTaskRequest` 的区别：
- 不要求 `prompt` 字段（文本在 content 数组中）
- 不处理 multipart/form-data（官方 API 仅接受 JSON）
- 不处理 `image` 单图兼容（图片在 content 数组中）

### Step 6: 修改 Doubao TaskAdaptor

**文件:** `relay/channel/task/doubao/adaptor.go`

三个方法按 `info.RelayMode` 分支：

| 方法 | RelayModeSeedanceSubmit 行为 | 其他行为 |
|---|---|---|
| `ValidateRequestAndSetAction` | 调用 `ValidateSeedanceTaskRequest` | 调用 `ValidateBasicTaskRequest` |
| `EstimateBilling` | 调用 `GetSeedanceTaskRequest` + `req.HasVideo()` 检测 | 调用 `GetTaskRequest` + `hasVideoInMetadata` 检测 |
| `BuildRequestBody` | 调用 `buildSeedanceRequestBody`（直接序列化） | 调用 `buildStandardRequestBody`（转换后序列化） |

新增两个方法：
- `buildSeedanceRequestBody` — 直接序列化 `SeedanceSubmitReq`，应用 model mapping
- `buildStandardRequestBody` — 原有逻辑（从 `BuildRequestBody` 重命名提取）

### Step 7: Fetch 端口注册

**文件:** `relay/relay_task.go`

在 `fetchRespBuilders` map 中注册：

```go
relayconstant.RelayModeSeedanceFetchByID: videoFetchByIDRespBodyBuilder,
```

复用 `videoFetchByIDRespBodyBuilder`，该 builder 通过 `c.Param("task_id")` 获取任务 ID 并查询。

---

## 四、文件变更清单

| # | 文件 | 变更类型 | 说明 |
|---|------|------|---|
| 1 | `relay/constant/relay_mode.go` | 修改 | 新增 2 个 relay mode 常量 |
| 2 | `router/video-router.go` | 修改 | 新增 `/s1` 路由组（POST + GET） |
| 3 | `middleware/distributor.go` | 修改 | 新增 `/s1/video/generations` 路径识别 |
| 4 | `relay/common/relay_info.go` | 修改 | 新增 3 个结构体 + 3 个方法 |
| 5 | `relay/common/relay_utils.go` | 修改 | 新增 2 个函数 |
| 6 | `relay/channel/task/doubao/adaptor.go` | 修改 | 3 个方法分支 + 2 个新方法 + 1 个 import |
| 7 | `relay/relay_task.go` | 修改 | fetchRespBuilders map 新增注册 |

---

## 五、渠道兼容性分析

### 所有 Task 渠道列表

| 渠道类型 | 渠道 ID | 对应模型 | 支持 `/s1/` 格式 |
|---|---|---|---|
| ChannelTypeDoubaoVideo | 54 | doubao-seedance-* | 是（RelayModeSeedanceSubmit 分支生效） |
| ChannelTypeVolcEngine | 45 | doubao-seedance-* | 是（同上） |
| ChannelTypeKling | 50 | kling-v* | 否（仍调用 ValidateBasicTaskRequest，要求 prompt 字段） |
| ChannelTypeJimeng | 51 | jimeng_vgfm_* | 否（同上） |
| ChannelTypeSora/OpenAI | 55/1 | sora-2 | 否（同上） |
| ChannelTypeGemini | 24 | video-* | 否（同上） |
| ChannelTypeVidu | 52 | vidu-* | 否（同上） |
| ChannelTypeMiniMax | 35 | hailuo-* | 否（同上） |
| ChannelTypeAli | 17 | wanxiang-* | 否（同上） |
| ChannelTypeVertexAi | 13 | video-* | 否（同上） |

### 结论

- **渠道选择机制相同**：两个接口都通过 `model` 字段 + 渠道模型映射来选择渠道
- **仅 Doubao 支持**：目前仅 Doubao adaptor 识别了 `RelayModeSeedanceSubmit` 并做分支处理
- **其他渠道行为**：如果通过 `/s1/` 调用非 Doubao 模型，请求会走到对应 adaptor 的标准路径，但该路径要求 `prompt` 字段，而 `/s1/` 的请求格式不含 `prompt`，**会报错** `prompt is required`
- **建议使用方式**：`/s1/` 应仅用于 DoubaoVideo (54) 或 VolcEngine (45) 渠道

---

## 六、验证方式

1. **编译检查**: `go build ./...` — 已通过
2. **回归验证**: 确认 `/v1/video/generations` 功能不受影响
3. **接口测试**:
   - POST `/s1/video/generations` 发送官方 content 数组格式
   - 确认请求正确转发到上游 Doubao API
   - GET `/s1/video/generations/:task_id` 查询任务状态
