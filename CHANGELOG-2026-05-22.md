# 2026-05-22 修改日志

## 一、代码修改

### 1.1 `/s1/video/generations` 查询接口 — 添加 `format` 参数区分响应格式

**背景**：`GET /s1/video/generations/:task_id` 当前返回双层 `data` 嵌套（外层 new-api 任务元数据，内层上游原始响应），难以区分。

**修改**：添加 `format` 查询参数：
- **无参数**：直接透传上游原始响应体
- **`?format=ni`**：返回 `{ code, message, data: { ni: {任务元数据}, upstream: {上游原始响应} } }`，用 `ni` 和 `upstream` 字段明确区分

#### 修改 1：`relay/relay_task.go` — `videoFetchByIDRespBodyBuilder`

```go
// Before
// 通用 TaskDto 格式
respBody, err = common.Marshal(dto.TaskResponse[any]{
    Code: "success",
    Data: TaskModel2Dto(originTask),
})

// After
format := c.Query("format")

if format == "ni" {
    respBody, err = common.Marshal(dto.TaskResponse[any]{
        Code:    "success",
        Message: "",
        Data: gin.H{
            "ni":       TaskModel2Dto(originTask),
            "upstream": originTask.Data,
        },
    })
    // ...
    return
}

// 默认：只返回上游原始响应体
respBody = originTask.Data
return
```

---

### 1.2 `/s1/video/generations` 新增 DELETE 接口 — 取消/删除视频任务

**背景**：根据火山方舟 Seedance 2.0 官方文档，`DELETE /api/v3/contents/generations/tasks/{id}` 支持取消排队中任务或删除已完成/失败任务记录。`/s1` 路径只有 POST（提交）和 GET（查询），缺少取消/删除接口。

#### 修改 2：`relay/constant/relay_mode.go` — 新增 relay mode 常量

```go
RelayModeSeedanceCancelByID
```

#### 修改 3：`middleware/distributor.go` — DELETE 方法分支

在 `/s1/video/generations` 处理块中添加：
```go
} else if c.Request.Method == http.MethodDelete {
    relayMode = relayconstant.RelayModeSeedanceCancelByID
    shouldSelectChannel = false
}
```

#### 修改 4：`model/task.go` — 新增取消状态常量

```go
TaskStatusCancelled = "CANCELLED"
```

#### 修改 5：`relay/channel/task/doubao/adaptor.go` — 新增 `CancelTask` 方法

```go
func (a *TaskAdaptor) CancelTask(baseURL, key, taskID, proxy string) (*http.Response, error) {
    uri := fmt.Sprintf("%s/api/v3/contents/generations/tasks/%s", baseURL, taskID)
    req, err := http.NewRequest(http.MethodDelete, uri, nil)
    // 设置 Authorization: Bearer {key}
    // ...
}
```

#### 修改 6：`relay/relay_task.go` — 新增 `RelayTaskCancel` + `videoCancelRespBodyBuilder`

- 分发器 `cancelRespBuilders` 映射 relay mode 到 builder
- `RelayTaskCancel` 函数：类似 `RelayTaskFetch` 的模式
- `videoCancelRespBodyBuilder`：
  1. 查询本地任务，校验用户权限
  2. 检查状态：`IN_PROGRESS` 和 `CANCELLED` 不支持操作，返回 400
  3. 通过 Doubao adaptor 调用上游 DELETE 接口
  4. `QUEUED` 状态被取消 → 退还 quota，更新状态为 `CANCELLED`
  5. 已完成/失败任务 → 仅删除上游记录

#### 修改 7：`controller/relay.go` — 新增 `RelayTaskCancel` 控制器

```go
func RelayTaskCancel(c *gin.Context) {
    relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
    if err != nil { ... }
    if taskErr := relay.RelayTaskCancel(c, relayInfo.RelayMode); taskErr != nil {
        respondTaskError(c, taskErr)
    }
}
```

#### 修改 8：`router/video-router.go` — 新增 DELETE 路由

```go
videoS1Router.DELETE("/video/generations/:task_id", controller.RelayTaskCancel)
```

---

## 二、文档修改

### 2.1 `docs/api/s1-video-generations.md` — 更新接口文档

- 概述部分新增 `DELETE /s1/video/generations/:task_id` 路径
- 查询任务接口新增 `format` 参数说明，含默认响应和 `?format=ni` 响应的完整示例
- 新增 `DELETE /s1/video/generations/:task_id` 完整文档：状态说明、操作含义、响应示例、curl 示例
- 注意事项中补充查询响应格式说明

---

## 三、涉及文件清单

| 文件 | 修改类型 | 说明 |
|------|---------|------|
| `relay/relay_task.go` | 修改+新增 | `videoFetchByIDRespBodyBuilder` 添加 `format` 参数分支；新增 `RelayTaskCancel`、`videoCancelRespBodyBuilder` |
| `relay/constant/relay_mode.go` | 新增 | `RelayModeSeedanceCancelByID` 常量 |
| `middleware/distributor.go` | 新增 | `/s1/video/generations` DELETE 方法分支 |
| `model/task.go` | 新增 | `TaskStatusCancelled` 常量 |
| `relay/channel/task/doubao/adaptor.go` | 新增 | `CancelTask` 方法 |
| `controller/relay.go` | 新增 | `RelayTaskCancel` 控制器 |
| `router/video-router.go` | 新增 | DELETE 路由 |
| `docs/api/s1-video-generations.md` | 新增 | `format` 参数和 DELETE 接口文档 |
