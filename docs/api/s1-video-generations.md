# `/s1/video/generations` 接口调用文档

## 概述

`/s1/video/generations` 是火山方舟 Seedance 2.0 官方 API 格式的接口，支持 content 数组多模态输入（图片参考、视频参考、音频生成等）。

**完整路径：** `POST /s1/video/generations`
**轮询路径：** `GET /s1/video/generations/:task_id`
**取消/删除路径：** `DELETE /s1/video/generations/:task_id`

---

## 认证

```
Authorization: Bearer sk-your-api-key
Content-Type: application/json
```

---

## 提交任务 — `POST /s1/video/generations`

### 请求体

```json
{
  "model": "doubao-seedance-2-0-260128",
  "content": [
    {
      "type": "text",
      "text": "一只猫在草地上奔跑"
    },
    {
      "type": "image_url",
      "image_url": {
        "url": "https://example.com/cat.jpg"
      },
      "role": "first_frame"
    }
  ],
  "resolution": "720p",
  "ratio": "16:9",
  "duration": 5,
  "generate_audio": true,
  "seed": 42
}
```

### 字段说明

#### 顶层字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | **是** | 模型名称 |
| `content` | ContentItem[] | **是** | 多模态内容数组，非空，必须包含至少一个 `text` 类型项 |
| `resolution` | string | 否 | 分辨率：`480p`、`720p`、`1080p` |
| `ratio` | string | 否 | 画面比例，如 `16:9` |
| `duration` | int | 否 | 视频时长（秒） |
| `seed` | int | 否 | 随机种子，用于结果可复现 |
| `generate_audio` | bool | 否 | 是否生成音频 |
| `return_last_frame` | bool | 否 | 是否返回最后一帧 |
| `service_tier` | string | 否 | 服务级别（如 `default`/`flex`） |
| `execution_expires_after` | int | 否 | 执行超时（秒） |
| `draft` | bool | 否 | 草稿模式 |
| `watermark` | bool | 否 | 是否添加水印 |
| `camera_fixed` | bool | 否 | 是否固定机位 |
| `callback_url` | string | 否 | 异步回调 URL |
| `safety_identifier` | string | 否 | 安全标识符（用于滥用举报） |
| `tools` | array | 否 | 工具列表，如 `[{"type": "web_search"}]` |
| `metadata` | object/string | 否 | 任意元数据 |

#### ContentItem 结构

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | **是** | 类型：`text`、`image_url`、`video_url`、`audio_url`、`draft_task` |
| `text` | string | 条件 | `type=text` 时必填，提示词内容 |
| `image_url` | `{url: string}` | 条件 | `type=image_url` 时使用，图片 URL |
| `video_url` | `{url: string}` | 条件 | `type=video_url` 时使用，参考视频 URL |
| `audio_url` | `{url: string}` | 条件 | `type=audio_url` 时使用，音频 URL |
| `draft_task` | `{id: string}` | 条件 | `type=draft_task` 时使用，草稿任务 ID |
| `role` | string | 否 | 角色提示：`first_frame`、`last_frame`、`reference_video` 等 |

**校验规则：**
- `content` 数组不能为空
- `content` 中必须至少包含一个 `type=text` 的项
- 每项的 `type` 必须是上述五种之一
- `text` 类型的 `text` 字段不能为空

### 支持模型

| 模型 ID | 说明 |
|---------|------|
| `doubao-seedance-1-0-pro-250528` | Seedance 1.0 Pro |
| `doubao-seedance-1-0-lite-t2v` | Seedance 1.0 Lite (文生视频) |
| `doubao-seedance-1-0-lite-i2v` | Seedance 1.0 Lite (图生视频) |
| `doubao-seedance-1-5-pro-251215` | Seedance 1.5 Pro |
| `doubao-seedance-2-0-260128` | Seedance 2.0 Pro |
| `doubao-seedance-2-0-fast-260128` | Seedance 2.0 Fast |

### 响应

