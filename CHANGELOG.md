# new-api-2026-5-20 变更日志

## 概述

本项目从 `new-api-new` 移植了 5 个功能模块，并进行了 CNY 货币基线审计修复。所有变更已于 2026-05-20 完成。

---

## 模块一：CNY 货币基线变更

### 背景

所有用户为国内客户，消除硬编码的 USD 汇率（7.3）复杂度。`QuotaPerUnit` 保持 500,000，但语义从 `$1` 改为 `¥1`。

### 变更清单

#### 1. `common/constants.go`

- **行 62**：注释从 `$0.002` 改为 `¥0.002`
- 代码：
  ```go
  var QuotaPerUnit = 500 * 1000.0 // ¥0.002 / 1K tokens
  ```

#### 2. `setting/ratio_setting/model_ratio.go`

- 删除了 `USD2RMB=7.3`、`USD=500`、`RMB` 常量
- ERNIE/GLM 模型比例直接使用计算后的值（如 `0.120 * RMB` → `0.016438`）
- 注释从 `$0.002` 改为 `¥0.002`

#### 3. `setting/operation_setting/payment_setting_old.go`

- 删除了 `Price=7.3` 和 `USDExchangeRate=7.3` 变量
- 文件仅保留支付网关配置（PayAddress、EpayId、EpayKey、PayMethods、MinTopUp）

#### 4. `setting/operation_setting/general_setting.go`

- 新增 4 个展示类型常量：`QuotaDisplayTypeUSD`、`QuotaDisplayTypeCNY`、`QuotaDisplayTypeTokens`、`QuotaDisplayTypeCustom`
- **默认值**：`QuotaDisplayType: QuotaDisplayTypeCNY`
- 新增字段：`CustomCurrencySymbol`、`CustomCurrencyExchangeRate`
- 新增函数：
  - `IsCurrencyDisplay()` — 判断是否以货币形式展示
  - `IsCNYDisplay()` — 判断是否以人民币展示
  - `GetQuotaDisplayType()` — 返回当前展示类型
  - `GetCurrencySymbol()` — 返回对应符号（`$`/`¥`/`¤`）
  - `GetUsdToCurrencyRate(usdToCny float64) float64` — 返回 1 USD = X <currency> 的 X

#### 5. `controller/topup.go`

- `getPayMoney()` 函数（行 148-175）：
  - 删除了 `dPrice := decimal.NewFromFloat(operation_setting.Price)` 汇率乘法
  - 改为 `payMoney := dAmount.Mul(dTopupGroupRatio).Mul(dDiscount)`
  - 新增 `QuotaDisplayTypeTokens` 分支处理
- `getMinTopup()` 函数（行 177-185）：新增 TOKENS 展示类型转换
- `RequestEpay()` 函数（行 241-245）：TOKENS 展示类型时转回原始金额存储

#### 6. `controller/channel-billing.go`

- Moonshot 计费逻辑（行 322-354）重写：
  - 删除了 CNY → USD 转换逻辑
  - 直接使用 CNY：`channel.UpdateBalance(availableBalanceCny); return availableBalanceCny, nil`
  - 注释标注："Moonshot 余额为 CNY，系统内部基准也为 CNY，无需转换"

#### 7. `controller/billing.go`

- `GetSubscription` 处理函数（行 11-69）：
  ```go
  switch operation_setting.GetQuotaDisplayType() {
  case operation_setting.QuotaDisplayTypeCNY:
      amount = amount / common.QuotaPerUnit
  case operation_setting.QuotaDisplayTypeTokens:
      // amount 保持 tokens 数量
  default:
      amount = amount / common.QuotaPerUnit
  }
  ```
- `GetUsage` 处理函数（行 71-108）：相同 switch 模式

#### 8. `controller/misc.go`

- `GetStatus` API 响应新增字段（行 51-120）：
  ```go
  "quota_display_type":            operation_setting.GetQuotaDisplayType(),
  "custom_currency_symbol":        operation_setting.GetGeneralSetting().CustomCurrencySymbol,
  "custom_currency_exchange_rate": operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate,
  ```

#### 9. `controller/ratio_sync.go`

- `convertOpenRouterToRatioData`（行 777）：
  ```go
  ratio := promptPrice * common.QuotaPerUnit
  ```
- `convertModelsDevToRatioData`（行 960）：
  ```go
  modelRatio := candidate.Input * (common.QuotaPerUnit / 1000) / modelsDevInputCostRatioBase
  ```
- 两处均不再引用 `ratio_setting.USD`

#### 10. `logger/logger.go`

