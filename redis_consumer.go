package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"pg-logger/storage"
)

type redisStreamConsumer struct {
	addr      string
	streamKey string
	postgres  *storage.Postgres
	blockMS   int64
}

func newRedisStreamConsumer(postgres *storage.Postgres) (*redisStreamConsumer, bool) {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	streamKey := strings.TrimSpace(os.Getenv("REDIS_STREAM_KEY"))
	if addr == "" || streamKey == "" {
		log.Printf("redis stream consumer disabled, REDIS_ADDR or REDIS_STREAM_KEY not set")
		return nil, false
	}

	blockMS := int64(2000)
	if raw := strings.TrimSpace(os.Getenv("REDIS_STREAM_BLOCK_MS")); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v <= 0 {
			log.Printf("invalid REDIS_STREAM_BLOCK_MS=%q, fallback to 2000", raw)
		} else {
			blockMS = v
		}
	}

	return &redisStreamConsumer{addr: addr, streamKey: streamKey, postgres: postgres, blockMS: blockMS}, true
}

func (c *redisStreamConsumer) run(ctx context.Context) {
	log.Printf("redis stream consumer started, stream=%s, addr=%s", c.streamKey, c.addr)
	defer log.Printf("redis stream consumer stopped")

	lastID := "$"
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		messages, err := c.readOnce(ctx, lastID)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("redis stream read failed: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, msg := range messages {
			lastID = msg.ID
			input, parseErr := parseRedisLogMessage(msg.Values)
			if parseErr != nil {
				log.Printf("parse redis log message failed, id=%s, err=%v", msg.ID, parseErr)
				continue
			}

			if _, saveErr := c.postgres.CreateLog(ctx, input); saveErr != nil {
				log.Printf("store redis log failed, id=%s, app_code=%s, err=%v", msg.ID, input.AppCode, saveErr)
				continue
			}
			log.Printf("stored redis log, id=%s, app_code=%s, level=%s", msg.ID, input.AppCode, input.Level)
		}
	}
}

type streamMessage struct {
	ID     string
	Values map[string]any
}

func (c *redisStreamConsumer) readOnce(ctx context.Context, lastID string) ([]streamMessage, error) {
	conn, err := net.DialTimeout("tcp", c.addr, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial redis: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(time.Duration(c.blockMS+3000) * time.Millisecond)
	_ = conn.SetDeadline(deadline)

	cmd := redisArray("XREAD", "BLOCK", strconv.FormatInt(c.blockMS, 10), "COUNT", "100", "STREAMS", c.streamKey, lastID)
	if _, err = conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("write xread: %w", err)
	}

	resp, err := readRESP(bufio.NewReader(conn))
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			return nil, nil
		}
		return nil, fmt.Errorf("read xread response: %w", err)
	}

	return parseXReadResponse(resp)
}

func redisArray(parts ...string) string {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(parts)))
	b.WriteString("\r\n")
	for _, p := range parts {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(p)))
		b.WriteString("\r\n")
		b.WriteString(p)
		b.WriteString("\r\n")
	}
	return b.String()
}

func parseRedisLogMessage(values map[string]any) (storage.CreateLogInput, error) {
	if raw, ok := values["payload"]; ok {
		if s, ok := raw.(string); ok {
			var input storage.CreateLogInput
			if err := json.Unmarshal([]byte(s), &input); err != nil {
				return storage.CreateLogInput{}, err
			}
			if input.AppCode == "" || input.Level == "" || input.Message == "" {
				return storage.CreateLogInput{}, errors.New("app_code, level and message are required")
			}
			return input, nil
		}
	}

	b, err := json.Marshal(values)
	if err != nil {
		return storage.CreateLogInput{}, err
	}
	var input storage.CreateLogInput
	if err = json.Unmarshal(b, &input); err != nil {
		return storage.CreateLogInput{}, err
	}
	if input.AppCode == "" || input.Level == "" || input.Message == "" {
		return storage.CreateLogInput{}, errors.New("app_code, level and message are required")
	}
	return input, nil
}

func readRESP(r *bufio.Reader) (any, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+', '-':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		if prefix == '-' {
			return nil, errors.New(line)
		}
		return line, nil
	case ':':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(line, 10, 64)
	case '$':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err = io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		items := make([]any, n)
		for i := 0; i < n; i++ {
			item, err := readRESP(r)
			if err != nil {
				return nil, err
			}
			items[i] = item
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unknown RESP prefix: %q", prefix)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func parseXReadResponse(resp any) ([]streamMessage, error) {
	if resp == nil {
		return nil, nil
	}
	root, ok := resp.([]any)
	if !ok {
		return nil, errors.New("xread response format invalid")
	}
	messages := make([]streamMessage, 0)
	for _, streamRaw := range root {
		streamArr, ok := streamRaw.([]any)
		if !ok || len(streamArr) != 2 {
			continue
		}
		entries, ok := streamArr[1].([]any)
		if !ok {
			continue
		}
		for _, entryRaw := range entries {
			entryArr, ok := entryRaw.([]any)
			if !ok || len(entryArr) != 2 {
				continue
			}
			id, _ := entryArr[0].(string)
			kv, ok := entryArr[1].([]any)
			if !ok {
				continue
			}
			values := make(map[string]any, len(kv)/2)
			for i := 0; i+1 < len(kv); i += 2 {
				k, _ := kv[i].(string)
				v := kv[i+1]
				values[k] = v
			}
			messages = append(messages, streamMessage{ID: id, Values: values})
		}
	}
	return messages, nil
}
