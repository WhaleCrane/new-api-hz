# 2026-05-21 修改日志

## 一、文档修改

### 1. `docs/volcengine-api-doc.md` — 为所有接口补充 `project_name` 可选字段

**背景**：对比 `F:/真人人像/` 下火山官方文档，发现 `volcengine-api-doc.md` 中所有接口的请求体都缺少 `project_name` 可选字段。

**修改内容**：为 s1/s2 路径共 18 个接口（1 个已有无需修改）的请求体补充 `project_name` 字段，统一格式：

```
| `project_name` | string | 否 | 项目名称，默认 `default` |
```

**涉及接口**（20 个，其中 1 个已有）：

| 路径 | 接口 | 修改 |
|------|------|------|
| `/s1/asset/visual-validate/session` | CreateVisualValidateSession | 加在 `callback_url` 之后 |
| `/s1/asset/visual-validate/result` | GetVisualValidateResult | 加在 `byted_token` 之后 |
| `/s1/asset/asset-groups/list` | ListAssetGroups | 加在 `sort_order` 之后 |
| `/s1/asset/asset-groups/get` | GetAssetGroup | 加在 `id` 之后 |
| `/s1/asset/asset-groups/update` | UpdateAssetGroup | 加在 `description` 之后 |
| `/s1/asset/asset-groups/delete` | DeleteAssetGroup | 加在 `id` 之后 |
| `/s1/asset/assets/create` | CreateAsset | 加在 `name` 之后 |
| `/s1/asset/assets/list` | ListAssets | 加在 `sort_order` 之后 |
| `/s1/asset/assets/get` | GetAsset | 加在 `id` 之后 |
| `/s1/asset/assets/update` | UpdateAsset | 加在 `name` 之后 |
| `/s1/asset/assets/delete` | DeleteAsset | 加在 `id` 之后 |
| `/s2/aigc-asset/asset-groups/create` | CreateAIGCAssetGroup | 已有，无需修改 |
| `/s2/aigc-asset/asset-groups/list` | ListAIGCAssetGroups | 加在 `page_size` 之后 |
| `/s2/aigc-asset/asset-groups/get` | GetAIGCAssetGroup | 加在 `id` 之后 |
| `/s2/aigc-asset/asset-groups/update` | UpdateAIGCAssetGroup | 加在 `description` 之后 |
| `/s2/aigc-asset/asset-groups/delete` | DeleteAIGCAssetGroup | 加在 `id` 之后 |
| `/s2/aigc-asset/assets/create` | CreateAIGCAsset | 加在 `name` 之后 |
| `/s2/aigc-asset/assets/get` | GetAIGCAsset | 加在 `id` 之后 |
| `/s2/aigc-asset/assets/update` | UpdateAIGCAsset | 加在 `name` 之后 |
| `/s2/aigc-asset/assets/delete` | DeleteAIGCAsset | 加在 `id` 之后 |

---

## 二、代码修改

### 2.1 修复用户重复创建素材组不记录映射的问题

**背景**：用户多次调用 `/s2/aigc-asset/asset-groups/create` 创建素材组时，`aigc_asset_group_mappings` 表只记录了首次创建的 `group_id`，后续创建的素材组 ID 未被记录。s1 路径的 `CreateUserAssetMapping` 存在同样问题。

**根因**：`CreateAIGCAssetMapping` 和 `CreateUserAssetMapping` 检查映射是否存在时，用 `GetUserAIGCAssetGroupMapping(userId)` 只查 `user_id`，只要该用户有任何一条映射就提前返回，不判断具体 `group_id` 是否已存在。

#### 修改 1：`model/aigc_asset_group_mapping.go` — 新增 `GetAIGCAssetGroupMapping` 方法

```go
// GetAIGCAssetGroupMapping 按 user_id + group_id 查询映射（用于判重）
func GetAIGCAssetGroupMapping(userId int, groupId string) (*AIGCAssetGroupMapping, error) {
    mapping := &AIGCAssetGroupMapping{}
    err := DB.Where("user_id = ? AND group_id = ?", userId, groupId).First(mapping).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return mapping, nil
}
```

