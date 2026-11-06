package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/response"
)

func TestNewRequestID(t *testing.T) {
	first := NewRequestID()
	if !strings.HasPrefix(first, "req-") || len(first) != len("req-")+16 {
		t.Fatalf("unexpected request id: %q", first)
	}
	second := NewRequestID()
	if first == second {
		t.Fatalf("request ids should differ: %q", first)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	hz := server.New()
	var seen string
	hz.Use(RequestID())
	hz.GET("/id", func(ctx context.Context, c *app.RequestContext) {
		seen = response.RequestID(c)
		c.String(200, "ok")
	})

	w := ut.PerformRequest(hz.Engine, "GET", "/id", nil,
		ut.Header{Key: "X-Request-Id", Value: "req-fixed"})
	if got := string(w.Header().Peek("X-Request-Id")); got != "req-fixed" {
		t.Fatalf("X-Request-Id = %q", got)
	}
	if seen != "req-fixed" {
		t.Fatalf("request id in context = %q", seen)
	}

	seen = ""
	w = ut.PerformRequest(hz.Engine, "GET", "/id", nil)
	generated := string(w.Header().Peek("X-Request-Id"))
	if !strings.HasPrefix(generated, "req-") || seen != generated {
		t.Fatalf("generated id = %q, context = %q", generated, seen)
	}
}

func TestRecoverMiddleware(t *testing.T) {
	hz := server.New()
	hz.Use(Recover(slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))))
	hz.GET("/panic", func(ctx context.Context, c *app.RequestContext) {
		panic("boom")
	})

	w := ut.PerformRequest(hz.Engine, "GET", "/panic", nil)
	if w.Code != 500 {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), apperr.CodeInternal) {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestCORSMiddleware(t *testing.T) {
	hz := server.New()
	hz.Use(CORS())
	hz.GET("/cors", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "ok")
	})
	w := ut.PerformRequest(hz.Engine, "GET", "/cors", nil)
	if got := string(w.Header().Peek("Access-Control-Allow-Origin")); got != "*" {
		t.Fatalf("allow origin = %q", got)
	}
	if got := string(w.Header().Peek("Access-Control-Allow-Methods")); !strings.Contains(got, "PUT") {
		t.Fatalf("allow methods = %q", got)
	}

	w = ut.PerformRequest(hz.Engine, "OPTIONS", "/cors", nil)
	if w.Code != 204 {
		t.Fatalf("preflight status = %d", w.Code)
	}
	if body := w.Body.String(); body != "" {
		t.Fatalf("preflight body = %q", body)
	}
}

func TestAccessLogMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	hz := server.New()
	hz.Use(RequestID(), AccessLog(logger))
	hz.GET("/log", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "ok")
	})

	ut.PerformRequest(hz.Engine, "GET", "/log", nil,
		ut.Header{Key: "X-Request-Id", Value: "req-log"})

	out := buf.String()
	for _, want := range []string{"http_request", "/log", "req-log", "200"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log %q missing %q", out, want)
		}
	}
}