```json
{
  "id": "task_7f3a9b2c1d4e5f6a",
  "task_id": "task_7f3a9b2c1d4e5f6a",
  "object": "video",
  "model": "doubao-seedance-2-0-260128",
  "status": "queued",
  "progress": 0,
  "created_at": 1745600000
}
```

---

## 查询任务 — `GET /s1/video/generations/:task_id`

### 请求

```
GET /s1/video/generations/task_7f3a9b2c1d4e5f6a
Authorization: Bearer sk-your-api-key
```

### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `format` | string | 否 | 响应格式，可选值：`ni`。默认不携带时返回上游原始响应体 |

### 响应

#### 默认响应（不携带 `format` 参数）

直接透传上游（火山方舟 Seedance）原始响应体：

```json
{
  "id": "cgt-20260522152917-h694x",
  "model": "doubao-seedance-2-0-260128",
  "status": "succeeded",
  "created_at": 1779434957,
  "updated_at": 1779435066,
  "content": {
    "video_url": "https://example.com/generated_video.mp4"
  },
  "duration": 5,
  "resolution": "480p",
  "ratio": "1:1",
  "usage": {
    "completion_tokens": 48400,
    "total_tokens": 48400
  }
}
```

> 注：上游原始响应的具体字段因模型版本和上游接口可能有所差异，以上为典型示例。

#### `?format=ni` 响应

返回 new-api 任务元数据 + 上游原始响应体，用 `ni` 和 `upstream` 字段明确区分：

```json
{
  "code": "success",
  "message": "",
  "data": {
    "ni": {
      "id": 1,
      "created_at": 1779434957,
      "updated_at": 1779435070,
      "task_id": "task_7f3a9b2c1d4e5f6a",
      "platform": "54",
      "user_id": 1,
      "group": "default",
      "channel_id": 1,
      "quota": 1104000,
      "action": "generate",
      "status": "SUCCESS",
      "fail_reason": "",
      "result_url": "https://example.com/generated_video.mp4",
      "submit_time": 1779434957,
      "start_time": 1779434958,
      "finish_time": 1779435070,
      "progress": "100%",
      "properties": {
        "upstream_model_name": "doubao-seedance-2-0-260128",
        "origin_model_name": "doubao-seedance-2-0-260128"
      }
    },
    "upstream": {
      "id": "cgt-20260522152917-h694x",
      "model": "doubao-seedance-2-0-260128",
      "status": "succeeded",
      "created_at": 1779434957,
      "updated_at": 1779435066,
      "content": {
        "video_url": "https://example.com/generated_video.mp4"
      },
      "duration": 5,
      "resolution": "480p",
      "ratio": "1:1",
      "usage": {
        "completion_tokens": 48400,
        "total_tokens": 48400
      }
    }
  }
}
```

#### `ni` 字段说明（new-api 任务元数据）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | new-api 内部任务 ID |
| `task_id` | string | 任务唯一标识 |
| `user_id` | int | 发起任务的用户 ID |
| `channel_id` | int | 使用的渠道 ID |
| `group` | string | 用户所属分组 |
| `quota` | int | 预扣费的 quota 数量 |
| `status` | string | 任务状态：`PENDING`、`SUCCESS`、`FAILURE` |
| `fail_reason` | string | 失败原因（仅失败时） |
| `result_url` | string | 结果 URL（仅成功时） |
| `progress` | string | 进度百分比，如 `"100%"` |
| `submit_time` | int | 提交时间戳 |
| `start_time` | int | 开始处理时间戳 |
| `finish_time` | int | 完成时间戳 |
| `properties` | object | 附加属性，包含上游模型名等信息 |

#### `upstream` 字段说明

上游（火山方舟 Seedance）原始响应体，字段以实际上游接口为准。

### 状态枚举

| 状态 | 说明 |
|------|------|
| `queued` | 任务排队中 |
| `in_progress` | 处理中 |
| `completed` | 已完成 |
| `failed` | 失败 |