- `LogQuota()` 函数（行 122-146）完全重写，支持 `QuotaDisplayType` switch：
  ```go
  case QuotaDisplayTypeCNY:    → fmt.Sprintf("¥%.6f 额度", cny)
  case QuotaDisplayTypeCustom: → fmt.Sprintf("%s%.6f 额度", symbol, v)
  case QuotaDisplayTypeTokens: → fmt.Sprintf("%d 点额度", quota)
  default (USD):               → fmt.Sprintf("$%.6f 额度", ...)
  ```
- `FormatQuota()` 函数（行 148-171）：相同 switch 模式

#### 11. `model/subscription.go`

- `SubscriptionPlan.Currency` 默认值（行 153）：
  ```go
  Currency string `gorm:"...default:'CNY'"`
  ```

#### 12. `model/main.go`

- SQLite `subscription_plans` 表 CREATE TABLE（行 398）：
  ```sql
  `currency` varchar(8) NOT NULL DEFAULT 'CNY'
  ```
- SQLite `subscription_plans` 表 ADD COLUMN 迁移（行 431）：
  ```go
  {Name: "currency", DDL: "`currency` varchar(8) NOT NULL DEFAULT 'CNY'"}
  ```

#### 13. `model/option.go`

- 删除了 `Price` 和 `USDExchangeRate` 选项注册
- 删除了 `case "Price"` 和 `case "USDExchangeRate"` 更新处理
- `DisplayInCurrencyEnabled` 向后兼容处理（行 278-287）：
  ```go
  case "DisplayInCurrencyEnabled":
      newVal := "USD"
      if !boolValue { newVal = "TOKENS" }
      // 映射到新的 general_setting.quota_display_type
  ```

---

## 模块二：/s1/video/generations 端点（Seedance 2.0 官方 API）

### 背景

现有 `/v1/` 端点的简化格式无法支持 Seedance 2.0 的完整多模态内容数组（视频/音频引用、角色区分、草稿模式）。

### 变更清单

#### 1. 新建 `relay/constant/relay_mode.go`

- 新增中继模式常量：
  ```go
  RelayModeVideoFetchByID  // 行 43
  RelayModeVideoSubmit     // 行 44
  RelayModeSeedanceSubmit  // 行 46
  RelayModeSeedanceFetchByID // 行 47
  ```

#### 2. 新建 `router/video-router.go`

- `/s1` 路由组（行 35-43）：
  ```go
  videoS1Router := router.Group("/s1")
  videoS1Router.Use(middleware.RouteTag("relay"))
  videoS1Router.Use(middleware.TokenAuth(), middleware.Distribute())
  {
      videoS1Router.POST("/video/generations", controller.RelayTask)
      videoS1Router.GET("/video/generations/:task_id", controller.RelayTaskFetch)
  }
  ```

#### 3. 修改 `middleware/distributor.go`

- `/s1/video/generations` 路径识别（行 265-280）：
  - POST 请求从 body 提取 model，设置 `RelayModeSeedanceSubmit`
  - GET 请求设置 `RelayModeSeedanceFetchByID`，`shouldSelectChannel = false`
- `/s1/asset/` 和 `/s2/aigc-asset/` 绕过（行 289-290）：
  ```go
  } else if strings.HasPrefix(c.Request.URL.Path, "/s1/asset/") || strings.HasPrefix(c.Request.URL.Path, "/s2/aigc-asset/") {
      shouldSelectChannel = false
  }
  ```

#### 4. 修改 `relay/common/relay_info.go`

- `ChannelTypeVolcEngine` 添加到 `streamSupportedChannels` 映射（行 319）
- 新增 Seedance 结构体：
  - `SeedanceContentItem` — 内容项结构
  - `SeedanceMediaURL` — 媒体 URL 结构
  - `SeedanceSubmitReq` — 提交请求结构（行 898-939）
    - `GetPrompt()` 方法（行 941-948）
    - `HasVideo()` 方法（行 950-957）

#### 5. 修改 `relay/common/relay_utils.go`

- 新增 `ValidateSeedanceTaskRequest` 函数（行 226-262）
  - 验证 Seedance 2.0 官方格式请求
  - 验证 content 数组结构

#### 6. 新建 `relay/channel/task/doubao/adaptor.go`

完整 Doubao 任务适配器：

