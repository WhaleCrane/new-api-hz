# Seedance 2.0 官方接口 vs /v1/video/generations 对比分析

> 分析日期: 2026-04-25
> 官方文档来源: F:\seedance2.0 文档.md
> 官方接口: `POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks`

---

## 一、接口概览

| 维度 | Seedance 2.0 官方 | `/v1/video/generations` |
|---|---|---|
| 请求地址 | `POST ark.cn-beijing.volces.com/api/v3/contents/generations/tasks` | `POST /v1/video/generations` |
| 鉴权 | 仅 API Key | JWT Token / Session Auth |
| 定位 | 单上游直调（火山方舟） | 统一网关（代理多平台：Doubao/可灵/即梦/Sora/Vidu/海螺/阿里/Gemini） |
| 接口性质 | 异步任务（返回 task_id，需轮询查询） | 异步任务（返回 task_id，需轮询查询） |
| 请求体格式 | JSON | JSON 或 multipart/form-data |

## 二、架构差异

```
官方接口:
  client -> content: [{type, url, role}] -> 火山方舟

本项目:
  client -> {prompt, images[], metadata:{}}
         -> middleware/distributor.go 解析 model，设置 relay_mode
         -> controller.RelayTask 调度到 Doubao adaptor
         -> doubao/adaptor.go:convertToRequestPayload() 组装为官方格式
         -> 火山方舟 /api/v3/contents/generations/tasks
```

**关键转换逻辑** (`doubao/adaptor.go:convertToRequestPayload`, line 270-304):

```go
// 当前实现：仅处理图片和文本，且所有图片统一作为 image_url 处理，无 role 区分
r.Content = append(r.Content, ContentItem{Type: "image_url", ImageURL: &MediaURL{URL: imgURL}})
// 缺少 role 字段，无法区分 first_frame / last_frame / reference_image

r.Content = append(r.Content, ContentItem{Type: "text", Text: req.Prompt})
// 仅处理 prompt 和 images，未处理 video_url / audio_url / draft_task
```

**Metadata 透传机制** (`taskcommon.UnmarshalMetadata`):
- `metadata` 中的键值对通过 `UnmarshalMetadata` 反序列化到 `requestPayload`
- 支持 `resolution`, `ratio`, `seed`, `camera_fixed`, `watermark`, `generate_audio`, `return_last_frame`, `service_tier`, `execution_expires_after`, `draft`, `tools`, `safety_identifier` 等字段
- **缺陷**：弱校验，拼写错误会被忽略

## 三、模型能力对比

### 3.1 官方模型支持的场景

| 模型 | 多模态参考生视频 | 首尾帧 | 首帧 | 文生视频 | 有声视频 | 样片模式 |
|---|---|---|---|---|---|---|
| seedance 2.0 | 0~9图+0~3视频+0~3音频 | 支持 | 支持 | 支持 | 支持 | - |
| seedance 2.0 fast | 0~9图+0~3视频+0~3音频 | 支持 | 支持 | 支持 | 支持 | - |
| seedance 1.5 pro | - | 支持 | 支持 | 支持 | 支持 | 支持(draft) |
| seedance 1.0 pro | - | 支持 | 支持 | 支持 | - | - |
| seedance 1.0 pro fast | - | - | 支持 | 支持 | - | - |
| seedance 1.0 lite-t2v | - | - | - | 支持 | - | - |
| seedance 1.0 lite-i2v | 1~4图+文本 | 支持 | 支持 | - | - | - |

### 3.2 本项目模型注册情况

