package response

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/mrvs/backend/internal/apperr"
)

func TestResponseHelpers(t *testing.T) {
	hz := server.New()
	hz.GET("/ok", func(ctx context.Context, c *app.RequestContext) {
		c.Set(RequestIDKey, "req-ok")
		OK(c, http.StatusCreated, map[string]any{"id": 1})
	})
	hz.GET("/fail", func(ctx context.Context, c *app.RequestContext) {
		c.Set(RequestIDKey, "req-fail")
		e := apperr.ValidationFields("参数错误", map[string]string{"mr_id": "必填"})
		Fail(c, e)
	})
	hz.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		OKHealth(c)
	})

	w := ut.PerformRequest(hz.Engine, "GET", "/ok", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d", w.Code)
	}
	var okEnv Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &okEnv); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if okEnv.Code != apperr.CodeOK || okEnv.RequestID != "req-ok" || okEnv.Data == nil {
		t.Fatalf("envelope = %+v", okEnv)
	}

	w = ut.PerformRequest(hz.Engine, "GET", "/fail", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
	var failEnv Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &failEnv); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if failEnv.Code != apperr.CodeValidation || failEnv.RequestID != "req-fail" {
		t.Fatalf("envelope = %+v", failEnv)
	}
	fields, ok := failEnv.Data.(map[string]any)["fields"]
	if !ok || fields == nil {
		t.Fatalf("data = %+v", failEnv.Data)
	}

	w = ut.PerformRequest(hz.Engine, "GET", "/health", nil)
	if w.Code != http.StatusOK || w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("health = %d %s", w.Code, w.Body.String())
	}
}

func TestRequestIDEmpty(t *testing.T) {
	c := app.NewContext(0)
	if got := RequestID(c); got != "" {
		t.Fatalf("request id = %q", got)
	}
}
