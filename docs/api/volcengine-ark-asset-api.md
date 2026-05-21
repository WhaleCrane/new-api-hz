# Volcengine Ark 私域人像素材资产 API 接入文档

## 概述

本项目已接入火山引擎 Ark 平台的**私域人像素材资产**相关 API，支持两种素材类型：

- **真人人像素材**（s1，GroupType=`LivenessFace`）：需先完成真人认证，素材需与认证基准人脸一致
- **虚拟人像素材**（s2，GroupType=`AIGC`）：无需认证，直接创建素材组即可上传

所有 API 统一暴露在 `/s1/asset/`（真人）和 `/s2/aigc-asset/`（虚拟）路径下。

## 两套接口对比

| 对比项 | 真人素材 `/s1/asset/` | 虚拟人像 `/s2/aigc-asset/` |
|--------|-----------------------|--------------------------|
| GroupType | `LivenessFace` | `AIGC` |
| 前置要求 | H5 真人认证 | 无需认证 |
| 素材组创建 | 认证后自动创建 | 手动 `CreateAssetGroup` |
| 面部一致性校验 | 素材需与认证人脸比对 | 无 |
| Channel 类型 | VolcEngine (Type 45) | VolcEngine (Type 45) |

## 用户数据隔离

本系统采用**按用户隔离**的数据隔离策略：

- 所有下游用户共享平台方的同一套火山引擎 AK/SK
- 网关层通过 `user_id` 自动将用户与火山引擎 Asset Group 绑定
- 用户只能查看和操作自己名下的素材资产，**无法看到其他用户的素材**
- 请求中**无需传 `channel_id` 或 `project_name`**，网关自动从当前用户的 Token 中解析身份

## 前置准备

### 创建火山引擎 Channel

在 new-api 管理后台创建一个 **VolcEngine** 类型的 Channel（真人和虚拟素材共用）：

- **类型**：VolcEngine（Type 45，label 显示为「字节火山方舟、豆包通用」）
- **Key 格式**：`access_key|secret_key`（用 `|` 分隔）
- **BaseURL**：可留空（默认 `https://ark.cn-beijing.volces.com`），也可自定义
- **Models**：按需填写（如 `doubao-seedance-2-0-260128`）
- **Group**：按需设置

### Token 鉴权

下游用户使用自己的 API Token 调用本接口，网关自动识别用户身份并隔离数据。

---

## 一、真人素材流程（/s1/asset/）

### 1. 真人认证

#### 1.1 创建真人认证 H5 会话

拉起端上 H5 活体认证页。

- **路由**：`POST /s1/asset/visual-validate/session`

**请求体**：