| Model ID | 本项目已注册 | 官方能力 | 本项目实际支持 | 差距 |
|---|---|---|---|---|
| `doubao-seedance-2-0-260128` | 是 | 多模态(图0-9/视频0-3/音频0-3) | **仅文本+图片** | 缺视频/音频/role |
| `doubao-seedance-2-0-fast-260128` | 是 | 多模态(图0-9/视频0-3/音频0-3) | **仅文本+图片** | 缺视频/音频/role |
| `doubao-seedance-1-5-pro-251215` | 是 | 首帧/首尾帧/文生视频/draft | **仅文本+图片** | 缺首尾帧区分/draft |
| `doubao-seedance-1-0-pro-250528` | 是 | 首帧/首尾帧/文生视频 | **仅文本+图片** | 缺首尾帧区分 |
| `doubao-seedance-1-0-lite-t2v` | 是 | 文生视频 | 文本 | 完全支持 |
| `doubao-seedance-1-0-lite-i2v` | 是 | 首帧/首尾帧/参考图(1-4) | **仅图片(无role)** | 缺role区分 |

## 四、请求参数详细对比

### 4.1 顶层参数

| 官方参数 | 类型 | 必填 | 默认值 | 本项目对应 | 状态 | 备注 |
|---|---|---|---|---|---|---|
| `model` | string | 是 | - | `TaskSubmitReq.Model` | 已支持 | - |
| `content` | object[] | 是 | - | `Prompt`+`Images`+`Metadata` | **部分支持** | 核心差异，详见 4.2 |
| `callback_url` | string | 否 | - | **无** | **缺失** | 任务状态回调通知 |
| `return_last_frame` | boolean | 否 | false | `metadata` 透传 | 已支持 | 返回尾帧图像 |
| `service_tier` | string | 否 | default | `metadata` 透传 | 已支持 | default/flex(离线) |
| `execution_expires_after` | integer | 否 | 172800 | `metadata` 透传 | 已支持 | 超时阈值(秒) |
| `generate_audio` | boolean | 否 | true | `metadata` 透传 | 已支持 | 仅2.0/1.5 pro支持 |
| `draft` | boolean | 否 | false | `metadata` 透传 | 已支持 | 仅1.5 pro支持 |
| `tools` | object[] | 否 | - | `metadata` 透传 | 已支持 | 仅2.0/2.0 fast支持 |
| `safety_identifier` | string | 否 | - | `metadata` 透传 | 已支持 | 用户标识符 |
| `resolution` | string | 否 | 720p/1080p | `metadata` 透传 | 已支持 | 480p/720p/1080p |
| `ratio` | string | 否 | adaptive/16:9 | `metadata` 透传 | 已支持 | 详见 4.4 |
| `duration` | integer | 否 | 5 | `TaskSubmitReq.Duration`/`Seconds` | 已支持 | 详见 4.5 |
| `frames` | integer | 否 | - | `metadata` 透传 | 已支持 | 2.0/1.5 pro暂不支持 |
| `seed` | integer | 否 | -1 | `metadata` 透传 | 已支持 | [-1, 2^32-1] |
| `camera_fixed` | boolean | 否 | false | `metadata` 透传 | 已支持 | 参考图场景不支持 |
| `watermark` | boolean | 否 | false | `metadata` 透传 | 已支持 | - |

### 4.2 content 数组详细对比

官方 `content` 数组支持 5 种内容类型，具体如下：

#### 4.2.1 文本信息 (`type: "text"`)

| 官方字段 | 类型 | 必填 | 本项目映射 | 状态 |
|---|---|---|---|---|
| `type` | string | 是 | 硬编码为 `"text"` | 已支持 |
| `text` | string | 是 | `TaskSubmitReq.Prompt` | 已支持 |

**官方约束**：
- 语言：所有模型支持中英文；2.0/2.0 fast 额外支持日语、印尼语、西班牙语、葡萄牙语
- 字数：中文不超过500字，英文不超过1000词

#### 4.2.2 图片信息 (`type: "image_url"`)

| 官方字段 | 类型 | 必填 | 本项目映射 | 状态 |
|---|---|---|---|---|
| `type` | string | 是 | 硬编码为 `"image_url"` | 已支持 |
| `image_url.url` | string | 是 | `TaskSubmitReq.Images[]` | 已支持 |
| `role` | string | 条件必填 | **无对应字段** | **缺失** |

