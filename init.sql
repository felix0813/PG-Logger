CREATE TABLE IF NOT EXISTS log_app (
                                       id              BIGSERIAL PRIMARY KEY,
                                       app_code        VARCHAR(64) NOT NULL UNIQUE,
                                       app_name        VARCHAR(128) NOT NULL,
                                       env             VARCHAR(32) NOT NULL DEFAULT 'prod',
                                       enabled         BOOLEAN NOT NULL DEFAULT TRUE,
                                       retention_days  INTEGER NOT NULL DEFAULT 30,
                                       description     TEXT,
                                       created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                       updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE log_app IS '应用日志配置表，用于记录日志来源应用、所属环境、是否启用及日志保留天数等配置';

COMMENT ON COLUMN log_app.id IS '主键ID';
COMMENT ON COLUMN log_app.app_code IS '应用编码，唯一标识一个应用，例如 monitor-api';
COMMENT ON COLUMN log_app.app_name IS '应用名称，用于展示';
COMMENT ON COLUMN log_app.env IS '运行环境，例如 prod、test、dev';
COMMENT ON COLUMN log_app.enabled IS '是否启用该应用的日志采集或展示';
COMMENT ON COLUMN log_app.retention_days IS '该应用日志默认保留天数';
COMMENT ON COLUMN log_app.description IS '应用说明或备注';
COMMENT ON COLUMN log_app.created_at IS '记录创建时间';
COMMENT ON COLUMN log_app.updated_at IS '记录更新时间';

CREATE TABLE IF NOT EXISTS app_log (
                                       id              BIGSERIAL ,
                                       log_time        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                       app_code        VARCHAR(64) NOT NULL,
                                       env             VARCHAR(32) NOT NULL DEFAULT 'prod',
                                       level           VARCHAR(16) NOT NULL,
                                       host            VARCHAR(128),
                                       instance_id     VARCHAR(128),
                                       trace_id        VARCHAR(128),
                                       span_id         VARCHAR(128),
                                       module          VARCHAR(128),
                                       logger          VARCHAR(256),
                                       user_id         VARCHAR(128),
                                       request_id      VARCHAR(128),
                                       path            TEXT,
                                       method          VARCHAR(16),
                                       status_code     INTEGER,
                                       duration_ms     INTEGER,
                                       message         TEXT NOT NULL,
                                       exception       TEXT,
                                       extra           JSONB NOT NULL DEFAULT '{}'::jsonb,
                                       created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                                       PRIMARY KEY (id, log_time)
) PARTITION BY RANGE (log_time);

COMMENT ON TABLE app_log IS '应用日志主表，用于存储应用运行过程中产生的日志，支持结构化字段和扩展 JSON 字段';

COMMENT ON COLUMN app_log.id IS '主键ID';
COMMENT ON COLUMN app_log.log_time IS '日志产生时间';
COMMENT ON COLUMN app_log.app_code IS '应用编码，对应 log_app.app_code';
COMMENT ON COLUMN app_log.env IS '运行环境，例如 prod、test、dev';
COMMENT ON COLUMN app_log.level IS '日志级别，例如 DEBUG、INFO、WARN、ERROR、FATAL';
COMMENT ON COLUMN app_log.host IS '主机名或服务器标识';
COMMENT ON COLUMN app_log.instance_id IS '应用实例ID，例如容器ID或进程实例标识';
COMMENT ON COLUMN app_log.trace_id IS '链路追踪ID，用于串联一次请求的完整日志';
COMMENT ON COLUMN app_log.span_id IS '链路追踪中的当前调用跨度ID';
COMMENT ON COLUMN app_log.module IS '业务模块名称';
COMMENT ON COLUMN app_log.logger IS '日志记录器名称，例如类名或组件名';
COMMENT ON COLUMN app_log.user_id IS '用户ID，便于排查用户相关问题';
COMMENT ON COLUMN app_log.request_id IS '请求ID，用于标识单次请求';
COMMENT ON COLUMN app_log.path IS '请求路径，例如 /api/user/list';
COMMENT ON COLUMN app_log.method IS 'HTTP请求方法，例如 GET、POST';
COMMENT ON COLUMN app_log.status_code IS 'HTTP状态码或业务状态码';
COMMENT ON COLUMN app_log.duration_ms IS '请求耗时，单位毫秒';
COMMENT ON COLUMN app_log.message IS '日志正文内容';
COMMENT ON COLUMN app_log.exception IS '异常堆栈或错误详细信息';
COMMENT ON COLUMN app_log.extra IS '扩展字段，使用 JSONB 存储额外业务信息';
COMMENT ON COLUMN app_log.created_at IS '记录写入数据库的时间';

CREATE OR REPLACE FUNCTION create_app_log_partition(p_target_date DATE)
    RETURNS VOID
    LANGUAGE plpgsql
AS $$
DECLARE
    v_month_start   DATE;
    v_next_month    DATE;
    v_partition_name TEXT;
BEGIN
    -- 取传入日期所在月份的第一天
    v_month_start := date_trunc('month', p_target_date)::date;
    v_next_month  := (v_month_start + INTERVAL '1 month')::date;

    -- 生成分区表名，例如 app_log_2026_05
    v_partition_name := 'app_log_' || to_char(v_month_start, 'YYYY_MM');

    -- 创建分区
    EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF app_log
             FOR VALUES FROM (%L) TO (%L)',
            v_partition_name,
            v_month_start,
            v_next_month
            );
END;
$$;