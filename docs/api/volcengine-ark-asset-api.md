# Volcengine Ark 私域真人人像素材资产 API 接入文档

## 概述

本项目已接入火山引擎 Ark 平台的**私域真人人像素材资产**相关 API，支持以下功能：

- **真人认证**：拉起 H5 活体检测页面，完成真人认证后获取素材组（Asset Group）ID
- **素材资产管理**：上传、查询、更新、删除素材资产（Image/Video/Audio）
- **素材组管理**：查询、更新、删除素材资产组（Asset Group）

所有 API 统一暴露在 `/s1/asset/` 路径下。

## 前置准备

### 1. 创建火山引擎 Channel

在 new-api 管理后台创建一个 **VolcEngine** 类型的 Channel：

- **类型**：VolcEngine（Type 45）
- **Key 格式**：`access_key|secret_key`（用 `|` 分隔）
- **BaseURL**：可留空（默认 `https://ark.cn-beijing.volces.com`），也可自定义
- **Models**：按需填写（如 `doubao-seedance-2-0-260128`）
- **Group**：按需设置

### 2. 获取 Channel ID

创建 Channel 后，记录该 Channel 的 ID，所有素材 API 请求中都需要传入 `channel_id` 字段。

---

## 接口列表

### 一、真人认证

#### 1.1 创建真人认证 H5 会话

拉起端上 H5 活体认证页。

- **路由**：`POST /s1/asset/visual-validate/session`

**请求体**：

```json
{
  "channel_id": 1,
  "callback_url": "https://your-domain.com/callback",
  "project_name": "default"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | int | 是 | VolcEngine Channel ID |
| callback_url | string | 是 | 认证结束后跳转的回调地址 |
| project_name | string | 否 | 资源项目名称，默认 `default` |

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "BytedToken": "202603311449168C23BA26******",
    "H5Link": "https://h5-v2.kych5.com?accessKeyId=...&secretAccessKey=...&bytedToken=...&lng=zh",
    "CallbackURL": "https://your-domain.com/callback"
  }
}
```

| 返回字段 | 说明 |
|----------|------|
| BytedToken | 本次认证的唯一凭证，用于后续查询结果（有效期 120 秒） |
| H5Link | H5 活体认证页链接，有效期 120 秒 |
| CallbackURL | 回调地址 |

---

#### 1.2 获取真人认证结果

真人认证成功后，使用该接口获取本次认证创建的 Asset Group ID。

- **路由**：`POST /s1/asset/visual-validate/result`

**请求体**：

```json
{
  "channel_id": 1,
  "byted_token": "20260331145619CA67F03F8F*****"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | int | 是 | VolcEngine Channel ID |
| byted_token | string | 是 | 认证凭证（有效期 120 秒） |
| project_name | string | 否 | 资源项目名称 |

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "Result": {
      "GroupId": "group-20260331145705-*****"
    },
    "ResponseMetadata": {
      "Version": "2024-01-01",
      "Service": "ark",
      "Region": "cn-beijing",
      "RequestId": "20260331171930804FBCAD6EBE0C*****",
      "Action": "GetVisualValidateResult"
    }
  }
}
```

---

#### 1.3 CallbackURL 回调参数解析

终端用户完成真人认证后，将跳转至 `callback_url`，URL 拼接参数如下：

```
https://your-domain.com/callback?bytedToken=xxx&resultCode=10000&algorithmBaseRespCode=0&reqMeasureInfoValue=1&verify_type=real_time
```

| 参数 | 说明 |
|------|------|
| bytedToken | 本次认证的唯一凭证标识 |
| resultCode | **`10000` 表示认证成功**，其他值为错误码 |
| algorithmBaseRespCode | 服务端子错误码 |
| reqMeasureInfoValue | 是否计费：0 不计费，1 计费 |
| verify_type | 认证类型，固定为 `real_time` |

---

### 二、素材资产 CRUD

#### 2.1 上传素材（Create Asset）

上传图像/视频/音频素材。该接口为**异步接口**，需轮询查询状态。

- **路由**：`POST /s1/asset/assets/create`

**请求体**：

```json
{
  "channel_id": 1,
  "group_id": "group-20260318070359-*****",
  "url": "https://example.com/image.jpg",
  "asset_type": "Image",
  "name": "全身参考图",
  "project_name": "default"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | int | 是 | VolcEngine Channel ID |
| group_id | string | 是 | 素材组 ID |
| url | string | 是 | 素材可访问的 URL |
| asset_type | string | 是 | 素材类型：`Image` / `Video` / `Audio` |
| name | string | 否 | 素材名称，可用于模糊搜索 |
| project_name | string | 否 | 资源项目名称 |

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "Id": "asset-20260318071009-*****"
  }
}
```

**注意**：
- 每次请求仅上传一个素材文件
- 素材入库为异步处理，需轮询 `Get Asset` 接口确认状态
- 素材上传后系统会将上传图像与真人认证基准图像进行面部特征一致性比对

---

#### 2.2 查询素材列表（List Assets）

- **路由**：`POST /s1/asset/assets/list`

**请求体**：

