package response

import (
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/mrvs/backend/internal/apperr"
)

// RequestIDKey 是请求上下文中的 request_id 键。
const RequestIDKey = "request_id"

// Envelope 是统一响应包络。
type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

// OK 输出成功响应。
func OK(c *app.RequestContext, status int, data any) {
	c.JSON(status, Envelope{
		Code:      apperr.CodeOK,
		Message:   "ok",
		Data:      data,
		RequestID: RequestID(c),
	})
}

// Fail 输出错误响应。
func Fail(c *app.RequestContext, e *apperr.Error) {
	var data any
	if len(e.Fields) > 0 {
		data = map[string]any{"fields": e.Fields}
	}
	c.JSON(e.Status, Envelope{
		Code:      e.Code,
		Message:   e.Message,
		Data:      data,
		RequestID: RequestID(c),
	})
}

// RequestID 读取中间件写入的请求 ID。
func RequestID(c *app.RequestContext) string {
	if v, ok := c.Get(RequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Health 是健康检查响应体，不套响应包络。
type Health struct {
	Status string `json:"status"`
}

// OKHealth 输出健康检查响应。
func OKHealth(c *app.RequestContext) {
	c.JSON(http.StatusOK, Health{Status: "ok"})
}