**官方图片约束**：
- 格式：jpeg、png、webp、bmp、tiff、gif；1.5 pro 额外支持 heic/heif
- 宽高比：(0.4, 2.5)
- 宽高长度：(300, 6000) px
- 大小：< 30 MB
- URL 支持：公网 URL / Base64 编码 / 素材 ID (`asset://<ASSET_ID>`)

**图片数量限制**：

| 场景 | 数量限制 | 本项目支持 |
|---|---|---|
| 首帧生视频 | 1 张 | 支持（但无 role 区分） |
| 首尾帧生视频 | 2 张 | **不支持**（无法指定 first_frame/last_frame） |
| 2.0/2.0 fast 多模态参考 | 1~9 张 | **部分支持**（无 role，无法指定 reference_image） |
| 1.0 lite 参考图 | 1~4 张 | **部分支持**（无 role） |

#### 4.2.3 视频信息 (`type: "video_url"`)

| 官方字段 | 类型 | 必填 | 本项目映射 | 状态 |
|---|---|---|---|---|
| `type` | string | 是 | **无** | **缺失** |
| `video_url.url` | string | 是 | **无** | **缺失** |
| `role` | string | 条件必填 | **无** | **缺失**（仅支持 reference_video） |

**官方视频约束**（仅 2.0/2.0 fast 支持）：
- 格式：mp4、mov
- 分辨率：480p, 720p, 1080p
- 时长：单个视频 [2, 15] s，最多 3 个参考视频，总时长不超过 15 s
- 宽高比：[0.4, 2.5]
- 宽高长度：[300, 6000] px
- 总像素数：[409600, 2086876]
- 大小：< 50 MB
- 帧率：[24, 60]
- URL 支持：公网 URL / 素材 ID (`asset://<ASSET_ID>`)，**不支持 Base64**

**人脸信任机制**：平台信任 2.0/2.0 fast 模型生成的含人脸视频（本账号近30天内），可作为输入素材二次创作。

#### 4.2.4 音频信息 (`type: "audio_url"`)

| 官方字段 | 类型 | 必填 | 本项目映射 | 状态 |
|---|---|---|---|---|
| `type` | string | 是 | **无** | **缺失** |
| `audio_url.url` | string | 是 | **无** | **缺失** |
| `role` | string | 条件必填 | **无** | **缺失**（仅支持 reference_audio） |

**官方音频约束**（仅 2.0/2.0 fast 支持）：
- 格式：wav、mp3
- 时长：单个音频 [2, 15] s，最多 3 段参考音频，总时长不超过 15 s
- 大小：< 15 MB
- URL 支持：公网 URL / Base64 编码 / 素材 ID (`asset://<ASSET_ID>`)
- **不可单独输入音频**，至少包含 1 个参考视频或图片
- 生成的有声视频均为**单声道**，与传入音频声道数无关

#### 4.2.5 样片信息 (`type: "draft_task"`)

| 官方字段 | 类型 | 必填 | 本项目映射 | 状态 |
|---|---|---|---|---|
| `type` | string | 是 | **无** | **缺失** |
| `draft_task.id` | string | 是 | **无** | **缺失** |

**仅 1.5 pro 支持**：基于样片任务 ID 生成正式视频，复用 Draft 视频的用户输入（model, content.text, content.image_url, generate_audio, seed, ratio, duration, camera_fixed）。

### 4.3 role 字段详细对比

官方文档明确说明 **3 种互斥场景，不可混用**：

| 场景 | 支持模型 | 官方 role 取值 | 图片数量 | 本项目支持 |
|---|---|---|---|---|
| 图生视频-首帧 | 所有图生视频模型 | `first_frame` 或不填 | 1 张 | **不支持 role** |
| 图生视频-首尾帧 | 2.0/2.0 fast, 1.5 pro, 1.0 pro, 1.0 lite i2v | 首帧=`first_frame`，尾帧=`last_frame` | 2 张 | **完全不支持** |
| 多模态参考生视频 | 2.0/2.0 fast(1-9), 1.0 lite i2v(1-4) | `reference_image` | 1~9 / 1~4 张 | **不支持 role** |
| 参考视频 | 2.0/2.0 fast | `reference_video` | 1~3 个 | **完全不支持** |
| 参考音频 | 2.0/2.0 fast | `reference_audio` | 1~3 段 | **完全不支持** |