| 函数 | 行号 | 功能 |
|------|------|------|
| `TaskAdaptor` | - | 嵌入 `BaseBilling` 的结构体 |
| `ValidateRequestAndSetAction` | - | 分发到 Seedance 或标准验证 |
| `BuildRequestURL` | - | `https://{baseURL}/api/v3/contents/generations/tasks` |
| `EstimateBilling` | 143-162 | 4 级视频分辨率计费检测（`low`/`high`/`low_video`/`high_video`） |
| `estimateSeedanceBilling` | 165-189 | Seedance 2.0 官方格式计费处理 |
| `buildSeedanceRequestBody` | 286-303 | 直接透传 Seedance 2.0 格式到上游 |
| `buildStandardRequestBody` | 306-326 | 将标准 `TaskSubmitReq` 转换为 Doubao 格式 |
| `ParseTaskResult` | - | 映射 Doubao 状态到内部任务状态 |
| `ConvertToOpenAIVideo` | - | 将 Doubao 响应转为 OpenAI 兼容视频格式 |

#### 7. 新建 `relay/channel/task/doubao/constants.go`

- `ModelList` — Doubao Seedance 模型列表（5 个模型）
- `ChannelName = "doubao-video"`
- `resolutionVideoRatioMap` — 4 级折扣比例映射：
  | tierKey | doubao-seedance-2-0-260128 | doubao-seedance-2-0-fast-260128 |
  |---------|---------------------------|---------------------------------|
  | `low` | 1.0 | 1.0 |
  | `high` | ≈1.1087 | ≈1.1087 |
  | `low_video` | ≈0.6087 | ≈0.6087 |
  | `high_video` | ≈0.6739 | ≈0.6739 |
- `GetResolutionVideoRatio` 函数

---

## 模块三：分辨率 × 视频输入 4 级计费

### 背景

旧版计费将 1080p 与 480p/720p 等同计费，但官方价格差异 10.87%。

### 变更清单

#### 1. `relay/channel/task/doubao/constants.go`

- 替换 `videoInputRatioMap`/`GetVideoInputRatio` 为 `resolutionVideoRatioMap`/`GetResolutionVideoRatio`
- 4 级费率：`low=1.0`、`high≈1.1087`、`low_video≈0.6087`、`high_video≈0.6739`

#### 2. `relay/channel/task/doubao/adaptor.go`

- `EstimateBilling` 替换为四级逻辑：
  - `estimateSeedanceBilling` — Seedance 官方格式计费
  - `resolveResolutionTier` — 解析分辨率等级
  - `tierKey` — 分级键

---

## 模块四：VolcEngine Ark 素材 API（/s1/asset）— 真人头像

### 背景

需要私有域真人头像素材管理，含活体验证和人脸一致性检查。

### 新建文件清单

#### 1. `dto/asset.go`

| 结构体 | 用途 |
|--------|------|
| `CreateVisualValidateSessionReq` / `Resp` | 活体验证会话创建 |
| `GetVisualValidateResultReq` / `Resp` | 活体验证结果查询 |
| `CreateAssetReq` / `Resp` | 素材创建 |
| `ListAssetsReq` / `AssetDTO` / `Resp` | 素材列表查询 |
| `GetAssetReq` | 获取单个素材 |
| `UpdateAssetReq` | 更新素材 |
| `DeleteAssetReq` | 删除素材 |
| `ListAssetGroupsReq` / `AssetGroupDTO` / `Resp` | 素材组列表查询 |
| `GetAssetGroupReq` | 获取单个素材组 |
| `UpdateAssetGroupReq` | 更新素材组 |
| `DeleteAssetGroupReq` | 删除素材组 |

#### 2. `model/asset_group_mapping.go`

- `AssetGroupMapping` GORM 模型：
  ```go
  type AssetGroupMapping struct {
      ID              int    `gorm:"primaryKey;autoIncrement"`
      UserId          int    `gorm:"uniqueIndex:idx_user_group;index"`
      GroupId         string `gorm:"uniqueIndex:idx_user_group;type:varchar(128)"`
      ChannelId       int    `gorm:"index"`
      VolcProjectName string `gorm:"type:varchar(64)"`
      Name            string `gorm:"type:varchar(128)"`
      CreatedAt       int64
      UpdatedAt       int64
  }
  ```
- 函数：`GetUserAssetGroupMapping`、`InsertAssetGroupMapping`、`GetAssetGroupMappingByGroupId`、`GetUserAssetGroupNames`

#### 3. `service/asset.go`

核心业务逻辑，集成 VolcEngine SDK：

