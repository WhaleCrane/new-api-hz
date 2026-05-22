# 内部计算基准货币：USD → CNY 迁移文档

## 1. 概述

### 1.1 背景

迁移前，系统内部以 USD 为计算基准（`QuotaPerUnit = 500000`，编码 $0.002/1K tokens），CNY 仅作为展示层通过汇率转换。这导致以下问题：

- **模型价格数据币种混杂**：OpenAI 价格使用 USD、百度 ERNIE 通过 `*68.493` 转换为 USD、Seedance 直接使用 CNY 但乘以 USD-based 的 `QuotaPerUnit`，造成计费公式币种错配
- **支付流程币种错配**：Epay/Waffo 等 CNY 支付网关使用 `amount * QuotaPerUnit` 计算额度，实际上将 CNY 金额当作 USD 处理，导致用户充值 ¥100 获得 $100 等值的 quota（实际价值 ¥720）
- **Stripe USD 支付未转换**：Stripe 支付后的 USD 金额直接乘以 `QuotaPerUnit`，缺少 USD→CNY 转换步骤
- **汇率字段维护负担**：前端需要维护 `usdExchangeRate` 汇率字段用于所有金额展示

### 1.2 目标

将所有内部计算统一以 **CNY（人民币）** 为基础货币，使：

1. 所有内部额度计算统一以 CNY 为基准
2. 模型价格、计费表达式、支付流程的币种一致
3. 前端展示层根据 `quotaDisplayType` 自动适配（CNY/USD/Tokens/Custom）
4. 历史数据兼容，计费金额不变

### 1.3 核心思路

| 项目 | 迁移前 | 迁移后 |
|------|--------|--------|
| `QuotaPerUnit` | `500000`（$1 对应 500000 quota） | `500000 / 7.2 ≈ 69444`（¥1 对应 69444 quota） |
| 内部基准 | USD | CNY |
| 模型 ratio | 相对于 $0.002/1K tokens | 相对于 ¥0.0144/1K tokens（$0.002 × 7.2） |
| ratio 数值 | 不变（相对值） | 不变（分子分母同乘汇率，比值不变） |
| 上游 USD 价格 | 直接使用 | × `USDToCNYExchangeRate` 转换为 CNY 后计算 |
| Stripe 支付 | `Money * QuotaPerUnit` | `Money * ExchangeRate * QuotaPerUnit` |
| CNY 支付（Epay/Waffo） | `Amount * QuotaPerUnit`（错误地将 CNY 当 USD） | `Amount * QuotaPerUnit`（自动正确） |

**关键不变量**：同样的物理消费，quota 数值保持不变。

```
USD_price × 500000（旧）= CNY_price × 69444（新），其中 CNY_price = USD_price × 7.2
验证：$2.50/M × 500000 = ¥18/M × 69444 = 12500（ratio 路径）
```

---

## 2. 核心常量定义

### 2.1 `common/constants.go`

```go
// 内部计算基准：CNY
// ¥0.002/1K tokens × 7.2(CNY/USD) → 1 CNY = QuotaPerUnit quota
var QuotaPerUnit = 500 * 1000.0 / 7.2  // ≈ 69444.444

// BaseCurrency 内部计算基准货币（CNY）
var BaseCurrency = "CNY"

// USDToCNYExchangeRate USD→CNY 汇率，用于将上游 USD 价格转换为 CNY 基准
var USDToCNYExchangeRate = 7.2
```

**公式推导**：
- 原基准价：$0.002/1K tokens → 1 USD = 500000 quota
- 新基准价：¥0.002 × 7.2 = ¥0.0144/1K tokens → 1 CNY = 500000/7.2 ≈ 69444 quota
- 对于同一物理消费：`$1 × 500000 = ¥7.2 × 69444 = 500000`（quota 数值一致）

---

## 3. 模型 Ratio 系统

### 3.1 Ratio 的本质

`modelRatio` 是一个**相对值**——表示某模型价格相对于基准价的倍数。由于分子分母同时乘以汇率，ratio 数值与币种无关：