**官方重要提示**：
- 多模态参考生视频可通过提示词指定参考图片作为首帧/尾帧，间接实现"首尾帧+多模态参考"效果
- 若需严格保障首尾帧和指定图片一致，**优先使用图生视频-首尾帧**（配置 role 为 first_frame/last_frame）
- 参考图生视频的提示词可用自然语言指定多张图组合，推荐用 `[图1]xxx，[图2]xxx` 方式指定

### 4.4 ratio（宽高比）参数详细对比

| ratio 值 | 480p 宽高像素 | 720p 宽高像素 | 1080p 宽高像素 | 本项目支持 |
|---|---|---|---|---|
| 16:9 | 864×480 (2.0 fast: 864×496) | 1248×704 (2.0: 1280×720) | 1920×1088 (2.0: 1920×1080) | metadata 透传 |
| 4:3 | 736×544 (2.0: 752×560) | 1120×832 (2.0: 1112×834) | 1664×1248 | metadata 透传 |
| 1:1 | 640×640 | 960×960 | 1440×1440 | metadata 透传 |
| 3:4 | 544×736 (2.0: 560×752) | 832×1120 (2.0: 834×1112) | 1248×1664 | metadata 透传 |
| 9:16 | 480×864 (2.0: 496×864) | 704×1248 (2.0: 720×1280) | 1088×1920 (2.0: 1080×1920) | metadata 透传 |
| 21:9 | 960×416 (2.0: 992×432) | 1504×640 (2.0: 1470×630) | 2176×928 (2.0: 2206×946) | metadata 透传 |
| adaptive | 根据场景自动选择 | 根据场景自动选择 | 根据场景自动选择 | metadata 透传 |

**默认值差异**：
- 2.0/2.0 fast、1.5 pro：默认 `adaptive`
- 1.0 lite 参考图场景：默认 `16:9`
- 其他模型：文生视频默认 `16:9`，图生视频默认 `adaptive`

**adaptive 适配规则**：
- 文生视频：根据提示词智能选择
- 首帧/首尾帧：根据首帧图片比例自动选择最接近值
- 多模态参考：根据提示词意图判断，以首帧图片/视频为准

**限制**：
- 1080p：1.0 lite 参考图场景不支持，2.0 fast 不支持

### 4.5 duration（时长）参数详细对比

| 模型 | 取值范围 | 特殊值 | 本项目支持 |
|---|---|---|---|
| 1.0 pro / pro fast / lite | [2, 12] s | - | 已支持 |
| 1.5 pro | [4, 12] 或 `-1` | `-1`=模型智能选择 | 已支持（整数） |
| 2.0 / 2.0 fast | [4, 15] 或 `-1` | `-1`=模型智能选择 | 已支持（整数） |

**frames 支持情况**：2.0/2.0 fast、1.5 pro **暂不支持** frames 参数。

### 4.6 参数传入方式对比

官方支持两种参数传入方式：

| 方式 | 示例 | 校验强度 | 本项目支持 |
|---|---|---|---|
| 新方式（推荐） | Request Body 中直接传 `resolution`, `ratio` 等 | **强校验**，参数错误会报错 | metadata 透传，**弱校验** |
| 旧方式 | 文本提示词后追加 `--rs 720p --rt 16:9` 等 | **弱校验**，参数错误被忽略 | 不支持 |

## 五、本项目数据结构

### 5.1 Doubao Adaptor 请求结构体（已定义完整字段）