```json
{
  "callback_url": "https://your-domain.com/callback"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| callback_url | string | 是 | 认证结束后跳转的回调地址 |

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

#### 1.2 获取认证结果

真人认证成功后，使用该接口获取本次认证创建的 Asset Group ID。**网关会自动将 GroupId 与当前用户绑定。**

- **路由**：`POST /s1/asset/visual-validate/result`

**请求体**：

```json
{
  "byted_token": "20260331145619CA67F03F8F*****"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| byted_token | string | 是 | 认证凭证（有效期 120 秒） |

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

### 2. 素材资产 CRUD

#### 2.1 上传素材（Create Asset）

上传图像/视频/音频素材。该接口为**异步接口**，需轮询查询状态。

- **路由**：`POST /s1/asset/assets/create`

**请求体**：

```json
{
  "group_id": "group-20260318070359-*****",
  "url": "https://example.com/image.jpg",
  "asset_type": "Image",
  "name": "全身参考图"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | string | 是 | 素材组 ID（完成真人认证后获取） |
| url | string | 是 | 素材可访问的 URL |
| asset_type | string | 是 | 素材类型：`Image` / `Video` / `Audio` |
| name | string | 否 | 素材名称，可用于模糊搜索 |

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
| group_type | string | 否 | 素材组类型，固定 `LivenessFace` |
| statuses | string[] | 否 | 状态过滤：`Active` / `Processing` / `Failed` |
| name | string | 否 | 素材名称（模糊搜索） |
| page_number | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 10 |
| sort_by | string | 否 | 排序字段 |
| sort_order | string | 否 | 排序方向：`Asc` / `Desc` |

> 注：系统会自动注入当前用户绑定的 `group_ids` 过滤条件，用户无需手动传入。

---

#### 2.3 获取单个素材（Get Asset）

轮询该接口确认素材上传状态。

- **路由**：`POST /s1/asset/assets/get`

**请求体**：

```json
{
  "id": "asset-20260318070533-*****"
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
  "id": "asset-20260318070533-*****",
  "name": "新素材名称"
}
```

---

#### 2.5 删除素材（Delete Asset）

- **路由**：`POST /s1/asset/assets/delete`

**请求体**：

```json
{
  "id": "asset-20260318070533-*****"
}
```

---

### 3. 素材资产组 CRUD

#### 3.1 查询素材组列表

- **路由**：`POST /s1/asset/asset-groups/list`

**请求体**：

```json
{
  "name": "figure_group",
  "group_type": "LivenessFace",
  "page_number": 1,
  "page_size": 10
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 素材组名称（模糊搜索） |
| group_type | string | 否 | 素材组类型，固定 `LivenessFace` |
| page_number | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

> 注：系统会自动注入当前用户绑定的 `group_ids` 过滤条件。

---

#### 3.2 获取素材组（Get Asset Group）

- **路由**：`POST /s1/asset/asset-groups/get`

**请求体**：

```json
{
  "id": "group-20260318033332-*****"
}
```

---

#### 3.3 更新素材组（Update Asset Group）

- **路由**：`POST /s1/asset/asset-groups/update`

**请求体**：

```json
{
  "id": "group-20260318033332-*****",
  "name": "新名称",
  "title": "新标题",
  "description": "新描述"
}
```

---

#### 3.4 删除素材组（Delete Asset Group）

- **路由**：`POST /s1/asset/asset-groups/delete`

**请求体**：

```json
{
  "id": "group-20260318033332-*****"
}
```

---

## 二、虚拟人像素材流程（/s2/aigc-asset/）

虚拟人像流程**无需真人认证**，直接创建素材组即可上传。

### 1. 创建素材资产组合

首次创建素材资产组合前，需先在火山引擎控制台签署授权函。

- **路由**：`POST /s2/aigc-asset/asset-groups/create`

**请求体**：

```json
{
  "name": "figure_group_1",
  "description": "虚拟人物A素材组",
  "project_name": "default"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 素材组合名称，上限 64 字符 |
| description | string | 否 | 素材组合描述，上限 300 字符 |
| project_name | string | 否 | 项目名称，默认 `default` |

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "Id": "group-20260318033332-*****"
  }
}
```

> 当前仅支持 AIGC 类型（虚拟人像），GroupType 由系统自动注入。

---

### 2. 查询素材组列表

- **路由**：`POST /s2/aigc-asset/asset-groups/list`

**请求体**：

```json
{
  "name": "figure_group",
  "group_type": "AIGC",
  "page_number": 1,
  "page_size": 10
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 素材组名称（模糊搜索） |
| group_type | string | 否 | 素材组类型，固定 `AIGC` |
| page_number | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 10 |

> 注：系统会自动注入当前用户绑定的 `group_ids` 过滤条件。

---

### 3. 获取素材组

- **路由**：`POST /s2/aigc-asset/asset-groups/get`

**请求体**：

```json
{
  "id": "group-20260318033332-*****"
}
```

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "Id": "group-20260318033332-*****",
    "Name": "figure_group_1",
    "Title": "figure_group_1",
    "Description": "Figure group 1",
    "GroupType": "AIGC",
    "ProjectName": "default",
    "CreateTime": "2026-03-18T03:33:32Z",
    "UpdateTime": "2026-03-18T03:33:32Z"
  }
}
```

---

### 4. 更新素材组

- **路由**：`POST /s2/aigc-asset/asset-groups/update`

**请求体**：

```json
{
  "id": "group-20260318033332-*****",
  "name": "新名称",
  "title": "新标题",
  "description": "新描述"
}
```

---

### 5. 删除素材组

- **路由**：`POST /s2/aigc-asset/asset-groups/delete`

**请求体**：

```json
{
  "id": "group-20260318033332-*****"
}
```

> 删除素材组将批量删除组内所有素材资产，该操作不可逆，请谨慎操作。

---

### 6. 上传素材（Create Asset）

上传图像/视频/音频素材。该接口为**异步接口**，需轮询查询状态。

- **路由**：`POST /s2/aigc-asset/assets/create`

**请求体**：

```json
{
  "group_id": "group-20260318070359-*****",
  "url": "https://example.com/image.jpg",
  "asset_type": "Image",
  "name": "全身参考图"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | string | 是 | 素材组 ID（调用 CreateAssetGroup 获取） |
| url | string | 是 | 素材可访问的 URL |
| asset_type | string | 是 | 素材类型：`Image` / `Video` / `Audio` |
| name | string | 否 | 素材名称，可用于模糊搜索 |

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

**素材文件格式要求**：

| 类型 | 格式 | 尺寸限制 | 大小限制 |
|------|------|----------|----------|
| Image | jpeg, png, webp, bmp, tiff, gif, heic/heif | 宽高比 (0.4, 2.5)，宽高 (300, 6000)px | < 30 MB |
| Video | mp4, mov | 480p/720p/1080p，时长 [2, 15]s，宽高比 [0.4, 2.5]，FPS [24, 60] | < 50 MB |
| Audio | wav, mp3 | 时长 [2, 15]s | < 15 MB |

---

### 7. 查询素材列表

- **路由**：`POST /s2/aigc-asset/assets/list`

**请求体**：

```json
{
  "group_type": "AIGC",
  "statuses": ["Active", "Processing"],
  "name": "全身",
  "page_number": 1,
  "page_size": 10,
  "sort_by": "CreateTime",
  "sort_order": "Desc"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_type | string | 否 | 素材组类型，固定 `AIGC` |
| statuses | string[] | 否 | 状态过滤：`Active` / `Processing` / `Failed` |
| name | string | 否 | 素材名称（模糊搜索） |
| page_number | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 10 |
| sort_by | string | 否 | 排序字段：`CreateTime` / `UpdateTime` / `GroupId` |
| sort_order | string | 否 | 排序方向：`Asc` / `Desc` |

> 注：系统会自动注入当前用户绑定的 `group_ids` 过滤条件。

---

### 8. 获取单个素材（轮询状态）

- **路由**：`POST /s2/aigc-asset/assets/get`

**请求体**：

```json
{
  "id": "asset-20260318070533-*****"
}
```

**响应示例**：

```json
{
  "success": true,
  "message": "",
  "data": {
    "Id": "asset-20260318035710-*****",
    "Name": "全身参考图",
    "URL": "https://...",
    "GroupId": "group-20260318033332-*****",
    "AssetType": "Image",
    "Status": "Active",
    "ProjectName": "default",
    "CreateTime": "2026-03-18T03:57:10Z",
    "UpdateTime": "2026-03-18T03:57:14Z"
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

### 9. 更新素材

- **路由**：`POST /s2/aigc-asset/assets/update`

**请求体**：

```json
{
  "id": "asset-20260318070533-*****",
  "name": "新素材名称"
}
```

---

### 10. 删除素材

- **路由**：`POST /s2/aigc-asset/assets/delete`

**请求体**：

```json
{
  "id": "asset-20260318070533-*****"
}
```

---

## 完整使用流程

### 真人素材流程（s1）

```
1. POST /s1/asset/visual-validate/session
   → 获取 H5Link 和 BytedToken

2. 用户通过 H5Link 完成真人认证
   → 跳转 callback_url，检查 resultCode 是否为 10000

3. POST /s1/asset/visual-validate/result（120秒内）
   → 获取 GroupId（网关自动绑定到当前用户）

4. POST /s1/asset/assets/create
   → 获取 AssetId，状态为 Processing

5. 轮询 POST /s1/asset/assets/get
   → 等待 Status 变为 Active 或 Failed

6. 使用 Asset URI：asset://<asset_id>
   → 在 Seedance 2.0 视频生成 API 中作为参考素材
```

### 虚拟人像流程（s2）

```
1. POST /s2/aigc-asset/asset-groups/create
   → 获取 GroupId（无需认证）

2. POST /s2/aigc-asset/assets/create
   → 获取 AssetId，状态为 Processing

3. 轮询 POST /s2/aigc-asset/assets/get
   → 等待 Status 变为 Active 或 Failed

4. 使用 Asset URI：asset://<asset_id>
   → 在 Seedance 2.0 视频生成 API 中作为参考素材
```

### 管理素材

```
/s1/asset/assets/list          → 查询真人素材列表
/s1/asset/asset-groups/list    → 查询真人素材组列表
/s1/asset/assets/delete        → 删除真人素材
/s1/asset/asset-groups/delete  → 删除真人素材组

/s2/aigc-asset/assets/list          → 查询虚拟素材列表
/s2/aigc-asset/asset-groups/list    → 查询虚拟素材组列表
/s2/aigc-asset/assets/delete        → 删除虚拟素材
/s2/aigc-asset/asset-groups/delete  → 删除虚拟素材组
```

---

## 限流要求

| 接口 | 限流 |
|------|------|
| CreateVisualValidateSession | 3 QPS |
| GetVisualValidateResult | 3 QPS |
| CreateAssetGroup | 10 QPS |
| CreateAsset | 根据权益等级，详见火山引擎文档 |
| ListAssetGroups / ListAssets | 10 QPS |
| GetAsset | 100 QPS |
| UpdateAsset / UpdateAssetGroup | 10 QPS |
| DeleteAsset | 10 QPS |
| DeleteAssetGroup | 5 QPS |

---

## 注意事项

1. **用户隔离**：每个用户只能查看和操作自己的素材资产，数据自动隔离
2. **真人认证前置**：真人素材用户必须先完成真人认证并获取 GroupId，才能上传素材
3. **虚拟人像无需认证**：直接创建素材组即可上传
4. **Project 隔离**：同一用户的素材和推理接入点必须在同一项目中
5. **面部一致性**：真人素材上传时，系统会将素材图像与真人认证基准人脸进行面部特征比对，非同一个人物无法入库
6. **异步处理**：CreateAsset 为异步接口，上传后需轮询 GetAsset 确认状态
7. **URL 有效期**：GetAsset 返回的素材 URL 有效期为 12 小时
8. **素材要求**（真人素材）：
   - 全身参考图：竖版，人物全身正面图片
   - 人脸特写图：竖版，人物正面无表情特写，肩部以上，面部占画面 2/3 左右
9. **视频生成指代**：在 Seedance Prompt 中需使用"图片1"、"视频1"等序号指代素材，不可直接使用 Asset ID
10. **素材 URI 拼接**：`asset://<asset_id>`