```json
{
  "channel_id": 1,
  "group_ids": ["group-20260318033332-*****"],
  "group_type": "LivenessFace",
  "statuses": ["Active", "Processing"],
  "name": "全身",
  "page_number": 1,
  "page_size": 10,
  "sort_by": "GroupId",
  "sort_order": "Asc"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | int | 是 | VolcEngine Channel ID |
| group_ids | string[] | 否 | 素材组 ID 列表（精确匹配） |
| group_type | string | 否 | 素材组类型，固定 `LivenessFace` |
| statuses | string[] | 否 | 状态过滤：`Active` / `Processing` / `Failed` |
| name | string | 否 | 素材名称（模糊搜索） |
| page_number | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 10 |
| sort_by | string | 否 | 排序字段 |
| sort_order | string | 否 | 排序方向：`Asc` / `Desc` |

---

#### 2.3 获取单个素材（Get Asset）

轮询该接口确认素材上传状态。

- **路由**：`POST /s1/asset/assets/get`

**请求体**：

```json
{
  "channel_id": 1,
  "id": "asset-20260318070533-*****",
  "project_name": "default"
}
```

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "GroupId": "group-20260318033332-*****",
    "Status": "Active",
    "CreateTime": "2026-03-18T03:57:10Z",
    "AssetType": "Image",
    "UpdateTime": "2026-03-18T03:57:14Z",
    "ProjectName": "default",
    "Id": "asset-20260318035710-*****",
    "Name": "",
    "URL": "https://..."
  }
}
```

**素材状态**：

| 状态 | 说明 |
|------|------|
| `Processing` | 素材处理中，需继续轮询 |
| `Active` | 素材已就绪，可用于视频生成 |
| `Failed` | 处理失败，无法用于后续推理 |

---

#### 2.4 更新素材（Update Asset）

- **路由**：`POST /s1/asset/assets/update`

**请求体**：

```json
{
  "channel_id": 1,
  "id": "asset-20260318070533-*****",
  "name": "新素材名称",
  "project_name": "default"
}
```

---

#### 2.5 删除素材（Delete Asset）

- **路由**：`POST /s1/asset/assets/delete`

**请求体**：

```json
{
  "channel_id": 1,
  "id": "asset-20260318070533-*****",
  "project_name": "default"
}
```

---

### 三、素材资产组 CRUD

#### 3.1 查询素材组列表（List Asset Groups）

- **路由**：`POST /s1/asset/asset-groups/list`

**请求体**：

```json
{
  "channel_id": 1,
  "name": "figure_group",
  "group_ids": ["group-20260318033332-*****"],
  "group_type": "LivenessFace",
  "page_number": 1,
  "page_size": 10
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel_id | int | 是 | VolcEngine Channel ID |
| name | string | 否 | 素材组名称（模糊搜索） |
| group_ids | string[] | 否 | 素材组 ID 列表 |
| group_type | string | 否 | 素材组类型，固定 `LivenessFace` |
| page_number | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

---

#### 3.2 获取素材组（Get Asset Group）

- **路由**：`POST /s1/asset/asset-groups/get`

**请求体**：

```json
{
  "channel_id": 1,
  "id": "group-20260318033332-*****",
  "project_name": "default"
}
```

---

#### 3.3 更新素材组（Update Asset Group）

- **路由**：`POST /s1/asset/asset-groups/update`

**请求体**：

```json
{
  "channel_id": 1,
  "id": "group-20260318033332-*****",
  "name": "新名称",
  "title": "新标题",
  "description": "新描述",
  "project_name": "default"
}
```

---

#### 3.4 删除素材组（Delete Asset Group）

- **路由**：`POST /s1/asset/asset-groups/delete`

**请求体**：

```json
{
  "channel_id": 1,
  "id": "group-20260318033332-*****",
  "project_name": "default"
}
```

---

## 完整使用流程

### Step 1：创建真人人像素材组

```
1. 调用 POST /s1/asset/visual-validate/session
   → 获取 H5Link 和 BytedToken

2. 用户通过 H5Link 完成真人认证
   → 跳转 callback_url，检查 resultCode 是否为 10000

3. 调用 POST /s1/asset/visual-validate/result（120秒内）
   → 获取 GroupId
```

### Step 2：上传素材

```
1. 调用 POST /s1/asset/assets/create
   → 获取 Asset Id，状态为 Processing

2. 轮询 POST /s1/asset/assets/get
   → 等待 Status 变为 Active（可用）或 Failed（失败）

3. 使用 Asset URI 格式：asset://<asset_id>
   → 在 Seedance 2.0 视频生成 API 中作为参考素材
```

### Step 3：管理素材

```
- POST /s1/asset/assets/list     → 查询素材列表
- POST /s1/asset/asset-groups/list → 查询素材组列表
- POST /s1/asset/assets/delete   → 删除素材
- POST /s1/asset/asset-groups/delete → 删除素材组
```

---

## 限流要求

| 接口 | 限流 |
|------|------|
| CreateVisualValidateSession | 3 QPS |
| GetVisualValidateResult | 3 QPS |
| CreateAsset | 300 QPM |
| ListAssetGroups / ListAssets | 10 QPS |
| GetAsset | 100 QPS |
| UpdateAsset / UpdateAssetGroup / DeleteAsset | 10 QPS |
| DeleteAssetGroup | 5 QPS |

---

## 注意事项

1. **Project 隔离**：素材和推理接入点必须在同一项目中，建议统一使用 `default` 项目
2. **面部一致性**：上传的素材将与真人认证基准图像进行面部特征一致性比对，非同一个人物或多个人脸无法入库
3. **异步处理**：CreateAsset 为异步接口，上传后需轮询 GetAsset 确认状态
4. **URL 有效期**：GetAsset 返回的素材 URL 有效期为 12 小时
5. **素材要求**：
   - 全身参考图：竖版，人物全身正面图片
   - 人脸特写图：竖版，人物正面无表情特写，肩部以上，面部占画面 2/3 左右
6. **视频生成指代**：在 Seedance Prompt 中需使用"图片1"、"视频1"等序号指代素材，不可直接使用 Asset ID
