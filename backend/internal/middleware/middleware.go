package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/response"
)

// NewRequestID 生成 req- 前缀的 16 位十六进制请求 ID。
func NewRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "req-0000000000000000"
	}
	return "req-" + hex.EncodeToString(b)
}

// RequestID 注入或透传请求 ID。
func RequestID() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		id := string(c.Request.Header.Peek("X-Request-Id"))
		if id == "" {
			id = NewRequestID()
		}
		c.Set(response.RequestIDKey, id)
		c.Response.Header.Set("X-Request-Id", id)
		c.Next(ctx)
	}
}

// Recover 捕获 panic 并输出统一的 500 响应。
func Recover(log *slog.Logger) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "panic", r, "request_id", response.RequestID(c), "path", string(c.Path()))
				response.Fail(c, apperr.Internal("服务端内部错误"))
			}
		}()
		c.Next(ctx)
	}
}

// CORS 允许跨域访问，并处理预检请求。
func CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Response.Header.Set("Access-Control-Allow-Origin", "*")
		c.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		c.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, X-Request-Id")
		if string(c.Method()) == "OPTIONS" {
			c.Response.SetStatusCode(204)
			return
		}
		c.Next(ctx)
	}
}

// AccessLog 输出结构化访问日志。
func AccessLog(log *slog.Logger) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)
		log.Info("http_request",
			"method", string(c.Method()),
			"path", string(c.Path()),
			"status", c.Response.StatusCode(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", response.RequestID(c),
		)
	}
}
