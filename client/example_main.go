package client

import (
	"context"
	"log"
	"time"
)

// ExampleUsage 演示如何注册应用并发送日志。
// 复制本文件和 logger_client.go 到你的项目后可直接使用。
func ExampleUsage() {
	cli, err := NewFromEnv()
	if err != nil {
		log.Fatalf("create client failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = cli.RegisterApp(ctx, RegisterAppRequest{
		AppCode:       "demo-api",
		AppName:       "Demo API",
		Env:           "prod",
		Enabled:       true,
		RetentionDays: 30,
		Description:   "demo service",
	})
	if err != nil {
		log.Fatalf("register app failed: %v", err)
	}

	err = cli.SendLog(ctx, SendLogRequest{
		AppCode:    "demo-api",
		Env:        "prod",
		Level:      "INFO",
		Message:    "user login success",
		Path:       "/api/v1/login",
		Method:     "POST",
		StatusCode: 200,
		DurationMS: 48,
		TraceID:    "trace-demo-001",
		RequestID:  "req-demo-001",
		Extra: map[string]any{
			"user_id": 1001,
			"region":  "us-east-1",
		},
	})
	if err != nil {
		log.Fatalf("send log failed: %v", err)
	}
}
