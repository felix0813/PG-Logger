-- PG-Logger Grafana 查询模板
-- 适用数据表: app_log
-- 说明:
-- 1) 所有查询都使用 Grafana PostgreSQL 数据源宏
-- 2) 推荐在 Dashboard 里创建变量: app_code、level、env

/*
变量建议（Dashboard Variables）
--------------------------------
变量 app_code（Query）:
  SELECT DISTINCT app_code FROM app_log ORDER BY 1;

变量 level（Query）:
  SELECT DISTINCT level FROM app_log ORDER BY 1;

变量 env（Query）:
  SELECT DISTINCT env FROM app_log ORDER BY 1;

变量 multi-value + include All:
- 勾选 Multi-value
- 勾选 Include All option
- 推荐 All value 使用 .*（配合 ~ 正则匹配）
*/

-- 1) 日志明细（类 Kibana Discover）
-- Panel 类型: Table
SELECT log_time AS "time",
       app_code,
       env,
       level,
       host,
       trace_id,
       request_id,
       method,
       path,
       status_code,
       duration_ms,
       message,
       exception,
       extra
FROM app_log
WHERE $__timeFilter(log_time)
  AND app_code ~ '^(${app_code:regex})$'
  AND level ~ '^(${level:regex})$'
  AND env ~ '^(${env:regex})$'
ORDER BY log_time DESC
LIMIT ${limit:raw};

-- 2) 日志量趋势（按时间桶）
-- Panel 类型: Time series
SELECT $__timeGroupAlias(log_time, $__interval),
  count(*)::bigint AS value
FROM app_log
WHERE $__timeFilter(log_time)
  AND app_code ~ '^(${app_code:regex})$'
  AND level ~ '^(${level:regex})$'
  AND env ~ '^(${env:regex})$'
GROUP BY 1
ORDER BY 1;

-- 3) 按级别统计
-- Panel 类型: Bar chart / Pie chart / Table
SELECT level,
       count(*)::bigint AS total
FROM app_log
WHERE $__timeFilter(log_time)
  AND app_code ~ '^(${app_code:regex})$'
  AND env ~ '^(${env:regex})$'
GROUP BY level
ORDER BY total DESC;

-- 4) 按应用统计
-- Panel 类型: Bar chart / Table
SELECT app_code,
       count(*)::bigint AS total
FROM app_log
WHERE $__timeFilter(log_time)
  AND level ~ '^(${level:regex})$'
  AND env ~ '^(${env:regex})$'
GROUP BY app_code
ORDER BY total DESC;

-- 5) 错误日志 TopN（ERROR/FATAL）
-- Panel 类型: Table
SELECT message,
       count(*)::bigint AS total
FROM app_log
WHERE $__timeFilter(log_time)
  AND app_code ~ '^(${app_code:regex})$'
  AND env ~ '^(${env:regex})$'
  AND level IN ('ERROR'
    , 'FATAL')
GROUP BY message
ORDER BY total DESC
LIMIT 20;

-- 6) 最近错误日志（排障面板）
-- Panel 类型: Logs / Table
SELECT log_time AS "time",
       app_code,
       level,
       trace_id,
       request_id,
       message,
       exception
FROM app_log
WHERE $__timeFilter(log_time)
  AND app_code ~ '^(${app_code:regex})$'
  AND env ~ '^(${env:regex})$'
  AND level IN ('ERROR'
    , 'FATAL')
ORDER BY log_time DESC
LIMIT 200;

-- 7) 可选：用于日志详情跳转（通过 trace_id）
-- Panel 类型: Table
SELECT log_time AS "time",
       app_code,
       level,
       module,
       logger,
       message,
       exception,
       extra
FROM app_log
WHERE $__timeFilter(log_time)
  AND trace_id = '${trace_id}'
ORDER BY log_time ASC;