| 函数 | 功能 |
|------|------|
| `CallArkAPI` | 调用 VolcEngine Ark API，使用 `volcengine/universal` 自动 HMAC-SHA256 签名 |
| `ParseVolcengineAssetAuth` | 解析 channel key 格式 `access_key|secret_key` |
| `resolveChannelForUser` | 获取平台默认 VolcEngine channel（type=45）和用户的 project name |
| `CreateUserAssetMapping` | 活体验证后创建用户-素材组映射 |
| `ensureUserMapping` | 确保用户有映射，无则自动分配 channel |
| `GetGroupIDsForUser` | 返回用户拥有的 group IDs（用于查询隔离） |
| `CreateVisualValidateSession` | 创建活体验证会话 |
| `GetVisualValidateResult` | 获取活体验证结果 |
| `CreateAsset` / `ListAssets` / `GetAsset` / `UpdateAsset` / `DeleteAsset` | 素材 CRUD |
| `ListAssetGroups` / `GetAssetGroup` / `UpdateAssetGroup` / `DeleteAssetGroup` | 素材组 CRUD |

#### 4. `controller/asset.go`

11 个 HTTP 处理函数，对应 11 个端点。

#### 5. `router/asset-router.go`

- 路由组 `/s1/asset`
- 中间件：`middleware.TokenAuth()` + `middleware.Distribute()`
- 端点：
  ```
  POST /s1/asset/visual-validate/session
  POST /s1/asset/visual-validate/result
  POST /s1/asset/assets/create
  POST /s1/asset/assets/list
  POST /s1/asset/assets/get
  POST /s1/asset/assets/update
  POST /s1/asset/assets/delete
  POST /s1/asset/asset-groups/list
  POST /s1/asset/asset-groups/get
  POST /s1/asset/asset-groups/update
  POST /s1/asset/asset-groups/delete
  ```

### 修改的现有文件

| 文件 | 变更 |
|------|------|
| `go.mod` | 添加 `github.com/volcengine/volcengine-go-sdk v1.2.28` |
| `middleware/distributor.go` | 添加 `/s1/asset/` 前缀绕过（`shouldSelectChannel = false`） |
| `model/main.go` | `migrateDB()` 注册 `&AssetGroupMapping{}`，SQLite 迁移注册表结构 |
| `router/main.go` | 添加 `SetAssetRouter(router)` 调用 |

---

## 模块五：VolcEngine AIGC 素材 API（/s2/aigc-asset）— 虚拟头像

### 背景

需要虚拟头像素材管理，无需活体验证前置条件。

### 新建文件清单

#### 1. `dto/aigc_asset.go`

与模块四相同的 DTO 模式，`GroupType` 默认为 `AIGC`。

| 结构体 | 用途 |
|--------|------|
| `CreateAIGCAssetGroupReq` / `Resp` | AIGC 素材组创建 |
| `ListAIGCAssetGroupsReq` | AIGC 素材组列表查询 |
| `GetAIGCAssetGroupReq` / `UpdateAIGCAssetGroupReq` / `DeleteAIGCAssetGroupReq` | AIGC 素材组操作 |
| `CreateAIGCAssetReq` / `Resp` | AIGC 素材创建 |
| `ListAIGCAssetsReq` | AIGC 素材列表查询 |
| `GetAIGCAssetReq` / `UpdateAIGCAssetReq` / `DeleteAIGCAssetReq` | AIGC 素材操作 |

#### 2. `model/aigc_asset_group_mapping.go`

- `AIGCAssetGroupMapping` GORM 模型（与模块四独立表，索引 `idx_aigc_user_group`）：
  ```go
  type AIGCAssetGroupMapping struct {
      ID              int    `gorm:"primaryKey;autoIncrement"`
      UserId          int    `gorm:"uniqueIndex:idx_aigc_user_group;index"`
      GroupId         string `gorm:"uniqueIndex:idx_aigc_user_group;type:varchar(128)"`
      ChannelId       int    `gorm:"index"`
      VolcProjectName string `gorm:"type:varchar(64)"`
      Name            string `gorm:"type:varchar(128)"`
      CreatedAt       int64
      UpdatedAt       int64
  }
  ```
- 函数：`GetUserAIGCAssetGroupMapping`、`GetUserAIGCGroupIDs`、`InsertAIGCAssetGroupMapping`、`DeleteAIGCAssetGroupMapping`

#### 3. `service/aigc_asset.go`

| 函数 | 功能 |
|------|------|
| `resolveAIGCChannelForUser` | 查询 `AIGCAssetGroupMapping` 获取 channel 和 project name |
| `CreateAIGCAssetGroup` | 创建素材组，成功后自动创建映射 |
| `CreateAIGCAssetMapping` | 创建 AIGC 用户-素材映射 |
| `CreateAIGCAsset` / `ListAIGCAssets` / `GetAIGCAsset` / `UpdateAIGCAsset` / `DeleteAIGCAsset` | AIGC 素材 CRUD |
| `ListAIGCAssetGroups` / `GetAIGCAssetGroup` / `UpdateAIGCAssetGroup` / `DeleteAIGCAssetGroup` | AIGC 素材组 CRUD |