```go
// relay/channel/task/doubao/adaptor.go
type requestPayload struct {
    Model                 string         `json:"model"`
    Content               []ContentItem  `json:"content,omitempty"`
    CallbackURL           string         `json:"callback_url,omitempty"`
    ReturnLastFrame       *dto.BoolValue `json:"return_last_frame,omitempty"`
    ServiceTier           string         `json:"service_tier,omitempty"`
    ExecutionExpiresAfter *dto.IntValue  `json:"execution_expires_after,omitempty"`
    GenerateAudio         *dto.BoolValue `json:"generate_audio,omitempty"`
    Draft                 *dto.BoolValue `json:"draft,omitempty"`
    Tools                 []struct { Type string `json:"type,omitempty"` } `json:"tools,omitempty"`
    Resolution            string         `json:"resolution,omitempty"`
    Ratio                 string         `json:"ratio,omitempty"`
    Duration              *dto.IntValue  `json:"duration,omitempty"`
    Frames                *dto.IntValue  `json:"frames,omitempty"`
    Seed                  *dto.IntValue  `json:"seed,omitempty"`
    CameraFixed           *dto.BoolValue `json:"camera_fixed,omitempty"`
    Watermark             *dto.BoolValue `json:"watermark,omitempty"`
}

type ContentItem struct {
    Type     string    `json:"type,omitempty"`
    Text     string    `json:"text,omitempty"`
    ImageURL *MediaURL `json:"image_url,omitempty"`
    VideoURL *MediaURL `json:"video_url,omitempty"`
    AudioURL *MediaURL `json:"audio_url,omitempty"`
    Role     string    `json:"role,omitempty"`
}

type MediaURL struct {
    URL string `json:"url,omitempty"`
}
```

**关键发现**：`ContentItem` 已定义了 `VideoURL`、`AudioURL`、`Role` 字段，但 `convertToRequestPayload` 方法完全未使用它们。

### 5.2 本项目统一 TaskSubmitReq（缺失字段）

```go
// relay/common/relay_info.go
type TaskSubmitReq struct {
    Prompt         string                 `json:"prompt"`
    Model          string                 `json:"model,omitempty"`
    Mode           string                 `json:"mode,omitempty"`
    Image          string                 `json:"image,omitempty"`
    Images         []string               `json:"images,omitempty"`
    Size           string                 `json:"size,omitempty"`
    Duration       int                    `json:"duration,omitempty"`
    Seconds        string                 `json:"seconds,omitempty"`
    InputReference string                 `json:"input_reference,omitempty"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
