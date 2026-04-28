# `/s1/video/generations` 接口调用文档

## 概述

`/s1/video/generations` 是火山方舟 Seedance 2.0 官方 API 格式的接口，支持 content 数组多模态输入（图片参考、视频参考、音频生成等）。

**完整路径：** `POST /s1/video/generations`
**轮询路径：** `GET /s1/video/generations/:task_id`

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

### 响应

**任务进行中：**

```json
{
  "id": "task_7f3a9b2c1d4e5f6a",
  "task_id": "task_7f3a9b2c1d4e5f6a",
  "object": "video",
  "model": "doubao-seedance-2-0-260128",
  "status": "in_progress",
  "progress": 45,
  "created_at": 1745600000
}
```

**任务完成：**

```json
{
  "id": "task_7f3a9b2c1d4e5f6a",
  "task_id": "task_7f3a9b2c1d4e5f6a",
  "object": "video",
  "model": "doubao-seedance-2-0-260128",
  "status": "completed",
  "progress": 100,
  "created_at": 1745600000,
  "data": {
    "result_url": "https://example.com/generated_video.mp4"
  }
}
```

**任务失败：**

```json
{
  "id": "task_7f3a9b2c1d4e5f6a",
  "task_id": "task_7f3a9b2c1d4e5f6a",
  "object": "video",
  "model": "doubao-seedance-2-0-260128",
  "status": "failed",
  "progress": 0,
  "created_at": 1745600000,
  "fail_reason": "上游服务返回错误信息"
}
```

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

### 示例 3：轮询任务状态

```bash
curl -X GET http://localhost:3000/s1/video/generations/task_7f3a9b2c1d4e5f6a \
  -H "Authorization: Bearer sk-xxx"
```

---

## 注意事项

1. **URL 必须公网可访问** — `image_url`、`video_url`、`audio_url` 中的 URL 必须是火山方舟服务器能访问到的公网地址
2. **content 数组必须有 text 项** — 即使只有提示词也必须包含
3. **角色字段 `role`** — 用于告诉模型该素材的用途（如 `first_frame` 作为首帧、`reference_video` 作为参考视频），不影响必填校验
4. **异步任务** — 提交后立即返回 `queued` 状态，需通过 `GET /s1/video/generations/:task_id` 轮询结果
5. **首帧**