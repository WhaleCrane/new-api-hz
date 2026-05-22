# Volcengine Ark 素材资产 API 接入与修复记录

> 日期: 2026-05-19
> 分支: main
> 状态: 已完成

## 1. 背景

### 1.1 需求描述

在 new-api 网关中接入火山引擎 Ark 平台的**私域真人人像素材资产** API，支持以下能力：

- 真人认证（活体检测 H5 会话创建与结果查询）
- 素材资产 CRUD（图片/视频/音频的上传、查询、更新、删除）
- 素材组 CRUD（查询、更新、删除）

所有接口统一暴露在 `/s1/asset/` 路径下，由网关自动完成用户身份识别与数据隔离。

### 1.2 核心架构设计

| 维度 | 方案 |
|------|------|
| 鉴权方式 | 使用官方 `volcengine-go-sdk` 的 `universal.DoCall()` 自动签名 |
| 用户隔离 | 通过 `AssetGroupMapping` 表将用户与 Asset Group 绑定 |
| 渠道解析 | 网关层根据用户 ID 自动解析对应的 VolcEngine Channel |
| 字段约定 | 请求体使用 PascalCase（`GroupId`、`AssetType`），与火山引擎 API 一致 |

---

## 2. 遇到的问题与解决过程

### 2.1 问题一：Distribute 中间件拦截 — "Model name not specified"

**现象**: 调用 `/s1/asset/visual-validate/session` 返回 HTTP 400，错误信息 `Model name not specified`。

**根因**: `/s1/asset/` 路由绑定了 `middleware.Distribute()` 中间件。该中间件负责根据模型名自动选择上游 Channel，而素材资产接口不需要模型名，导致校验失败。

**解决方案**: 在 `middleware/distributor.go` 中为 `/s1/asset/` 路径添加旁路，跳过 Channel 选择：

```go
// middleware/distributor.go:289-291
} else if strings.HasPrefix(c.Request.URL.Path, "/s1/asset/") {
    shouldSelectChannel = false
}
```

Channel 的选择由 Service 层的 `resolveChannelForUser()` 在业务逻辑中自行处理。

### 2.2 问题二：手动签名始终返回 401 — AuthenticationError

**现象**: 调用接口返回 HTTP 401，错误信息 `AuthenticationError`，火山引擎侧无法验证签名。

**原始实现**: 在 `service/volcengine_sign.go` 中手写 HMAC-SHA256 签名，流程为：
1. 构造 Canonical Request（HTTP Method + URI + Query + Headers + SignedHeaders + Payload Hash）
2. 构造 String to Sign（Algorithm + RequestDate + CredentialScope + HashedCanonicalRequest）
3. 计算签名密钥（HMAC-SHA256 链式派生：Region → Service → Signing）
4. 计算最终签名并附加到 `Authorization` Header

尝试将 `SignedHeaders` 从 `host;x-date;content-type;content-sha256` 缩减为 `host;x-date`，仍然 401。

**根因分析**: 经对比官方文档 `F:\私域真人人像素材资产使用指南.md`，发现官方所有示例均使用 `github.com/volcengine/volcengine-go-sdk` 的 `universal.DoCall()` 发起请求，签名由 SDK 自动处理。手写签名在 Canonical Request 构造细节（Header 排序、URI 编码、Payload 哈希计算等）上与火山引擎服务端校验逻辑存在不可见的差异。

**解决方案**: 完全移除手写签名代码，改用官方 SDK 统一发起请求：

```go
// service/asset.go — CallArkAPI
config := volcengine.NewConfig().
    WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
    WithRegion("cn-beijing")

sess, _ := session.NewSession(config)
client := universal.New(sess)

resp, err := client.DoCall(
    universal.RequestUniversal{
        ServiceName: "ark",
        Action:      action,
        Version:     arkAPIVersion,
        HttpMethod:  universal.POST,
        ContentType: universal.ApplicationJSON,
    },
    &reqBody,
)
```

**关键变更**:

| 变更项 | 修改前 | 修改后 |
|--------|--------|--------|
| 签名方式 | 手写 HMAC-SHA256（`service/volcengine_sign.go`） | SDK 自动签名（`universal.DoCall()`） |
| 请求发送 | 手动构建 `http.Request` + `http.Client.Do()` | `universal.Universal.DoCall()` |
| 字段命名 | DTO 层 snake_case，签名时未转换 | 构造 `map[string]any` 时直接使用 PascalCase |
| 依赖 | 无额外依赖 | `github.com/volcengine/volcengine-go-sdk v1.2.28` |

**已删除文件**: `service/volcengine_sign.go`（`SignArkRequest` 函数及其全部辅助函数）。

### 2.3 问题三：编译错误修复

在重写过程中修复了以下编译错误：