```

**已确认缺失的字段**：

| 缺失字段 | 类型 | 对应官方功能 |
|---|---|---|
| `Videos []VideoItem` | 视频列表（需携带 URL 和 role） | 参考视频输入 |
| `Audios []AudioItem` | 音频列表（需携带 URL 和 role） | 参考音频输入 |
| `ImageRoles []string` | 每张图片的 role | 首帧/尾帧/参考图区分 |
| `DraftTaskID string` | 样片任务 ID | 样片模式 |
| `CallbackURL string` | 回调通知地址 | 任务状态回调 |

## 六、响应格式对比

### 6.1 创建任务响应

| 字段 | 官方响应 | 本项目统一响应 (OpenAIVideo) |
|---|---|---|
| 任务ID | `id` (string) | `ID` + `TaskID` (string) |
| 创建时间 | - | `CreatedAt` (int64) |
| 模型 | - | `Model` (string) |

官方说明：任务 ID 仅保存 7 天（从 `created_at` 开始计算），超时自动清除。

### 6.2 查询任务响应

| 字段 | 官方 responseTask | 本项目统一响应 |
|---|---|---|
| 任务ID | `id` | `ID` + `TaskID` |
| 模型 | `model` | `Model` (Properties.OriginModelName) |
| 状态 | `status` | `Status` (映射后) |
| 进度 | - | `Progress` (字符串，如 "10%", "50%", "100%") |
| 视频URL | `content.video_url` | `Metadata.url` |
| 错误码 | `error.code` | `Error.Code` |
| 错误信息 | `error.message` | `Error.Message` |
| Seed | `seed` | - |
| 分辨率 | `resolution` | - |
| 时长 | `duration` | - |
| 宽高比 | `ratio` | - |
| 帧率 | `framespersecond` | - |
| Service Tier | `service_tier` | - |
| Tools | `tools` | - |
| Usage | `usage.completion_tokens` / `usage.total_tokens` | 内部用于计费，不暴露 |
| Tool Usage | `usage.tool_usage.web_search` | - |
| 创建时间 | `created_at` | `CreatedAt` |
| 更新时间 | `updated_at` | `CompletedAt` |

### 6.3 状态映射

| 官方 status | 本项目内部状态 | 进度 |
|---|---|---|
| `pending` | `TaskStatusQueued` | 10% |
| `queued` | `TaskStatusQueued` | 10% |
| `processing` | `TaskStatusInProgress` | 50% |
| `running` | `TaskStatusInProgress` | 50% |
| `succeeded` | `TaskStatusSuccess` | 100% |
| `failed` | `TaskStatusFailure` | 100% |
| `expired` | **未显式映射**（走 default -> InProgress 30%） | 30% |

**注意**：`expired` 状态未被正确映射，当前走 default 分支被当作 `TaskStatusInProgress` 处理，这可能导致超时任务仍显示为"进行中"。

### 6.4 回调通知

官方 `callback_url` 推送内容与查询任务 API 返回体一致，回调返回的 status 包括：`queued`, `running`, `succeeded`, `failed`, `expired`。失败时 5 秒内回调三次。

## 七、计费相关

### 7.1 视频输入折扣

```go
// relay/channel/task/doubao/constants.go
var videoInputRatioMap = map[string]float64{
    "doubao-seedance-2-0-260128":      28.0 / 46.0, // ~0.6087
    "doubao-seedance-2-0-fast-260128": 22.0 / 37.0, // ~0.5946
}
```

**EstimateBilling** 方法通过 `hasVideoInMetadata` 检查 metadata 的 content 数组是否包含 `video_url`，若存在则应用视频输入折扣。但由于 `convertToRequestPayload` 不处理视频输入，该折扣实际上**仅在客户端直接在 metadata.content 中构造 video_url 时才会生效**。

### 7.2 官方计费说明

- 2.0/2.0 fast 支持在线推理 (default) 和离线推理 (flex)，离线推理价格为在线推理的 50%
- `duration` 设置为 `-1` 时，由模型自主选择时长，**与计费相关**，官方建议谨慎设置
- 样片模式 (draft) 消耗 token 数较正常视频更少，成本更低

## 八、缺失功能清单

| 优先级 | 缺失项 | 影响模型 | 影响场景 | 改动点 |
|---|---|---|---|---|
| P0 | 参考视频输入 (`video_url`) | 2.0/2.0 fast | 无法使用多模态参考生视频中的视频输入 | `TaskSubmitReq` 新增 `Videos` 字段 + adaptor 转换 |
| P0 | 参考音频输入 (`audio_url`) | 2.0/2.0 fast | 无法生成有声视频的音频参考驱动 | `TaskSubmitReq` 新增 `Audios` 字段 + adaptor 转换 |
| P0 | 图片 `role` 字段 | 所有图生视频模型 | 无法区分首帧/尾帧/参考图，图生视频-首尾帧完全不可用 | 图片需携带 role 或新增 `ImageRoles` 字段 |
| P0 | `expired` 状态映射 | 所有 | 超时任务被误判为"进行中" | `ParseTaskResult` 增加 `expired` -> `TaskStatusExpired` 映射 |
| P1 | 样片模式 (`draft` + `draft_task`) | 1.5 pro | 无法使用低成本预览能力 | `TaskSubmitReq` 新增 `DraftTaskID` 字段 |
| P1 | 联网搜索 (`tools`) | 2.0/2.0 fast | 无法使用联网搜索提升视频生成时效性 | `metadata` 已透传，需验证 `UnmarshalMetadata` 处理 |
| P1 | Base64 图片/音频支持 | 所有 | 官方支持 Base64 编码直接传入 | 当前 `Images` 为 URL 字符串，需确认是否支持 `data:` 前缀 |
| P1 | 素材 ID 支持 (`asset://`) | 所有 | 官方支持虚拟人像素材 | 当前 `Images` 为 URL 字符串，需确认是否支持 `asset://` 前缀 |
| P2 | 回调通知 (`callback_url`) | 所有 | 无法接收任务状态变更通知 | `TaskSubmitReq` 新增字段 |
| P2 | 安全标识 (`safety_identifier`) | 所有 | 无法协助平台检测违规行为 | `metadata` 已透传，当前可用 |
| P2 | metadata 强校验 | 所有 | 参数拼写错误被静默忽略 | 考虑将常用参数提升到 `TaskSubmitReq` 顶层 |