#### 4. `controller/aigc_asset.go`

10 个 HTTP 处理函数（无活体验证端点）。

#### 5. `router/aigc-asset-router.go`

- 路由组 `/s2/aigc-asset`
- 中间件：`middleware.TokenAuth()` + `middleware.Distribute()`
- 端点：
  ```
  POST /s2/aigc-asset/asset-groups/create
  POST /s2/aigc-asset/asset-groups/list
  POST /s2/aigc-asset/asset-groups/get
  POST /s2/aigc-asset/asset-groups/update
  POST /s2/aigc-asset/asset-groups/delete
  POST /s2/aigc-asset/assets/create
  POST /s2/aigc-asset/assets/list
  POST /s2/aigc-asset/assets/get
  POST /s2/aigc-asset/assets/update
  POST /s2/aigc-asset/assets/delete
  ```

### 修改的现有文件

| 文件 | 变更 |
|------|------|
| `middleware/distributor.go` | 添加 `/s2/aigc-asset/` 前缀绕过 |
| `model/main.go` | `migrateDB()` 注册 `&AIGCAssetGroupMapping{}` |
| `router/main.go` | 添加 `SetAIGCAssetRouter(router)` 调用 |
| `constant/channel.go` | M4/M5 共用 `ChannelTypeVolcEngine = 45`（label: "VolcEngine"），通过路由路径和 GroupType 区分 |

### M4/M5 渠道合并说明

- M4（真人头像）和 M5（虚拟头像）均使用 `ChannelTypeVolcEngine = 45`
- Base URL：`https://ark.cn-beijing.volces.com`
- 通过以下维度区分：
  - **路由路径**：`/s1/asset/` vs `/s2/aigc-asset/`
  - **GroupType**：`Asset` vs `AIGC`
  - **映射表**：`asset_group_mappings` vs `aigc_asset_group_mappings`

---

## 模块六：CNY 审计关键 BUG 修复

### 背景

CNY 货币基线变更完成后进行全量审计，发现 2 处 `USD` 未改为 `CNY` 的关键 BUG，均位于金融逻辑路径。

### BUG 1：默认 QuotaDisplayType 为 USD 而非 CNY

- **文件**：`setting/operation_setting/general_setting.go` 行 30
- **修改前**：`QuotaDisplayType: QuotaDisplayTypeUSD`
- **修改后**：`QuotaDisplayType: QuotaDisplayTypeCNY`
- **影响**：新安装默认 USD 展示，`IsCNYDisplay()` 和 `GetCurrencySymbol()` 返回值错误，直到管理员手动修改设置

### BUG 2：SQLite 迁移 ADD COLUMN 中 currency 默认值为 USD

- **文件**：`model/main.go` 行 431
- **修改前**：`{Name: "currency", DDL: "`currency` varchar(8) NOT NULL DEFAULT 'USD'"}`
- **修改后**：`{Name: "currency", DDL: "`currency` varchar(8) NOT NULL DEFAULT 'CNY'"}`
- **影响**：已有 SQLite 数据库迁移时缺少 `currency` 列的新增列默认值为 `'USD'` 而非 `'CNY'`，影响现有用户计费计算
- **注意**：GORM 模型 `model/subscription.go` 和主 CREATE TABLE 中已正确改为 `DEFAULT 'CNY'`，仅 ADD COLUMN 备用路径遗漏

---

## 前端默认值（待处理）

以下前端默认值与后端不一致（后端已改为 CNY，前端仍为 USD），优先级较低，与源项目行为一致。

| 文件 | 行号 | 当前值 | 应改为 |
|------|------|--------|--------|
| `web/default/src/stores/system-config-store.ts` | 51 | `quotaDisplayType: 'USD'` | `quotaDisplayType: 'CNY'` |
| `web/default/src/features/subscriptions/types.ts` | 30 | `currency: z.string().default('USD')` | `currency: z.string().default('CNY')` |

---

## 验证

- `go build ./...` 编译通过
- `model/main.go` 中无残留 `DEFAULT 'USD'` 字符串
- `general_setting.go` 中 `QuotaDisplayTypeUSD` 仅在常量声明和函数分支中使用，不再作为默认值
- 无残留对 `operation_setting.Price`/`USDExchangeRate`/`USD2RMB` 的引用