| 错误 | 原因 | 修复 |
|------|------|------|
| `undefined: newHTTPClientWithProxy` | 代理处理函数不存在 | 改为内联创建 `http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}` |
| `undefined: strings` | import 中缺少 `strings` 包 | 在 import 中添加 `"strings"` |
| `undefined: net/http` | import 中缺少 `net/http` 包 | 在 import 中添加 `"net/http"` |
| `cannot use parts[0] as string` | `ParseVolcengineAssetAuth` 中直接对字符串按字节索引 | 改为 `strings.Split(key, "|")` 后再取索引 |

---

## 3. 文件变更清单

### 3.1 新增/修改

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `service/asset.go` | **重写** | 使用官方 SDK 替换手写签名，统一 11 个接口的请求体为 PascalCase |
| `middleware/distributor.go` | **修改** | 为 `/s1/asset/` 路径添加 Distribute 旁路 |
| `go.mod` | **修改** | 新增 `github.com/volcengine/volcengine-go-sdk v1.2.28` |

### 3.2 删除

| 文件 | 说明 |
|------|------|
| `service/volcengine_sign.go` | 手写 HMAC-SHA256 签名代码，已完全移除 |

### 3.3 保持不变（未修改但需了解）

| 文件 | 说明 |
|------|------|
| `controller/asset.go` | Controller 层，仅负责参数绑定和转发，无需修改 |
| `router/asset-router.go` | 路由定义，已正确注册所有 11 个端点 |
| `model/asset_group_mapping.go` | 用户-资产组映射模型，已存在 |
| `constant/channel.go` | `ChannelTypeVolcEngine = 45` 常量，已存在 |
| `docs/api/volcengine-ark-asset-api.md` | API 使用文档，已存在 |

---

## 4. 架构详情

### 4.1 请求处理链路

```
Client Request
  │
  ▼
router/asset-router.go     →  路由匹配 /s1/asset/*
  │
  ▼
middleware.TokenAuth()     →  验证 Token，提取 user_id
  │
  ▼
middleware.Distribute()    →  检测到 /s1/asset/ 前缀，跳过 Channel 选择
  │
  ▼
controller/asset.go        →  参数绑定（JSON → DTO），提取 user_id
  │
  ▼
service/asset.go           →  业务逻辑
  │                          ├── resolveChannelForUser(userId)
  │                          │     ├── 查询 AssetGroupMapping 表
  │                          │     └── 获取对应的 VolcEngine Channel (type=45)
  │                          ├── CallArkAPI(ctx, channel, action, reqBody)
  │                          │     ├── 解析 AK/SK（格式: access_key|secret_key）
  │                          │     ├── 创建 SDK Session + 代理支持
  │                          │     └── universal.DoCall() → 自动签名 + 发送
  │                          └── 返回火山引擎响应
  │
  ▼
common.ApiSuccess()        →  统一响应格式
```

### 4.2 用户数据隔离机制

系统通过 `AssetGroupMapping` 表实现用户级别的数据隔离：

```
┌─────────────────────────────────────────────────┐
│               AssetGroupMapping                 │
├────────────┬──────────┬────────────┬────────────┤
│ user_id    │ group_id │ channel_id │ volc_proj  │
│ (unique)   │ (unique) │            │            │
├────────────┼──────────┼────────────┼────────────┤
│ 1001       │ grp-xxx  │ 45         │ default    │
└────────────┴──────────┴────────────┴────────────┘
```

- **用户完成真人认证后**，系统自动创建映射记录
- **每次请求时**，`resolveChannelForUser()` 根据 `user_id` 查询映射
- **未绑定用户**：自动分配平台第一个可用的 VolcEngine Channel（type=45, status=1）
- **请求无需传 `channel_id` 或 `project_name`**，网关自动注入

### 4.3 SDK 调用模式

所有 11 个接口统一使用相同的调用模式，仅 `Action` 和请求体不同：

```
CallArkAPI(ctx, channel, action, reqBody)
  ├── ServiceName: "ark"（固定）
  ├── Version: "2024-01-01"（固定）
  ├── HttpMethod: POST（固定）
  ├── ContentType: application/json（固定）
  ├── Action: 各接口不同（见下表）
  └── reqBody: PascalCase 字段
```

### 4.4 接口与 Action 映射

| 路由 | Action | 请求体关键字段 |
|------|--------|---------------|
| `POST /s1/asset/visual-validate/session` | `CreateVisualValidateSession` | `CallbackURL`, `ProjectName` |
| `POST /s1/asset/visual-validate/result` | `GetVisualValidateResult` | `BytedToken`, `ProjectName` |
| `POST /s1/asset/assets/create` | `CreateAsset` | `GroupId`, `URL`, `AssetType`, `Name`, `ProjectName` |
| `POST /s1/asset/assets/list` | `ListAssets` | `Filter.GroupIds`, `Filter.GroupType`, `Filter.Statuses`, `Filter.Name` |
| `POST /s1/asset/assets/get` | `GetAsset` | `Id`, `ProjectName` |
| `POST /s1/asset/assets/update` | `UpdateAsset` | `Id`, `Name`, `ProjectName` |
| `POST /s1/asset/assets/delete` | `DeleteAsset` | `Id`, `ProjectName` |
| `POST /s1/asset/asset-groups/list` | `ListAssetGroups` | `Filter.Name`, `Filter.GroupIds`, `Filter.GroupType` |
| `POST /s1/asset/asset-groups/get` | `GetAssetGroup` | `Id`, `ProjectName` |
| `POST /s1/asset/asset-groups/update` | `UpdateAssetGroup` | `Id`, `Name`, `Title`, `Description`, `ProjectName` |
| `POST /s1/asset/asset-groups/delete` | `DeleteAssetGroup` | `Id`, `ProjectName` |