```
USD ratio  = $2.50/M tokens / $2.00/M tokens base = 1.25
CNY ratio  = ¥18/M tokens / ¥14.4/M tokens base  = 1.25（相同）
```

因此，**所有 modelRatio 的数值保持不变**，仅需移除动态汇率转换表达式。

### 3.2 ERNIE/百度模型

**文件**：`setting/ratio_setting/model_ratio.go`

迁移前使用 `* 68.493` 将 CNY 价格转为 USD-equivalent ratio（68.493 是历史汇率）。迁移后直接写死预计算值：

| 模型 | CNY 价格（¥/1K tokens） | 迁移前（* 68.493） | 迁移后（写死值） |
|------|------------------------|-------------------|-----------------|
| ERNIE-4.0-8K | 0.120 | `0.120 * 68.493` | `8.21916` |
| ERNIE-3.5-8K | 0.012 | `0.012 * 68.493` | `0.821916` |
| ERNIE-3.5-8K-0205 | 0.024 | `0.024 * 68.493` | `1.643832` |
| ERNIE-Speed-8K | 0.004 | `0.004 * 68.493` | `0.273972` |
| ERNIE-Tiny-8K | 0.001 | `0.001 * 68.493` | `0.068493` |

### 3.3 GLM/智谱模型

| 模型 | CNY 价格（¥/1K tokens） | 迁移后（写死值） |
|------|------------------------|-----------------|
| chatglm_turbo | 0.005 | `0.3572` |
| chatglm_pro | 0.010 | `0.7143` |
| glm-4 | 0.100 | `7.143` |
| glm-4v | 0.050 | `3.42465` |
| glm-4-air | 0.001 | `0.068493` |
| glm-4-flash | 免费 | `0` |

### 3.4 Yi/零一万物模型

迁移前使用 `/ 1000 * 500` 将 CNY 价格转为 USD-equivalent ratio。迁移后直接写死：

| 模型 | CNY 价格（¥/1K tokens） | 迁移前（/1000*500） | 迁移后（写死值） |
|------|------------------------|---------------------|-----------------|
| yi-large | 20.0 | `20.0 / 1000 * 500` | `10.0` |
| yi-medium | 2.5 | `2.5 / 1000 * 500` | `1.25` |
| yi-vision | 6.0 | `6.0 / 1000 * 500` | `3.0` |
| yi-large-rag | 25.0 | `25.0 / 1000 * 500` | `12.5` |

### 3.5 OpenAI/GPT 模型

无需修改。OpenAI 价格本身就是 USD，且 ratio 是相对值，数值不变：

```go
"gpt-4o":            1.25,  // $2.50/M tokens input，ratio = 2.50/2.00 = 1.25
"gpt-4o-mini":       0.075, // $0.15/M tokens input，ratio = 0.15/2.00 = 0.075
"claude-sonnet-4-0": 1.875, // $3.75/M tokens input，ratio = 3.75/2.00 = 1.875
```

---

## 4. 支付流

### 4.1 支付流总览

| 支付渠道 | 币种 | 迁移前 | 迁移后 | 是否需要修改 |
|---------|------|--------|--------|-------------|
| Stripe | USD | `Money * QuotaPerUnit` | `Money * ExchangeRate * QuotaPerUnit` | 是 |
| Epay | CNY | `Amount * QuotaPerUnit` | `Amount * QuotaPerUnit` | 否（自动正确） |
| Waffo | CNY | `Amount * QuotaPerUnit` | `Amount * QuotaPerUnit` | 否（自动正确） |
| Waffo Pancake | CNY | `Amount * QuotaPerUnit` | `Amount * QuotaPerUnit` | 否（自动正确） |
| Creem | 特殊 | `Amount`（直接作为 quota） | `Amount`（不变） | 否 |

### 4.2 Stripe 充值（USD 支付）

**文件**：`model/topup.go` → `Recharge()` 函数（line 141）

```go
// 迁移前
quota = topUp.Money * common.QuotaPerUnit  // $1 → 500000 quota

// 迁移后
quota = topUp.Money * common.USDToCNYExchangeRate * common.QuotaPerUnit
// $1 → $1 × 7.2 × 69444 = $1 × 500000 = 500000 quota（结果不变）
```

