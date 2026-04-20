package storage

import (
	"context"
	"fmt"
	"time"
)

type App struct {
	ID            int64     `json:"id"`
	AppCode       string    `json:"app_code"`
	AppName       string    `json:"app_name"`
	Env           string    `json:"env"`
	Enabled       bool      `json:"enabled"`
	RetentionDays int       `json:"retention_days"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateAppInput struct {
	AppCode       string `json:"app_code"`
	AppName       string `json:"app_name"`
	Env           string `json:"env"`
	Enabled       *bool  `json:"enabled"`
	RetentionDays *int   `json:"retention_days"`
	Description   string `json:"description"`
}

type UpdateAppInput struct {
	AppName       string `json:"app_name"`
	Env           string `json:"env"`
	Enabled       *bool  `json:"enabled"`
	RetentionDays *int   `json:"retention_days"`
	Description   string `json:"description"`
}

func (p *Postgres) CreateApp(ctx context.Context, input CreateAppInput) (*App, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	retentionDays := 30
	if input.RetentionDays != nil {
		retentionDays = *input.RetentionDays
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO log_app (app_code, app_name, env, enabled, retention_days, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, app_code, app_name, env, enabled, retention_days, COALESCE(description, ''), created_at, updated_at`,
		input.AppCode, input.AppName, input.Env, enabled, retentionDays, input.Description,
	)

	var app App
	if err := row.Scan(&app.ID, &app.AppCode, &app.AppName, &app.Env, &app.Enabled, &app.RetentionDays, &app.Description, &app.CreatedAt, &app.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert app: %w", err)
	}

	return &app, nil
}

func (p *Postgres) ListApps(ctx context.Context) ([]App, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, app_code, app_name, env, enabled, retention_days, COALESCE(description, ''), created_at, updated_at
		FROM log_app
		ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("query apps: %w", err)
	}
	defer rows.Close()

	apps := make([]App, 0)
	for rows.Next() {
		var app App
		if scanErr := rows.Scan(&app.ID, &app.AppCode, &app.AppName, &app.Env, &app.Enabled, &app.RetentionDays, &app.Description, &app.CreatedAt, &app.UpdatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan app: %w", scanErr)
		}
		apps = append(apps, app)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate apps: %w", rows.Err())
	}

	return apps, nil
}

func (p *Postgres) UpdateAppByCode(ctx context.Context, appCode string, input UpdateAppInput) (*App, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	retentionDays := 30
	if input.RetentionDays != nil {
		retentionDays = *input.RetentionDays
	}

	row := p.pool.QueryRow(ctx, `
		UPDATE log_app
		SET app_name = $1,
			env = $2,
			enabled = $3,
			retention_days = $4,
			description = $5,
			updated_at = NOW()
		WHERE app_code = $6
		RETURNING id, app_code, app_name, env, enabled, retention_days, COALESCE(description, ''), created_at, updated_at`,
		input.AppName, input.Env, enabled, retentionDays, input.Description, appCode,
	)

	var app App
	if err := row.Scan(&app.ID, &app.AppCode, &app.AppName, &app.Env, &app.Enabled, &app.RetentionDays, &app.Description, &app.CreatedAt, &app.UpdatedAt); err != nil {
		return nil, err
	}

	return &app, nil
}

func (p *Postgres) DeleteAppByCode(ctx context.Context, appCode string) error {
	result, err := p.pool.Exec(ctx, `DELETE FROM log_app WHERE app_code = $1`, appCode)
	if err != nil {
		return fmt.Errorf("delete app: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("app not found")
	}
	return nil
}