---

## 5. 关键代码片段

### 5.1 CallArkAPI — SDK 调用核心函数

```go
func CallArkAPI(ctx context.Context, channel *model.Channel, action string, reqBody map[string]any) (map[string]any, error) {
    // 1. 解析 AK/SK（格式: access_key|secret_key）
    accessKey, secretKey, err := ParseVolcengineAssetAuth(channel.Key)
    if err != nil {
        return nil, fmt.Errorf("parse volcengine auth failed: %w", err)
    }

    // 2. 创建 SDK 配置
    config := volcengine.NewConfig().
        WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, "")).
        WithRegion("cn-beijing")

    // 3. 支持 HTTP 代理（从 channel.Setting 读取）
    if channel.Setting != nil {
        var setting dtoChannelSetting
        if err := common.UnmarshalJsonStr(*channel.Setting, &setting); err == nil && setting.Proxy != "" {
            proxyURL, parseErr := url.Parse(setting.Proxy)
            if parseErr == nil {
                config.HTTPClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
            }
        }
    }

    // 4. 创建 Session + Client
    sess, err := session.NewSession(config)
    if err != nil {
        return nil, fmt.Errorf("create session failed: %w", err)
    }
    client := universal.New(sess)

    // 5. 发起请求（SDK 自动处理 HMAC-SHA256 签名）
    resp, err := client.DoCall(
        universal.RequestUniversal{
            ServiceName: "ark",
            Action:      action,
            Version:     arkAPIVersion,
            HttpMethod:  universal.POST,
            ContentType: universal.ApplicationJSON,
        },
        &reqBody,
    )
    if err != nil {
        return nil, err
    }
    if resp == nil {
        return nil, errors.New("response is nil")
    }

    // 6. 将响应转换为 map[string]any
    respBytes, _ := common.Marshal(resp)
    var result map[string]any
    _ = common.Unmarshal(respBytes, &result)
    return result, nil
}
```

### 5.2 ParseVolcengineAssetAuth — Key 解析

```go
func ParseVolcengineAssetAuth(key string) (accessKey, secretKey string, err error) {
    if key == "" {
        return "", "", errors.New("volcengine channel key is empty")
    }
    parts := strings.Split(key, "|")
    if len(parts) != 2 {
        return "", "", errors.New("invalid volcengine key format, expected: access_key|secret_key")
    }
    return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
```

### 5.3 Distributor 旁路

```go
// middleware/distributor.go:289-291
} else if strings.HasPrefix(c.Request.URL.Path, "/s1/asset/") {
    shouldSelectChannel = false
}
```

---

## 6. 运维指南

### 6.1 创建 VolcEngine Channel

在 new-api 管理后台创建一个 **VolcEngine** 类型的 Channel：

| 配置项 | 值 |
|--------|-----|
| 类型 | VolcEngine（Type = 45） |
| Key 格式 | `access_key\|secret_key`（用 `\|` 分隔） |
| BaseURL | 留空（默认 `https://ark.cn-beijing.volces.com`） |
| Models | 按需填写 |
| Group | 按需设置 |
| Status | Enabled |

### 6.2 Channel Key 格式说明

Key 必须严格按照以下格式填写：

```
AKLTMDNmxxxxxxxxxxxxxxx|WVRabVkyVxxxxxxxxxxxxxxxxxxxxx==
```

- `|` 左侧为 Access Key ID
- `|` 右侧为 Secret Access Key
- 前后不得有多余空格

### 6.3 用户认证流程

```
1. POST /s1/asset/visual-validate/session
   → 返回 H5Link（活体检测页面）和 BytedToken

2. 用户通过 H5Link 完成真人认证
   → 跳转 callback_url，检查 resultCode 是否为 10000

3. POST /s1/asset/visual-validate/result
   → 返回 GroupId，网关自动创建 AssetGroupMapping

4. 后续所有素材操作自动按用户隔离
```

---

## 7. 依赖变更

```diff
 // go.mod
+github.com/volcengine/volcengine-go-sdk v1.2.28
+github.com/volcengine/volc-sdk-golang v1.0.23  # 传递依赖
```

---

## 8. 验证结果

- `go build ./...` — 编译通过
- `go mod tidy` — 依赖清理完成
- 11 个 `/s1/asset/` 接口均已对齐官方文档的实现方式
