# PG-Logger

一个基于 Go + PostgreSQL 的日志服务，支持应用管理和日志管理。

## 启动前准备

1. 创建数据库表：

```bash
psql "$DATABASE_URL" -f init.sql
```

2. 配置环境变量（必须配置，否则应用会直接退出）：

- `DATABASE_URL`：PostgreSQL 连接串
- `SERVER_ADDR`：HTTP 服务监听地址，例如 `:8080`

3. 启动应用：

```bash
go run .
```

## API 说明

### 健康检查

- `GET /logger/health`

### 应用管理（log_app）

- `POST /logger/apps`：新增应用
- `GET /logger/apps`：查询应用列表
- `PUT /logger/apps/{app_code}`：更新应用
- `DELETE /logger/apps/{app_code}`：删除应用

#### 新增应用请求示例

```json
{
  "app_code": "monitor-api",
  "app_name": "Monitor API",
  "env": "prod",
  "enabled": true,
  "retention_days": 30,
  "description": "核心监控服务"
}
```

### 日志管理（app_log）

- `POST /logger/logs`：新增日志
- `GET /logger/logs`：查询日志
  - 可选 query 参数：`app_code`、`level`、`env`、`limit`
- `DELETE /logger/logs?id={id}&log_time={RFC3339}`：删除日志

#### 新增日志请求示例

```json
{
  "app_code": "monitor-api",
  "env": "prod",
  "level": "ERROR",
  "message": "database timeout",
  "path": "/api/v1/jobs",
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

## ENV 配置示例

可以参考 `.env.example`：

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
SERVER_ADDR=:8080
```