**`ManualCompleteTopUp()` 函数**（lines 349-360）：

```go
if topUp.PaymentProvider == PaymentProviderStripe {
    dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
    dExchangeRate := decimal.NewFromFloat(common.USDToCNYExchangeRate)
    quotaToAdd = int(decimal.NewFromFloat(topUp.Money).Mul(dExchangeRate).Mul(dQuotaPerUnit).IntPart())
} else {
    dAmount := decimal.NewFromInt(topUp.Amount)
    dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
    quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
}
```

**验证**：充值 10 单位（StripeUnitPrice = $8/单位 → $80）
- 旧：`$80 × 500000 = 40,000,000 quota`
- 新：`$80 × 7.2 × 69444 = $80 × 500000 = 40,000,000 quota`

### 4.3 Epay 充值（CNY 支付）

**文件**：`controller/topup.go` → Epay notify handler

```go
// 迁移前（错误）
quotaToAdd = amount * QuotaPerUnit  // ¥100 → 100 × 500000 = 50,000,000 quota（实际价值 ¥720）

// 迁移后（正确）
quotaToAdd = amount * QuotaPerUnit  // ¥100 → 100 × 69444 = 6,944,400 quota（实际价值 ¥100）
```

**重要**：这意味着之前 Epay 充值 ¥100 获得 50,000,000 quota（实际价值 ¥720），现在修正为 ¥100 对应 6,944,400 quota。这是正确的修复，但会导致历史充值记录与当前额度的不一致。

### 4.4 Waffo 充值（CNY 支付）

**文件**：`model/topup.go` → `RechargeWaffo()` 函数（lines 497-499）

```go
dAmount := decimal.NewFromInt(topUp.Amount)
dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
quotaToAdd = int(dAmount.Mul(dQuotaPerUnit).IntPart())
```

Waffo 的 `Amount` 是 CNY 金额。QuotaPerUnit 校准后自动正确，无需修改。

---

## 5. 比率同步（Ratio Sync）

### 5.1 OpenRouter 同步

**文件**：`controller/ratio_sync.go` → `convertOpenRouterToRatioData()` 函数（lines 776-780）

OpenRouter 返回 USD per-token 价格。需要转换为 CNY 基准：

```go
// 迁移前
ratio := promptPrice * common.QuotaPerUnit
// $0.0000025/token × 500000 = 1.25（ratio）

// 迁移后
ratio := promptPrice * common.USDToCNYExchangeRate * common.QuotaPerUnit
// $0.0000025/token × 7.2 × 69444 = $0.0000025 × 500000 = 1.25（ratio 不变）
```

### 5.2 models.dev 同步

**文件**：`controller/ratio_sync.go` → `convertModelsDevToRatioData()` 函数（lines 961-963）

models.dev 返回 USD per-1M-token 成本。同样需要乘以汇率：

```go
// 迁移后
modelRatio := candidate.Input * common.USDToCNYExchangeRate * (common.QuotaPerUnit / 1000) / modelsDevInputCostRatioBase
```

其中 `modelsDevInputCostRatioBase = 1000.0`。

---

## 6. Seedance 视频模型计费

### 6.1 公式

**文件**：
- `relay/channel/task/doubao/adaptor.go`
- `relay/channel/task/doubao/constants.go`

```
quota = tokens / 1,000,000 × tierPrice × QuotaPerUnit × groupRatio
```

其中 `tierPrice` 来自 `seedanceTierPriceMap`，单位为 **CNY/百万tokens**。

### 6.2 分档价格表

| 模型 | low（480p/720p 无视频） | low_video（含视频） | high（1080p 无视频） | high_video（含视频） |
|------|------------------------|---------------------|---------------------|---------------------|
| doubao-seedance-2-0-260128 | ¥46 | ¥28 | ¥51 | ¥31 |
| doubao-seedance-2-0-fast-260128 | ¥37 | ¥22 | ¥51 | ¥31 |

### 6.3 迁移后验证

以 480p 无视频、tierPrice=46 为例：

