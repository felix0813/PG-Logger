# PG-Logger Go Client 示例

这个目录提供了一个可直接复制到业务项目中的 Go 客户端示例。

## 文件说明

- `logger_client.go`：最小可用客户端，包含：
  - 注册应用（`POST /logger/apps`）
  - 发送日志 JSON（`POST /logger/logs`）
- `example_main.go`：调用示例

## 使用方式

1. 将 `logger_client.go` 复制到你的项目（可选同时复制 `example_main.go`）。
2. 配置环境变量：

```bash
export LOGGER_SERVER_URL="http://localhost:8080"
```

3. 在代码中创建客户端并调用：

```go
cli, err := client.NewFromEnv()
if err != nil {
    // handle error
}

ctx := context.Background()
_ = cli.RegisterApp(ctx, client.RegisterAppRequest{
    AppCode:       "demo-api",
    AppName:       "Demo API",
    Env:           "prod",
    Enabled:       true,
    RetentionDays: 30,
    Description:   "demo service",
})

_ = cli.SendLog(ctx, client.SendLogRequest{
    AppCode: "demo-api",
    Env:     "prod",
    Level:   "INFO",
    Message: "hello pg-logger",
})
```

## 请求 JSON 结构

### 注册应用 JSON

```json
{
  "app_code": "demo-api",
  "app_name": "Demo API",
  "env": "prod",
  "enabled": true,
  "retention_days": 30,
  "description": "demo service"
}
```

### 发送日志 JSON

```json
{
  "app_code": "demo-api",
  "env": "prod",
  "level": "ERROR",
  "message": "database timeout",
  "path": "/api/v1/orders",
  "method": "GET",
  "status_code": 500,
  "duration_ms": 1200,
  "trace_id": "trace-123",
  "request_id": "req-456",
  "exception": "context deadline exceeded",
  "extra": {
    "node": "node-a"
  }
}
```
