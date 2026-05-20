# 修改日志 - 接入火山引擎私域虚拟人像素材资产接口（s2）

## 日期
2026-05-20

## 需求背景
在现有 `/s1/asset/`（真人素材）基础上，新增 `/s2/aigc-asset/`（虚拟人像素材）接口，
支持用户直接创建 Asset Group 并上传虚拟人像素材，无需真人认证前置流程。

## 变更内容

### 后端
1. **constant/channel.go** — 新增 Channel 类型常量 `ChannelTypeVolcEngineAIGC = 58`（type=58）
2. **dto/aigc_asset.go** — 新建 AIGC 素材相关的请求/响应 DTO
3. **model/aigc_asset_group_mapping.go** — 新建 AIGC Asset Group 用户映射 Model（独立于真人素材的 asset_group_mappings 表）
4. **model/main.go** — 在 migrateDB() 和 migrateDBFast() 的 AutoMigrate 列表中注册 `AIGCAssetGroupMapping`
5. **service/aigc_asset.go** — 新建 AIGC 素材业务逻辑层（创建素材组、CRUD 素材等），复用 CallArkAPI，按用户数据隔离
6. **controller/aigc_asset.go** — 新建 AIGC 素材 HTTP 处理器
7. **router/aigc-asset-router.go** — 新建 AIGC 素材路由，注册到 `/s2/aigc-asset/` 路径下
8. **router/main.go** — 注册 `SetAIGCAssetRouter`
9. **middleware/distributor.go** — 修复 Distribute 中间件，将 `/s2/aigc-asset/` 加入跳过 Channel 选择的白名单（解决 "Model name not specified" 报错）
10. **docs/api/volcengine-ark-asset-api.md** — 更新 API 文档，合并真人素材和虚拟人像两套流程为一份文档

### 前端
1. **web/default/src/features/channels/constants.ts** — CHANNEL_TYPES 新增 58: 'VolcEngineAIGC'，CHANNEL_TYPE_DISPLAY_ORDER 增加 58
2. **web/default/src/features/channels/lib/channel-utils.ts** — 模型图标映射新增 58: 'Volcengine'
3. **web/default/src/i18n/locales/en.json** — 新增 "VolcEngineAIGC" i18n key
4. **web/default/src/i18n/locales/zh.json** — 新增 "VolcEngineAIGC" i18n key（"字节火山-虚拟人像"）

## 数据库变更
- 新建表 `aigc_asset_group_mappings` — 记录用户与虚拟人像 Asset Group 的绑定关系
- 通过 GORM AutoMigrate 自动创建，无需手动操作
- 不影响现有 `asset_group_mappings` 表和数据

## 接口清单（/s2/aigc-asset/）

| 路由 | 说明 |
|------|------|
| POST /s2/aigc-asset/asset-groups/create | 创建素材资产组合 |
| POST /s2/aigc-asset/asset-groups/list | 查询素材组列表 |
| POST /s2/aigc-asset/asset-groups/get | 获取素材组详情 |
| POST /s2/aigc-asset/asset-groups/update | 更新素材组信息 |
| POST /s2/aigc-asset/asset-groups/delete | 删除素材组 |
| POST /s2/aigc-asset/assets/create | 上传素材（异步） |
| POST /s2/aigc-asset/assets/list | 查询素材列表 |
| POST /s2/aigc-asset/assets/get | 获取素材详情（轮询状态） |
| POST /s2/aigc-asset/assets/update | 更新素材名称 |
| POST /s2/aigc-asset/assets/delete | 删除素材 |