---

## 完整调用示例

### 示例 1：图生视频（单图参考）

```bash
curl -X POST http://localhost:3000/s1/video/generations \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "content": [
      {
        "type": "text",
        "text": "让画面动起来，微风拂过，树叶轻轻摇曳"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/landscape.jpg"
        },
        "role": "first_frame"
      }
    ],
    "resolution": "1080p",
    "ratio": "16:9",
    "duration": 5
  }'
```

### 示例 2：视频参考生成（多图 + 参考视频）

```bash
curl -X POST http://localhost:3000/s1/video/generations \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "doubao-seedance-2-0-260128",
    "content": [
      {
        "type": "text",
        "text": "风格转换，将参考视频转换为赛博朋克风格"
      },
      {
        "type": "image_url",
        "image_url": {
          "url": "https://example.com/style_ref.jpg"
        },
        "role": "reference"
      },
      {
        "type": "video_url",
        "video_url": {
          "url": "https://example.com/motion_ref.mp4"
        },
        "role": "reference_video"
      }
    ],
    "resolution": "720p",
    "duration": 10,
    "generate_audio": true,
    "seed": 12345
  }'
```

### 示例 3：轮询任务状态（默认返回上游原始响应）

```bash
curl -X GET http://localhost:3000/s1/video/generations/task_7f3a9b2c1d4e5f6a \
  -H "Authorization: Bearer sk-xxx"
```

### 示例 4：轮询任务状态（返回 new-api 任务元数据 + 上游原始响应）

```bash
curl -X GET 'http://localhost:3000/s1/video/generations/task_7f3a9b2c1d4e5f6a?format=ni' \
  -H "Authorization: Bearer sk-xxx"
```

---

## 取消/删除任务 — `DELETE /s1/video/generations/:task_id`

### 请求

```
DELETE /s1/video/generations/task_7f3a9b2c1d4e5f6a
Authorization: Bearer sk-your-api-key
```

### 说明

取消排队中的视频生成任务，或者删除视频生成任务记录。

| 当前任务状态 | 是否支持 | 操作含义 | 操作后状态 |
|-------------|---------|---------|-----------|
| `queued` | 是 | 取消排队，状态变更为 `CANCELLED`，退还已扣 quota | `CANCELLED` |
| `in_progress` (running) | 否 | 不支持取消正在处理的任务 | - |
| `succeeded` | 是 | 删除任务记录，后续不支持查询 | - |
| `failed` | 是 | 删除任务记录，后续不支持查询 | - |
| `cancelled` | 否 | 已取消，不支持重复操作 | - |
| `expired` | 是 | 删除任务记录，后续不支持查询 | - |

### 响应

**成功：**

```json
{
  "code": "success",
  "message": "",
  "data": null
}
```

**失败示例（任务正在处理）：**

```json
{
  "error": {
    "code": "task_running_cannot_cancel",
    "message": "任务正在处理中，无法取消",
    "type": "new_api_error"
  }
}
```

### 示例 5：取消排队中的任务

```bash
curl -X DELETE http://localhost:3000/s1/video/generations/task_queued_id \
  -H "Authorization: Bearer sk-xxx"
```

---

## 注意事项

1. **URL 必须公网可访问** — `image_url`、`video_url`、`audio_url` 中的 URL 必须是火山方舟服务器能访问到的公网地址（需自行配置可公网方位URl)
2. **content 数组必须有 text 项** — 即使只有提示词也必须包含
3. **角色字段 `role`** — 用于告诉模型该素材的用途（如 `first_frame` 作为首帧、`reference_video` 作为参考视频），不影响必填校验
4. **异步任务** — 提交后立即返回 `queued` 状态，需通过 `GET /s1/video/generations/:task_id` 轮询结果
5. **查询响应格式** — 默认返回上游原始响应体；携带 `?format=ni` 参数时，返回 new-api 任务元数据（`data.ni`）和上游原始响应（`data.upstream`），两者明确区分