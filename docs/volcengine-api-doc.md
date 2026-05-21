# VolcEngine API 调用文档

> 本文档面向接入方开发者，描述通过本网关代理的火山引擎 Ark 系列 API。
> 所有接口均通过 Bearer Token 鉴权，Base URL 由平台提供。

---

## 目录

1. [认证方式](#1-认证方式)
2. [/s1/asset — Ark Asset API（真人头像）](#2-s1asset--ark-asset-api真人头像)
   - 2.1 真人认证
   - 2.2 素材组 CRUD
   - 2.3 素材 CRUD
3. [/s2/aigc-asset — AIGC Asset API（虚拟头像）](#3-s2aigc-asset--aigc-asset-api虚拟头像)
   - 3.1 素材组 CRUD
   - 3.2 素材 CRUD
4. [/s1/video/generations — Seedance 2.0 视频生成](#4-s1videogenerations--seedance-20-视频生成)
5. [通用响应格式](#5-通用响应格式)
6. [附录 A: 用户隔离说明](#附录-a-用户隔离说明)
7. [附录 B: 视频生成计费说明](#附录-b-视频生成计费说明)

---

## 1. 认证方式

所有接口使用 Bearer Token 鉴权，在请求 Header 中携带：

```
Authorization: Bearer <your-api-key>
Content-Type: application/json
```

---

## 2. /s1/asset — Ark Asset API（真人头像）

**Base Path**: `/s1/asset`

此路径提供火山引擎 Ark 真人认证（活体检测）及素材管理能力。用户需先完成真人认证，才能获得素材组（Group）的访问权限。所有操作自动按用户隔离——每个用户只能访问自己认证后创建的素材组和素材。

### 2.1 真人认证

#### 2.1.1 创建真人认证会话

**POST** `/s1/asset/visual-validate/session`

创建活体检测 H5 会话，返回认证链接。用户需在 H5 页面完成真人认证流程。

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `callback_url` | string | 是 | 认证完成后的回调地址 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**请求示例**:

```json
{
  "callback_url": "https://your-domain.com/callback"
}
```

**响应示例**:

```json
{
  "code": 0,
  "data": {
    "byted_token": "xxxxx",
    "h5_link": "https://verify.volcengine.com/xxx",
    "callback_url": "https://your-domain.com/callback"
  }
}
```

#### 2.1.2 获取认证结果

**POST** `/s1/asset/visual-validate/result`

查询认证结果。认证成功后，系统会自动将用户与返回的 Group ID 绑定，后续素材操作无需额外指定 Group。

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `byted_token` | string | 是 | 创建会话时返回的 token |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**请求示例**:

```json
{
  "byted_token": "xxxxx"
}
```

**响应示例**:

```json
{
  "code": 0,
  "data": {
    "Result": {
      "GroupId": "ag-xxxxxxxx",
      "ValidateResult": "pass"
    }
  }
}
```

> **注意**：认证成功后，系统会自动建立用户与 Group 的绑定关系。后续所有素材查询将自动限定在该用户的 Group 范围内。

---

### 2.2 素材组 CRUD

#### 2.2.1 查询素材组列表

**POST** `/s1/asset/asset-groups/list`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 否 | 按名称模糊搜索 |
| `group_type` | string | 否 | 组类型，可选值：`LivenessFace`（真人素材）、`AIGC`（虚拟人像），默认 `LivenessFace` |
| `page_number` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 20 |
| `sort_by` | string | 否 | 排序字段：`CreateTime`、`UpdateTime` |
| `sort_order` | string | 否 | 排序方向：`Desc`、`Asc` |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**响应体** (`ListAssetGroupsResp`):

| 字段 | 类型 | 说明 |
|------|------|------|
| `items` | array | 素材组列表，元素为 `AssetGroupDTO` |
| `total_count` | int | 总数 |
| `page_number` | int | 当前页码 |
| `page_size` | int | 每页数量 |

**AssetGroupDTO**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 素材组 ID |
| `name` | string | 名称 |
| `title` | string | 标题 |
| `description` | string | 描述 |
| `group_type` | string | 组类型 |
| `project_name` | string | 项目名称 |
| `create_time` | string | 创建时间 |
| `update_time` | string | 更新时间 |

#### 2.2.2 获取单个素材组

**POST** `/s1/asset/asset-groups/get`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材组 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 2.2.3 更新素材组

**POST** `/s1/asset/asset-groups/update`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材组 ID |
| `name` | string | 否 | 新名称 |
| `description` | string | 否 | 新描述 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 2.2.4 删除素材组

**POST** `/s1/asset/asset-groups/delete`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材组 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

---

### 2.3 素材 CRUD

#### 2.3.1 创建素材

**POST** `/s1/asset/assets/create`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `group_id` | string | 是 | 素材组 ID（需为认证后自动绑定的 Group） |
| `url` | string | 是 | 素材 URL 地址 |
| `asset_type` | string | 是 | 素材类型：`Image`、`Video`、`Audio` |
| `name` | string | 否 | 素材名称 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**响应示例**:

```json
{
  "code": 0,
  "data": {
    "id": "ast-xxxxxxxx"
  }
}
```

#### 2.3.2 查询素材列表

**POST** `/s1/asset/assets/list`

> 系统会自动注入用户拥有的 Group ID 过滤条件，确保只返回当前用户的素材。

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `group_type` | string | 否 | 组类型，可选值：`LivenessFace`（真人素材）、`AIGC`（虚拟人像），默认 `LivenessFace` |
| `statuses` | string[] | 否 | 状态过滤：`Active`、`Processing`、`Failed` |
| `name` | string | 否 | 按名称模糊搜索 |
| `page_number` | int | 否 | 页码 |
| `page_size` | int | 否 | 每页数量 |
| `sort_by` | string | 否 | 排序字段 |
| `sort_order` | string | 否 | 排序方向：`asc`、`desc` |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**响应体** (`ListAssetsResp`):

| 字段 | 类型 | 说明 |
|------|------|------|
| `items` | array | 素材列表，元素为 `AssetDTO` |
| `total_count` | int | 总数 |
| `page_number` | int | 当前页码 |
| `page_size` | int | 每页数量 |

**AssetDTO**:

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 素材 ID |
| `group_id` | string | 所属素材组 ID |
| `status` | string | 状态 |
| `asset_type` | string | 素材类型 |
| `url` | string | 素材 URL |
| `name` | string | 素材名称 |
| `project_name` | string | 项目名称 |
| `create_time` | string | 创建时间 |
| `update_time` | string | 更新时间 |

#### 2.3.3 获取单个素材

**POST** `/s1/asset/assets/get`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 2.3.4 更新素材

**POST** `/s1/asset/assets/update`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材 ID |
| `name` | string | 否 | 新名称 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 2.3.5 删除素材

**POST** `/s1/asset/assets/delete`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

---

## 3. /s2/aigc-asset — AIGC Asset API（虚拟头像）

**Base Path**: `/s2/aigc-asset`

此路径提供 AIGC 虚拟头像素材管理能力。与 `/s1/asset` 不同，AIGC 的素材组由用户自行创建（通过 `CreateAIGCAssetGroup`），无需真人认证。每个用户首次创建素材组时，系统会自动建立用户与 Group 的绑定，后续操作自动隔离。

### 3.1 素材组 CRUD

#### 3.1.1 创建素材组

**POST** `/s2/aigc-asset/asset-groups/create`

> 调用此接口后，系统会自动将用户与新建的 Group ID 绑定。

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 素材组名称 |
| `description` | string | 否 | 素材组描述 |
| `project_name` | string | 否 | 项目名称（不传则使用默认值） |

**响应示例**:

```json
{
  "code": 0,
  "data": {
    "id": "ag-xxxxxxxx"
  }
}
```

#### 3.1.2 查询素材组列表

**POST** `/s2/aigc-asset/asset-groups/list`

> 系统会自动注入当前用户拥有的 Group ID 过滤条件。

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 否 | 按名称模糊搜索 |
| `group_type` | string | 否 | 组类型（固定 `AIGC`） |
| `page_number` | int | 否 | 页码 |
| `page_size` | int | 否 | 每页数量 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**响应字段**: 与 `/s1/asset/asset-groups/list` 相同。

#### 3.1.3 获取单个素材组

**POST** `/s2/aigc-asset/asset-groups/get`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材组 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 3.1.4 更新素材组

**POST** `/s2/aigc-asset/asset-groups/update`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材组 ID |
| `name` | string | 否 | 新名称 |
| `description` | string | 否 | 新描述 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 3.1.5 删除素材组

**POST** `/s2/aigc-asset/asset-groups/delete`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材组 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

> 删除成功后，系统会自动清理本地用户与素材组的绑定映射。

---

### 3.2 素材 CRUD

#### 3.2.1 创建素材

**POST** `/s2/aigc-asset/assets/create`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `group_id` | string | 是 | 素材组 ID |
| `url` | string | 是 | 素材 URL 地址 |
| `asset_type` | string | 是 | 素材类型：`Image`、`Video`、`Audio` |
| `name` | string | 否 | 素材名称 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

**响应**: 返回素材 `id`。

#### 3.2.2 查询素材列表

**POST** `/s2/aigc-asset/assets/list`

> 系统自动注入用户的 Group ID 过滤条件。

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `group_ids` | string[] | 否 | 素材组 ID 列表（系统自动注入） |
| `group_type` | string | 否 | 组类型（`AIGC`） |
| `statuses` | string[] | 否 | 状态：`Active`、`Processing`、`Failed` |
| `name` | string | 否 | 按名称搜索 |
| `page_number` | int | 否 | 页码 |
| `page_size` | int | 否 | 每页数量 |
| `sort_by` | string | 否 | 排序字段 |
| `sort_order` | string | 否 | 排序方向 |

**响应字段**: 与 `/s1/asset/assets/list` 相同。

#### 3.2.3 获取单个素材

**POST** `/s2/aigc-asset/assets/get`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 3.2.4 更新素材

**POST** `/s2/aigc-asset/assets/update`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材 ID |
| `name` | string | 否 | 新名称 |
| `project_name` | string | 否 | 项目名称，默认 `default` |

#### 3.2.5 删除素材

**POST** `/s2/aigc-asset/assets/delete`

**请求体**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 素材 ID |
| `project_name` | string | 否 | 项目名称，默认 `default` |

---

## 4. /s1/video/generations — Seedance 2.0 视频生成

**Base Path**: `/s1`

此路径使用 Seedance 2.0 官方 API 格式，支持多模态 content array（文本、图片、视频、音频混合输入）。

### 4.1 提交视频生成任务

**POST** `/s1/video/generations`

**请求体** (`SeedanceSubmitReq`):

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 模型名称，如 `seedance-2-0` |
| `content` | array | 是 | 多模态内容数组，必须包含至少一个 `text` 类型项 |
| `resolution` | string | 否 | 分辨率：`480p`、`720p`（默认）、`1080p` |
| `ratio` | string | 否 | 画面比例 |
| `duration` | int | 否 | 视频时长（秒） |
| `frames` | int | 否 | 帧数 |
| `seed` | int | 否 | 随机种子 |
| `callback_url` | string | 否 | 任务完成后的回调 URL |
| `return_last_frame` | bool | 否 | 是否返回最后一帧 |
| `service_tier` | string | 否 | 服务层级 |
| `execution_expires_after` | int | 否 | 执行超时（秒） |
| `generate_audio` | bool | 否 | 是否生成音频 |
| `draft` | bool | 否 | 是否为草稿模式 |
| `camera_fixed` | bool | 否 | 相机是否固定 |
| `watermark` | bool | 否 | 是否添加水印 |
| `tools` | array | 否 | 工具列表，每项含 `type`（如 `web_search`） |

**Content Item 结构**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 内容类型：`text`、`image_url`、`video_url`、`audio_url`、`draft_task` |
| `text` | string | 条件 | 当 type=`text` 时必填，视频描述文案 |
| `image_url.url` | string | 条件 | 当 type=`image_url` 时必填，图片 URL |
| `video_url.url` | string | 条件 | 当 type=`video_url` 时必填，参考视频 URL（用于图/视频生视频） |
| `audio_url.url` | string | 条件 | 当 type=`audio_url` 时必填，音频 URL |
| `draft_task.id` | string | 条件 | 当 type=`draft_task` 时必填，草稿任务 ID |

**请求示例** — 文生视频:

```json
{
  "model": "seedance-2-0",
  "content": [
    {
      "type": "text",
      "text": "一只猫在草地上奔跑"
    }
  ],
  "resolution": "720p",
  "duration": 5
}
```

**请求示例** — 图生视频:

```json
{
  "model": "seedance-2-0",
  "content": [
    {
      "type": "image_url",
      "image_url": {
        "url": "https://example.com/input-image.jpg"
      }
    },
    {
      "type": "text",
      "text": "让图片中的场景动起来"
    }
  ],
  "resolution": "1080p"
}
```

**请求示例** — 视频生视频（视频编辑）:

```json
{
  "model": "seedance-2-0",
  "content": [
    {
      "type": "video_url",
      "video_url": {
        "url": "https://example.com/input-video.mp4"
      }
    },
    {
      "type": "text",
      "text": "将视频风格改为油画风"
    }
  ],
  "resolution": "720p"
}
```

**响应** (`OpenAIVideo`):

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 任务 ID |
| `object` | string | 固定 `"video"` |
| `model` | string | 使用的模型 |
| `status` | string | 初始状态：`queued` |
| `created_at` | int64 | 创建时间戳 |

### 4.2 查询视频生成任务状态

**GET** `/s1/video/generations/{task_id}`

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| `task_id` | string | 提交任务时返回的任务 ID |

**响应** (`OpenAIVideo`):

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 任务 ID |
| `object` | string | `"video"` |
| `model` | string | 使用的模型 |
| `status` | string | `queued`（排队中）、`in_progress`（处理中）、`completed`（完成）、`failed`（失败） |
| `progress` | int | 进度百分比 |
| `created_at` | int64 | 创建时间戳 |
| `completed_at` | int64 | 完成时间戳 |
| `metadata.url` | string | 生成结果视频 URL（仅在 completed 时返回） |
| `error.message` | string | 错误信息（仅在 failed 时返回） |
| `error.code` | string | 错误码（仅在 failed 时返回） |

---

## 5. 通用响应格式

### 5.1 成功响应

所有 Asset API 返回统一格式：

```json
{
  "code": 0,
  "msg": "",
  "data": { ... }
}
```

### 5.2 错误响应

```json
{
  "code": -1,
  "msg": "错误描述信息",
  "data": null
}
```

### 5.3 视频任务响应

视频生成任务使用 OpenAI 兼容格式响应，不包裹在 `data` 字段中：

```json
{
  "id": "video_xxx",
  "object": "video",
  "model": "seedance-2-0",
  "status": "queued",
  "created_at": 1716307200
}
```
