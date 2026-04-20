package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type AppLog struct {
	ID         int64          `json:"id"`
	LogTime    time.Time      `json:"log_time"`
	AppCode    string         `json:"app_code"`
	Env        string         `json:"env"`
	Level      string         `json:"level"`
	Message    string         `json:"message"`
	Host       string         `json:"host,omitempty"`
	Path       string         `json:"path,omitempty"`
	Method     string         `json:"method,omitempty"`
	StatusCode int            `json:"status_code,omitempty"`
	DurationMS int            `json:"duration_ms,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	Exception  string         `json:"exception,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type CreateLogInput struct {
	LogTime    *time.Time     `json:"log_time"`
	AppCode    string         `json:"app_code"`
	Env        string         `json:"env"`
	Level      string         `json:"level"`
	Message    string         `json:"message"`
	Host       string         `json:"host"`
	Path       string         `json:"path"`
	Method     string         `json:"method"`
	StatusCode int            `json:"status_code"`
	DurationMS int            `json:"duration_ms"`
	TraceID    string         `json:"trace_id"`
	RequestID  string         `json:"request_id"`
	Exception  string         `json:"exception"`
	Extra      map[string]any `json:"extra"`
}

func (p *Postgres) CreateLog(ctx context.Context, input CreateLogInput) (*AppLog, error) {
	logTime := time.Now().UTC()
	if input.LogTime != nil {
		logTime = input.LogTime.UTC()
	}
	if input.Env == "" {
		input.Env = "prod"
	}
	if input.Extra == nil {
		input.Extra = map[string]any{}
	}

	extraJSON, err := json.Marshal(input.Extra)
	if err != nil {
		return nil, fmt.Errorf("marshal extra: %w", err)
	}

	if _, err = p.pool.Exec(ctx, `SELECT create_app_log_partition($1::date)`, logTime); err != nil {
		return nil, fmt.Errorf("create partition: %w", err)
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO app_log (log_time, app_code, env, level, message, host, path, method, status_code, duration_ms, trace_id, request_id, exception, extra)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, 0), NULLIF($10, 0), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), $14)
		RETURNING id, log_time, app_code, env, level, message, COALESCE(host, ''), COALESCE(path, ''), COALESCE(method, ''), COALESCE(status_code, 0), COALESCE(duration_ms, 0), COALESCE(trace_id, ''), COALESCE(request_id, ''), COALESCE(exception, ''), extra, created_at`,
		logTime, input.AppCode, input.Env, input.Level, input.Message, input.Host, input.Path, input.Method, input.StatusCode, input.DurationMS, input.TraceID, input.RequestID, input.Exception, extraJSON,
	)

	var logRecord AppLog
	var extraRaw []byte
	if err = row.Scan(&logRecord.ID, &logRecord.LogTime, &logRecord.AppCode, &logRecord.Env, &logRecord.Level, &logRecord.Message, &logRecord.Host, &logRecord.Path, &logRecord.Method, &logRecord.StatusCode, &logRecord.DurationMS, &logRecord.TraceID, &logRecord.RequestID, &logRecord.Exception, &extraRaw, &logRecord.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert app_log: %w", err)
	}

	if unmarshalErr := json.Unmarshal(extraRaw, &logRecord.Extra); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal extra: %w", unmarshalErr)
	}

	return &logRecord, nil
}

func (p *Postgres) ListLogs(ctx context.Context, appCode, level, env string, limit int32) ([]AppLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := p.pool.Query(ctx, `
		SELECT id, log_time, app_code, env, level, message, COALESCE(host, ''), COALESCE(path, ''), COALESCE(method, ''), COALESCE(status_code, 0), COALESCE(duration_ms, 0), COALESCE(trace_id, ''), COALESCE(request_id, ''), COALESCE(exception, ''), extra, created_at
		FROM app_log
		WHERE ($1 = '' OR app_code = $1)
		  AND ($2 = '' OR level = $2)
		  AND ($3 = '' OR env = $3)
		ORDER BY log_time DESC
		LIMIT $4`, appCode, level, env, limit)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	logs := make([]AppLog, 0)
	for rows.Next() {
		var item AppLog
		var extraRaw []byte
		if scanErr := rows.Scan(&item.ID, &item.LogTime, &item.AppCode, &item.Env, &item.Level, &item.Message, &item.Host, &item.Path, &item.Method, &item.StatusCode, &item.DurationMS, &item.TraceID, &item.RequestID, &item.Exception, &extraRaw, &item.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan log: %w", scanErr)
		}
		if unmarshalErr := json.Unmarshal(extraRaw, &item.Extra); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal log extra: %w", unmarshalErr)
		}
		logs = append(logs, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate logs: %w", rows.Err())
	}

	return logs, nil
}

func (p *Postgres) DeleteLog(ctx context.Context, id int64, logTime time.Time) error {
	result, err := p.pool.Exec(ctx, `DELETE FROM app_log WHERE id = $1 AND log_time = $2`, id, logTime)
	if err != nil {
		return fmt.Errorf("delete log: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("log not found")
	}
	return nil
}
