# 变更记录 - 2026-04-25

## 将系统结算基准货币从美元 (USD) 改为人民币 (CNY)

系统内部计价基准从美元切换为人民币，所有展示、支付、计费逻辑统一以 CNY 为单位，移除硬编码的美元-人民币汇率转换。

## 背景/需求

项目的下游用户全部是国内人员，不涉及美元支付场景。原先系统以美元为内部基准货币（`QuotaPerUnit = 500,000` = $1），通过展示层 `QuotaDisplayType` 转换为 CNY 显示（汇率 7.3），存在以下问题：

1. 实际结算和展示之间存在语义不一致——用户看到 ¥10，但系统内部计算基于 $10
2. Epay 国内支付需要乘以 `Price = 7.3` 做汇率转换，增加了不必要的计算复杂度
3. 硬编码汇率 `USDExchangeRate = 7.3` 和 `Price = 7.3` 散落在多处，维护困难

## 改动范围

### 后端 (Go)

| 文件 | 改动说明 |
|------|----------|
| `common/constants.go` | `QuotaPerUnit` 注释从 `$0.002` 改为 `¥0.002`；`ChannelDisableThreshold` 注释改为人民币 |
| `setting/ratio_setting/model_ratio.go` | 移除 `USD2RMB`、`USD`、`RMB` 三个常量；所有 `* RMB` 和 `* USD` 引用替换为数值计算（`RMB = 68.493`, `USD = 500`） |
| `setting/operation_setting/payment_setting_old.go` | 移除 `Price = 7.3` 和 `USDExchangeRate = 7.3` 变量定义 |
| `controller/topup.go` | `getPayMoney` 移除 `* Price` 汇率转换，用户充 ¥N 直接对应 `N * QuotaPerUnit` 配额 |
| `controller/channel-billing.go` | Moonshot 余额不再除以 7.3（CNY→CNY 无需转换）；移除 `decimal` 和 `operation_setting` 导入 |
| `controller/ratio_sync.go` | `ratio_setting.USD` 替换为 `common.QuotaPerUnit` 相关计算 |
| `controller/billing.go` | CNY 分支直接 `quota / QuotaPerUnit`（不再乘汇率）；更新注释 |
| `controller/misc.go` | 移除 `usd_exchange_rate` 和 `price` 从 `/api/status` 返回 |
| `logger/logger.go` | `LogQuota` / `FormatQuota` 的 CNY 分支直接计算，默认分支改为 `¥` |
| `model/subscription.go` | `Currency` 字段默认值从 `'USD'` 改为 `'CNY'` |
| `model/main.go` | 数据库 schema `currency` 默认值改为 `'CNY'`（两处） |
| `model/option.go` | 移除 `Price` 和 `USDExchangeRate` 的 OptionMap 注册和解析 |
| `relay/helper/price.go` | 注释从 `$/1M tokens` 改为 `¥/1M tokens` |
| `pkg/billingexpr/expr.md` | 文档中所有 `$` 改为 `¥`，价格描述改为 `¥/1M tokens` |

### 前端 (React)

| 文件 | 改动说明 |
|------|----------|
| `web/src/helpers/render.jsx` | `getCurrencyConfig` / `renderQuota` 等所有 `quota_display_type` 默认值从 `'USD'` 改为 `'CNY'`，符号从 `$` 改为 `¥` |
| `web/src/helpers/data.js` | `setStatusData` 中 `quota_display_type` 回退默认值改为 `'CNY'` |
| `web/src/helpers/utils.jsx` | `formatDynamicPriceSummary` / `calculateModelPrice` 默认改为 CNY/¥ |
| `web/src/hooks/model-pricing/useModelPricingData.jsx` | `currency` 初始值改为 `'CNY'`，`siteDisplayType` 回退改为 `'CNY'` |
| `web/src/pages/Setting/Operation/SettingsGeneral.jsx` | 移除 `USDExchangeRate` 字段；默认 `quota_display_type` 改为 `'CNY'`；汇率展示逻辑简化 |
| `web/src/components/settings/OperationSetting.jsx` | 移除 `USDExchangeRate`；默认改为 `'CNY'` |
| `web/src/pages/Setting/Payment/SettingsPaymentGateway.jsx` | 移除 `Price` 字段及其表单输入 |
| `web/src/components/settings/PaymentSetting.jsx` | 移除 `Price` 字段 |

## 关键设计决策

1. **`QuotaPerUnit` 数值不变，语义翻转**：`500,000` 仍是精度因子，含义从 `$1 = 500,000 quota` 变为 `¥1 = 500,000 quota`。这意味着存量用户配额数值不需要迁移。

2. **模型定价系数不变**：`defaultModelRatio` 中的系数数值不变，因为 `ratio * QuotaPerUnit` 的计算链路不变，只是基准货币从美元变为人民币。

3. **Epay 国内支付简化**：去掉 `* 7.3` 汇率转换，用户充 ¥10 直接获得 `10 * 500,000 = 5,000,000` 配额。

4. **国际支付（Stripe / Waffo）保留 USD 语义**：这些渠道本身收的是美元，`TopUp.Money` 字段在 Stripe 路径下仍是 USD 值，前端展示时提示用户。

5. **硬编码汇率完全移除**：`USDExchangeRate = 7.3`、`Price = 7.3`、`USD2RMB/RMB` 常量及其所有引用全部清除。

## 保留不变的部分

- `QuotaPerUnit` 的数值 `500,000` 不变
- 所有模型定价系数数值不变
- 用户配额存储格式（int）不变
- TopUp 的 Amount/Money 字段类型不变
- Stripe / Waffo / Waffo Pancake 支付渠道代码保留
- `QuotaDisplayType` 四个选项（USD/CNY/TOKENS/CUSTOM）全部保留，仅默认值改为 CNY
- 订阅计划的 `Currency` 字段仍支持多种货币

## 影响评估

- **兼容性影响**：无 breaking change。存量用户配额数值不变（`QuotaPerUnit` 数值不变），只是语义解释从美元变为人民币。
- **数据迁移**：不需要。数据库中的配额字段为 int 类型，与基准货币无关。新建实例的 `currency` 字段默认为 `'CNY'`。
- **依赖变更**：无。移除了 `controller/channel-billing.go` 中未使用的 `decimal` 导入。
- **配置项移除**：`/api/option/` 中不再接受 `Price` 和 `USDExchangeRate` 配置项，前端管理页面中相关输入框已移除。

## 验证方式

1. **编译检查**：`go build ./...` 无编译错误 ✅
2. **残留引用检查**：搜索 `USDExchangeRate`、`Price = 7.3`、`USD2RMB`、`RMB =` 无残留 ✅
3. **前端构建**：需要运行 `cd web && bun run build` 验证前端无错误
4. **本地启动验证**：
   - 新用户注册后余额显示为 `¥` 而非 `$`
   - 模型定价页面显示 `¥` 价格
   - Epay 充值 ¥10 获得 5,000,000 配额（不再有 ×7.3）
   - 使用 API 后扣费金额正确
   - 日志中额度显示为 `¥`