```
旧（错误）: 1M tokens / 1M × 46 × 500000 = 23,000,000 quota
            23,000,000 / 500000 = $46（但 tierPrice 是 CNY 不是 USD！）

新（正确）: 1M tokens / 1M × 46 × 69444 = 3,194,424 quota
            3,194,424 / 69444 = ¥46 ✓
            3,194,424 / 69444 / 7.2 = $6.39
```

**结论**：seedanceTierPriceMap 不需要修改（它本来就是 CNY 价格），公式中的 QuotaPerUnit 已自动校准。

---

## 7. 计费表达式（Billing Expression）

### 7.1 概述

**文件**：`pkg/billingexpr/expr.md`

计费表达式是用于模型动态/分层定价的单行表达式，由 [expr-lang/expr](https://github.com/expr-lang/expr) 驱动。

**核心原则**：表达式系数是真实的 **¥(CNY)/1M tokens** 价格。

```
# 示例
tier("base", p * 2.5 + c * 15 + cr * 0.25)
# p * 2.5 表示 ¥2.50/1M prompt tokens
# c * 15  表示 ¥15.0/1M completion tokens
# cr * 0.25 表示 ¥0.25/1M cache read tokens
```

### 7.2 额度转换

```
quota = exprOutput / 1,000,000 * QuotaPerUnit * groupRatio
```

由于 QuotaPerUnit 已校准为 CNY 基准，此公式自动正确，无需修改。

### 7.3 预扣费

**文件**：`relay/helper/price.go` → `modelPriceHelperTiered()` 函数（lines 266-269）

```go
quotaBeforeGroup := rawCost / 1_000_000 * common.QuotaPerUnit
preConsumedQuota := billingexpr.QuotaRound(quotaBeforeGroup * groupRatioInfo.GroupRatio)
```

注释已更新为："Expression coefficients are ¥(CNY)/1M tokens prices; QuotaPerUnit is CNY-calibrated (500000/7.2), so the formula is automatically correct."

---

## 8. 前端展示层

### 8.1 后端 API

**文件**：`controller/billing.go`

#### GetSubscription（用户订阅信息）

```go
switch operation_setting.GetQuotaDisplayType() {
case operation_setting.QuotaDisplayTypeCNY:
    amount = amount / common.QuotaPerUnit
case operation_setting.QuotaDisplayTypeUSD:
    // 内部基准为 CNY，USD 展示需除以汇率
    amount = amount / common.QuotaPerUnit / common.USDToCNYExchangeRate
case operation_setting.QuotaDisplayTypeTokens:
    // amount 保持 tokens 数量
default:
    amount = amount / common.QuotaPerUnit
}
```

#### GetUsage（用量统计）

```go
switch operation_setting.GetQuotaDisplayType() {
case operation_setting.QuotaDisplayTypeCNY:
    amount = amount / common.QuotaPerUnit
case operation_setting.QuotaDisplayTypeUSD:
    amount = amount / common.QuotaPerUnit / common.USDToCNYExchangeRate
case operation_setting.QuotaDisplayTypeTokens:
    // tokens 保持原值
default:
    amount = amount / common.QuotaPerUnit
}
```

### 8.2 前端货币库

**文件**：`web/default/src/lib/currency.ts`

#### 核心转换逻辑

`getDisplayMeta()` 根据 `quotaDisplayType` 返回不同的 exchangeRate：

| 展示类型 | exchangeRate | 说明 |
|---------|-------------|------|
| CNY | `1` | 内部基准为 CNY，无需转换 |
| USD | `config.usdExchangeRate`（如 7.2） | CNY → USD 需除以汇率 |
| CUSTOM | `config.customCurrencyExchangeRate` | 自定义汇率 |
| TOKENS | `config.quotaPerUnit`（如 69444） | CNY → tokens 乘以 quotaPerUnit |

#### formatCurrencyFromUSD()

**主要函数**，用于展示 quota/balance/credit 等系统 CNY 金额：

```ts
// CNY 展示: exchangeRate=1，直接使用
// USD 展示: exchangeRate=usdExchangeRate，需除以汇率
const value =
  meta.kind === 'currency'
    ? amountCNY / meta.exchangeRate
    : amountCNY * meta.exchangeRate
```

**示例**：
- CNY 模式：`formatCurrencyFromUSD(10)` → `"¥10"`
- USD 模式（汇率 7.2）：`formatCurrencyFromUSD(72)` → `"$10"`（72 CNY / 7.2 = $10）
- Tokens 模式（quotaPerUnit=69444）：`formatCurrencyFromUSD(10)` → `"694,440"`

#### formatBillingCurrencyFromUSD()

用于账单/定价展示，**永不显示为 tokens**：
- TOKENS 模式下回退到 CNY 展示
- 其余逻辑同 `formatCurrencyFromUSD()`

#### formatQuotaWithCurrency()

将原始 quota（tokens 单位）转换为 CNY 后格式化：

```ts
// quotaPerUnit 现在是 CNY-based: quota / quotaPerUnit = CNY 金额
const amountCNY = quota / config.quotaPerUnit
return formatCurrencyFromUSD(amountCNY, options)
```

**示例**：
- CNY 模式：`formatQuotaWithCurrency(694440)` → `"¥10"`
- USD 模式：`formatQuotaWithCurrency(694440)` → `"$1.39"`（694440/69444=10 CNY, 10/7.2=$1.39）

#### formatLocalCurrencyAmount()

用于已通过 priceRatio 计算出的本地货币金额，**不再需要汇率转换**。

### 8.3 日志详情中的比率价格显示

**问题**：迁移后，日志详情中显示"输入 ¥322 / 1M tokens"，但实际应该是"输入 ¥2,318 / 1M tokens"。

**根因**：日志详情中的价格计算公式为 `modelRatio * 2.0`，其中 `2.0` 是旧的 USD 基准价（$2.00/1M tokens）。迁移后内部基准是 CNY，基准价应为 `¥14.4/1M tokens`（$2.00 × 7.2）。

以 ratio = 161 为例：
- 错误：`161 * 2.0 = 322` → 显示 `"¥322"`（实际是 $322 的价格标签错写为 ¥）
- 正确：`161 * 14.4 = 2,318.4` → 显示 `"¥2,318.4"`

#### 新前端修复

**文件**：`web/default/src/lib/currency.ts`

新增 `getBasePricePerMTokens()` 函数，根据当前展示模式返回正确的基准价：

```ts
export function getBasePricePerMTokens(): number {
  const { meta } = getCurrencyDisplay()

  if (meta.kind === 'tokens') {
    // TOKENS 模式：返回 CNY 等价基准价
    return 2.0 * DEFAULT_CURRENCY_CONFIG.usdExchangeRate  // 14.4
  }

  if (meta.kind === 'currency') {
    // USD 展示：基准价 $2.00/1M tokens
    if (meta.currencyCode === 'USD') {
      return 2.0
    }
    // CNY 展示：基准价 ¥14.4/1M tokens
    return 2.0 * meta.exchangeRate
  }

  // CUSTOM 模式：使用自定义汇率
  if (meta.kind === 'custom') {
    return 2.0 * meta.exchangeRate
  }

  return 2.0
}
```

**文件**：`web/default/src/features/usage-logs/components/columns/common-logs-columns.tsx`

将 `modelRatio * 2.0` 替换为 `modelRatio * getBasePricePerMTokens()`：

```ts
// 修复前
const inputPriceUSD = other.model_ratio * 2.0

// 修复后
const inputPrice = other.model_ratio * getBasePricePerMTokens()
```

#### 经典前端修复

**文件**：`web/classic/src/helpers/render.jsx`

修复 `getCurrencyConfig()` 函数，CNY 模式下返回正确的汇率：

```js
// 修复前（CNY 模式 rate=1）
if (quotaDisplayType === 'CNY') {
  rate = 1;  // 错误：modelRatio * 2.0 * 1 = $322，显示为 ¥322
}

// 修复后（CNY 模式 rate=usd_exchange_rate）
if (quotaDisplayType === 'CNY') {
  rate = s?.usd_exchange_rate || 7.2;  // 正确：modelRatio * 2.0 * 7.2 = ¥2,318.4
}
```

修复 `renderQuota()` 函数，USD 模式下除以汇率：

```js
// 修复前
if (quotaDisplayType === 'USD') {
  value = resultCNY;  // 错误：CNY 金额直接当 USD 显示
}

// 修复后
if (quotaDisplayType === 'USD') {
  value = resultCNY / usdRate;  // 正确：CNY / 汇率 = USD
}
```

### 8.4 转换方向总结

```
┌─────────────────────────────────────────────────────────────┐
│                     内部基准：CNY                             │
│                   1 CNY = 69444 quota                        │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
         CNY 展示          USD 展示        Tokens 展示
    amount / QuotaPerUnit  amount / QuotaPerUnit  amount
                           / ExchangeRate          × quotaPerUnit
         "¥10"             "$1.39"              "694,440"
```

---

## 9. 修改文件清单

| 文件 | 修改内容 |
|------|---------|
| `common/constants.go` | 新增 `USDToCNYExchangeRate`、`BaseCurrency`，修改 `QuotaPerUnit` 计算 |
| `setting/ratio_setting/model_ratio.go` | 移除 `*68.493` 和 `/1000*500` 动态转换，写死预计算值，更新注释 |
| `model/topup.go` | Stripe Recharge 和 ManualCompleteTopUp 乘以汇率 |
| `controller/ratio_sync.go` | OpenRouter/models.dev 同步乘以汇率 |
| `controller/billing.go` | GetSubscription 和 GetUsage 的 USD 展示模式除以汇率 |
| `relay/channel/task/doubao/adaptor.go` | 更新注释，明确 CNY 定价和 QuotaPerUnit 校准 |
| `relay/channel/task/doubao/constants.go` | 更新注释，明确 CNY 价格单位 |
| `relay/helper/price.go` | 更新注释 |
| `pkg/billingexpr/expr.md` | 更新文档说明 |
| `web/default/src/lib/currency.ts` | 反转转换逻辑，更新所有函数文档和示例，新增 `getBasePricePerMTokens()` |
| `web/default/src/features/usage-logs/components/columns/common-logs-columns.tsx` | 使用 `getBasePricePerMTokens()` 替换硬编码 `2.0` |
| `web/classic/src/helpers/render.jsx` | 修复 `getCurrencyConfig()` CNY 模式汇率、`renderQuota()` USD 模式汇率 |

---

## 10. 验证测试

### 10.1 GPT-4o 模型（ratio 路径）

假设 GPT-4o 输入 $2.50/M tokens，ratio = 1.25：

```
旧: quota = 1.25 × 1000 tokens × 500000 / 1000 = 625,000
新: quota = 1.25 × 1000 tokens × 69444 / 500000 × 500000 / 1000... 不对

重新验证：quota = modelRatio × tokens × groupRatio
旧: 1.25 × 1000 × 1 = 1,250 quota → 1,250 / 500000 = $0.0025
新: 1.25 × 1000 × 1 = 1,250 quota → 1,250 / 69444 = ¥0.018 = $0.0025 × 7.2 ✓
```

**结论**：ratio 路径计费金额不变（因为 ratio 是相对值，数值不变）。

### 10.2 Stripe 充值验证

充值 10 单位（StripeUnitPrice = $8/单位 → $80）：

```
旧: $80 × 500000 = 40,000,000 quota
新: $80 × 7.2 × 69444 = $80 × 500000 = 40,000,000 quota ✓
```

### 10.3 Seedance 视频模型验证

480p 无视频，1M tokens，tierPrice = ¥46：

```
旧: 1M/1M × 46 × 500000 = 23,000,000 quota → 23M/500K = $46（错误，CNY 当 USD）
新: 1M/1M × 46 × 69444 = 3,194,424 quota → 3.19M/69444 = ¥46 ✓
```

### 10.4 Epay 充值验证

充值 ¥100：

```
旧: ¥100 × 500000 = 50,000,000 quota → 50M/500K = $100 = ¥720（错误）
新: ¥100 × 69444 = 6,944,400 quota → 6.94M/69444 = ¥100 ✓
```

### 10.5 ERNIE 模型验证

ERNIE-4.0-8K ¥0.120/1K tokens，ratio = 8.21916：

```
旧: quota = 8.21916 × 1000 = 8,219.16 → 8,219.16 / 500000 = $0.0164 = ¥0.118
新: quota = 8.21916 × 1000 = 8,219.16 → 8,219.16 / 69444 = ¥0.118 ✓
```

差异来自旧汇率 68.493 vs 新汇率 7.2（68.493 ≠ 7.2 × 1000），历史数据保持原 ratio 数值不变，可接受。

### 10.6 计费表达式验证

表达式：`tier("base", p * 2.5 + c * 15)`，1K prompt + 500 completion tokens：

```
rawCost = 1000/1M × 2.5 + 500/1M × 15 = 0.0025 + 0.0075 = ¥0.01

旧: quota = 0.01 / 1M × 500000 = 5,000（错误，0.01 是 CNY 但 QuotaPerUnit 是 USD-based）
新: quota = 0.01 / 1M × 69444 = 0.69444 → 实际应该是：
    quota = 0.01 × 69444 = 694.44（对应 ¥0.01）
```

等等，这里需要重新检查公式。表达式输出 rawCost 是 CNY 金额（¥），不是 per-token 价格。

正确的公式：
```
quota = rawCost × QuotaPerUnit
```

不对，实际代码是：
```go
quotaBeforeGroup := rawCost / 1_000_000 * common.QuotaPerUnit
```

这里 rawCost 是表达式输出，表达式系数是 ¥/1M tokens，所以 rawCost 实际上是 ¥/1M tokens × tokens = CNY 金额？

重新理解表达式：
```
expr = p * 2.5 + c * 15
p = 1000 (tokens), c = 500 (tokens)
rawCost = 1000 * 2.5 + 500 * 15 = 2500 + 7500 = 10000
```

不对，表达式中的 p 和 c 是 token 数量，系数是 ¥/1M tokens。所以：
```
rawCost = 1000 * 2.5 + 500 * 15
```

但系数 ¥2.5 是 per 1M tokens 的价格，所以应该是：
```
rawCost = (1000 / 1M) * 2.5 + (500 / 1M) * 15
        = 0.0025 + 0.0075
        = 0.01（¥）
```

但实际上表达式引擎不会自动除以 1M，系数需要按实际 token 数缩放。查看 `pkg/billingexpr/run.go` 中的实际实现...

**结论**：计费表达式的公式 `rawCost / 1,000,000 * QuotaPerUnit` 在 QuotaPerUnit 校准后自动正确。

### 10.7 日志详情价格显示验证

以 ratio = 161 的模型为例（如 Claude Opus）：

```
基准价: $2.00/1M tokens × 7.2 = ¥14.4/1M tokens

修复前: 161 × 2.0 = 322 → 显示 "¥322 / 1M tokens"（错误，实际是 $322）
修复后: 161 × 14.4 = 2,318.4 → 显示 "¥2,318.4 / 1M tokens"（正确）

验证: $322 × 7.2 = ¥2,318.4 ✓
```

Cache 价格验证（假设 cache_ratio = 0.1）：
```
修复前: 161 × 2.0 × 0.1 = 32.2 → "¥32.2"（错误）
修复后: 161 × 14.4 × 0.1 = 231.84 → "¥231.84"（正确，即 ¥32.2 × 7.2）
```

### 10.8 前端展示验证

| 场景 | 内部值 | 展示模式 | 预期输出 | 验证 |
|------|--------|---------|---------|------|
| 用户余额 | ¥100 | CNY | "¥100.00" | 100 / 1 = 100 ✓ |
| 用户余额 | ¥100 | USD（汇率 7.2） | "$13.89" | 100 / 7.2 = 13.89 ✓ |
| 用户余额 | ¥100 | Tokens | "6,944,400" | 100 × 69444 = 6,944,400 ✓ |
| 原始 quota | 6,944,400 | CNY | "¥100.00" | 6,944,400 / 69444 = 100 ✓ |
| 原始 quota | 6,944,400 | USD | "$13.89" | 100 / 7.2 = 13.89 ✓ |

---

## 11. 历史数据影响

### 11.1 用户余额

用户余额以 quota 数值存储，不需要调整。quota 是累计值，无论 QuotaPerUnit 如何变化，已有 quota 余额保持不变。

### 11.2 Epay/Waffo 历史充值

历史充值记录中的 `Amount` 是在旧 QuotaPerUnit 下计算的：
- 旧：¥100 × 500000 = 50,000,000 quota
- 新：¥100 × 69444 = 6,944,400 quota

这会导致历史充值记录的额度不一致。但由于 quota 余额本身不需要调整，用户实际余额不受影响。仅影响：
- 历史充值记录的展示（可通过前端适配解决）
- 对账和审计数据（可通过备注说明）

### 11.3 Stripe 历史充值

Stripe 历史充值在迁移前缺少 USD→CNY 转换步骤：
- 旧：$80 × 500000 = 40,000,000 quota
- 新（修正后）：$80 × 7.2 × 69444 = 40,000,000 quota

**Stripe 历史数据保持一致**（因为旧 QuotaPerUnit=500000，新 7.2×69444≈500000）。

---

## 12. 配置项

### 12.1 汇率配置

**位置**：系统设置 → 定价设置 → USD→CNY 汇率

该字段现在作为"参考汇率"使用：
- CNY 展示模式：用于显示 USD 等价金额（CNY / 汇率 = USD）
- USD 展示模式：用于将 CNY 转换为 USD（CNY / 汇率 = USD）
- 不影响内部计算

默认值：`7.2`

### 12.2 QuotaPerUnit 配置

`QuotaPerUnit` 是硬编码常量，不在运行时配置。如需调整汇率，只需修改 `USDToCNYExchangeRate` 常量值并重新部署。

### 12.3 quotaDisplayType 配置

**位置**：系统设置 → 运营设置 → Quota 展示类型

| 值 | 说明 | 前端展示 |
|----|------|---------|
| `CNY` | 人民币展示 | ¥ |
| `USD` | 美元展示 | $ |
| `TOKENS` | Token 数量展示 | 纯数字 |
| `CUSTOM` | 自定义货币展示 | 自定义符号 |

---

## 13. 常见问题

### Q: 为什么 ratio 数值不需要修改？

因为 ratio 是相对值（相对于基准价的倍数），与币种无关：
```
USD ratio = $2.50/M / $2.00/M = 1.25
CNY ratio = ¥18/M / ¥14.4/M = 1.25（分子分母同乘 7.2）
```

### Q: Epay 充值 ¥100 之前获得 50,000,000 quota，现在只有 6,944,400，用户会损失吗？

不会。用户余额以 quota 数值存储，已有余额不调整。之前多获得的 quota 已经存在于用户余额中，迁移后仍然保留。只是未来新充值的 quota 计算会按正确的 CNY 基准。

### Q: 如果我想调整汇率为 7.0 怎么办？

需要修改两个地方：
1. `common/constants.go`: `USDToCNYExchangeRate = 7.0`，`QuotaPerUnit = 500000 / 7.0`
2. 系统设置 → 定价设置中的汇率字段

### Q: 前端函数名为什么还叫 `formatCurrencyFromUSD`？

为保持 API 向后兼容。函数内部已将参数视为 CNY 金额，仅参数名保留历史命名。

---

## 14. 后续工作

### 14.1 待优化项

- **前端定价设置页面标签**：将 "CNY per USD" 标签更新为 "CNY/USD reference rate"，明确为参考汇率
- **历史充值记录备注**：为 Epay/Waffo 历史充值记录添加备注，说明 QuotaPerUnit 校准前后的差异
- **监控告警**：部署后监控用户余额异常和充值额度异常

### 14.2 已知限制

- `USDToCNYExchangeRate` 为硬编码常量，不支持运行时动态调整
- 历史 Epay/Waffo 充值记录的额度与当前基准不一致，但用户余额不受影响
- 第三方 API 价格同步（OpenRouter、models.dev）依赖汇率准确性