## 九、关键文件索引

| 文件 | 作用 |
|---|---|
| `router/video-router.go` | 视频路由注册，`/v1/video/generations` 指向 `controller.RelayTask` |
| `middleware/distributor.go` | 解析请求提取 model，设置 relay_mode，渠道分发 |
| `relay/common/relay_info.go` | `TaskSubmitReq` 统一请求结构体定义 (line 676) |
| `relay/common/relay_utils.go` | `ValidateBasicTaskRequest` 验证逻辑，`GetTaskRequest` 获取请求 |
| `relay/channel/task/doubao/adaptor.go` | Doubao 适配器：请求/响应结构体定义、转换逻辑、状态映射、计费估算 |
| `relay/channel/task/doubao/constants.go` | 模型列表 (`ModelList`) 和视频输入折扣 (`videoInputRatioMap`) |
| `middleware/kling_adapter.go` | 可灵 API 适配，转换为 `/v1/video/generations` 统一格式 |
| `middleware/jimeng_adapter.go` | 即梦 API 适配，转换为 `/v1/video/generations` 统一格式 |
| `controller/swag_video.go` | Swagger 文档注解，定义了各视频接口文档 |

## 十、核心结论

### 10.1 当前能力边界

Doubao adaptor 的 `requestPayload` 和 `ContentItem` 结构体已**完整定义**了 seedance 2.0 官方接口的所有字段（包括 `VideoURL`、`AudioURL`、`Role`），但 **`convertToRequestPayload` 转换逻辑仅实现了最基础的文本+图片能力**。

具体而言：
- **完全支持**：文生视频（text-only）、基础参数（resolution, ratio, duration, seed 等）通过 metadata 透传
- **部分支持**：图生视频-首帧（图片无 role 区分）、图生视频-首尾帧（**完全不可用**，无法指定 first_frame/last_frame）
- **完全不支持**：多模态参考生视频（缺视频/音频输入）、样片模式（缺 draft_task）、回调通知

### 10.2 需要改动的层面

1. **`TaskSubmitReq`** (`relay/common/relay_info.go`): 新增 `Videos`（携带 URL + role）、`Audios`（携带 URL + role）、图片 role 映射、`DraftTaskID`、`CallbackURL` 等字段
2. **`convertToRequestPayload`** (`doubao/adaptor.go`): 处理视频/音频输入转 `ContentItem`、role 映射（根据场景自动判断或读取客户端传入）、`draft_task` 类型 content
3. **`ValidateBasicTaskRequest`** (`relay/common/relay_utils.go`): 验证新增字段（视频数量限制、音频数量限制、不可单独输入音频等）
4. **`ParseTaskResult`** (`doubao/adaptor.go`): 增加 `expired` 状态映射
5. **前端/客户端**: 调用时传递新字段（视频 URL、音频 URL、图片 role 等）

### 10.3 改动策略建议

考虑到 `TaskSubmitReq` 是统一的多平台请求结构体（同时服务于 Doubao/可灵/即梦/Sora/Vidu/海螺等），建议：

- **方案 A**：在 `TaskSubmitReq` 中新增 `Videos`、`Audios` 等通用字段，各 adaptor 按需使用。适合长期多平台统一
- **方案 B**：仅在 `metadata` 中约定规范（如 `metadata.videos`, `metadata.audio`, `metadata.image_roles`），由 Doubao adaptor 自行解析。适合短期快速上线，但保持弱校验
- **推荐**：优先使用方案 B 快速支持核心功能（视频/音频输入、role 区分），后续迁移到方案 A
