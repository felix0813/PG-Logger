# Grafana + PostgreSQL 日志检索说明

本目录提供了将 `PG-Logger` 的 `app_log` 作为 Grafana 数据源进行日志分析的推荐做法，目标效果接近 **ES + Kibana** 的常用能力：

- 按时间范围筛选
- 按应用（`app_code`）筛选
- 按日志级别（`level`）筛选
- 查看日志明细与聚合趋势

---

## 1. 前置条件

1. 已按项目根目录 `init.sql` 初始化 `app_log` 与分区。
2. Grafana 已可访问 PostgreSQL。
3. PostgreSQL 用户对 `app_log` 有 `SELECT` 权限。

---

## 2. 配置 PostgreSQL 数据源

在 Grafana 中：

1. `Connections` -> `Add new connection` -> 选择 `PostgreSQL`
2. 填写连接信息（Host、Database、User、Password）
3. 推荐设置：
    - `TLS/SSL Mode` 按你的环境选择
    - `PostgreSQL version` 选择实际版本
    - `Min time interval` 可先设为 `1s` 或 `10s`
4. 保存并点击 `Save & test`

---

## 3. 创建 Dashboard 变量（核心）

为了实现类似 Kibana 的筛选体验，先创建变量：

### 3.1 `app_code`

- Type: `Query`
- Data source: PostgreSQL
- Query:

```sql
SELECT DISTINCT app_code FROM app_log ORDER BY 1;
```

- 打开 `Multi-value`
- 打开 `Include All option`

### 3.2 `level`

```sql
SELECT DISTINCT level FROM app_log ORDER BY 1;
```

同样开启 `Multi-value` + `Include All option`。

### 3.3 `env`

```sql
SELECT DISTINCT env FROM app_log ORDER BY 1;
```

同样开启 `Multi-value` + `Include All option`。

### 3.4 `limit`（可选）

- Type: `Custom`
- Values: `100,200,500,1000`
- 默认可设为 `200`

---

## 4. SQL 查询模板

完整 SQL 位于：`grafana/queries.sql`。

你可以直接复制其中查询到不同 Panel：

1. 日志明细（Table）
2. 日志量趋势（Time series）
3. 按级别统计（Bar/Pie/Table）
4. 按应用统计（Bar/Table）
5. 错误日志 TopN
6. 最近错误日志

这些 SQL 均使用了 Grafana 时间宏与变量，例如：

- `$__timeFilter(log_time)`
- `$__timeGroupAlias(log_time, $__interval)`
- `${app_code:regex}` / `${level:regex}` / `${env:regex}`

从而支持 **时间 + 应用 + 级别** 联合筛选。

---

## 5. 推荐 Dashboard 布局（示例）

第一行（总览）：

- 日志量趋势（Time series）
- 按级别统计（Bar/Pie）
- 按应用统计（Bar）

第二行（排障）：

- 最近错误日志（Table）
- 错误日志 TopN（Table）

第三行（明细）：

- 日志明细（Table，支持排序/搜索）

---

## 6. 性能优化建议

如果日志量较大（千万级以上），建议增加索引（按实际查询情况调整）：

```sql
CREATE INDEX IF NOT EXISTS idx_app_log_time ON app_log (log_time DESC);
CREATE INDEX IF NOT EXISTS idx_app_log_app_time ON app_log (app_code, log_time DESC);
CREATE INDEX IF NOT EXISTS idx_app_log_level_time ON app_log (level, log_time DESC);
CREATE INDEX IF NOT EXISTS idx_app_log_env_time ON app_log (env, log_time DESC);
```

此外：

- 保持按月分区持续创建
- 结合 `retention_days` 做过期数据清理
- 控制明细查询 `LIMIT`，避免一次拉取过大数据量

---

## 7. 常见问题

### Q1: 多选变量为什么要用正则匹配？

Grafana 多选变量会展开为多值列表。使用：

```sql
app_code ~ '^(${app_code:regex})$'
```

可以同时兼容单选、多选和 All 选项。

### Q2: 为什么不用 `IN (...)`？

可以用 `IN`，但在 Grafana 变量多选和转义场景下，`:regex` 往往更稳，模板更统一。

### Q3: 能否像 Kibana 一样“点日志看上下文”？

可以基于 `trace_id`、`request_id` 再做 drill-down（例如详情页变量跳转）。`queries.sql` 已提供按 `trace_id` 的示例。