#### 修改 2：`model/asset_group_mapping.go` — 新增 `GetAssetGroupMapping` + `DeleteAssetGroupMapping` 方法

```go
// GetAssetGroupMapping 按 user_id + group_id 查询映射（用于判重）
func GetAssetGroupMapping(userId int, groupId string) (*AssetGroupMapping, error) {
    mapping := &AssetGroupMapping{}
    err := DB.Where("user_id = ? AND group_id = ?", userId, groupId).First(mapping).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return mapping, nil
}

// DeleteAssetGroupMapping 删除用户与 Asset Group 的映射
func DeleteAssetGroupMapping(userId int, groupId string) error {
    return DB.Where("user_id = ? AND group_id = ?", userId, groupId).Delete(&AssetGroupMapping{}).Error
}
```

#### 修改 3：`service/aigc_asset.go` — `CreateAIGCAssetMapping` 判重逻辑

```go
// Before
existing, err := model.GetUserAIGCAssetGroupMapping(userId)

// After
existing, err := model.GetAIGCAssetGroupMapping(userId, groupId)
```

#### 修改 4：`service/asset.go` — `CreateUserAssetMapping` 判重逻辑

```go
// Before
existing, err := model.GetUserAssetGroupMapping(userId)

// After
existing, err := model.GetAssetGroupMapping(userId, groupId)
```

---

### 2.2 修复删除 AIGC 素材组时映射清理错误被忽略的问题

**背景**：`DeleteAIGCAssetGroup` 用 `_ =` 忽略了 `DeleteAIGCAssetGroupMapping` 的返回值，清理失败时不会报错。

#### 修改 5：`service/aigc_asset.go` — `DeleteAIGCAssetGroup`

```go
// Before
_ = model.DeleteAIGCAssetGroupMapping(userId, id)

// After
if err := model.DeleteAIGCAssetGroupMapping(userId, id); err != nil {
    return nil, fmt.Errorf("failed to delete asset group mapping: %w", err)
}
```

---

### 2.3 修复 s1 路径删除素材组时不清理本地映射的问题

**背景**：`DeleteAssetGroup`（真人素材路径）调用火山 DeleteAssetGroup 后没有清理本地 `asset_group_mappings` 映射，与 s2 路径行为不一致。

#### 修改 6：`service/asset.go` — `DeleteAssetGroup`

```go
// Before
return CallArkAPI(ctx, channel, "DeleteAssetGroup", reqBody)

// After
resp, err := CallArkAPI(ctx, channel, "DeleteAssetGroup", reqBody)
if err != nil {
    return nil, err
}

// 删除成功后，清理本地映射
if err := model.DeleteAssetGroupMapping(userId, id); err != nil {
    return nil, fmt.Errorf("failed to delete asset group mapping: %w", err)
}

return resp, nil
```

---

## 三、涉及文件清单

| 文件 | 修改类型 | 说明 |
|------|---------|------|
| `docs/volcengine-api-doc.md` | 新增字段 | 为 20 个接口补充 `project_name` 可选字段 |
| `model/aigc_asset_group_mapping.go` | 新增方法 | `GetAIGCAssetGroupMapping(userId, groupId)` |
| `model/asset_group_mapping.go` | 新增方法 | `GetAssetGroupMapping(userId, groupId)` + `DeleteAssetGroupMapping(userId, groupId)` |
| `service/aigc_asset.go` | 修改逻辑 | `CreateAIGCAssetMapping` 判重改为查 user_id+group_id；`DeleteAIGCAssetGroup` 不再忽略映射删除错误 |
| `service/asset.go` | 修改逻辑 | `CreateUserAssetMapping` 判重改为查 user_id+group_id；`DeleteAssetGroup` 新增映射清理逻辑 |
