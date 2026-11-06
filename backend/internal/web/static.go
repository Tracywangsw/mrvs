package web

import (
	"context"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/response"
)

// assets 由 Docker 构建阶段把 frontend/dist 复制到本目录后一同打包进二进制。
//
//go:embed all:dist
var assets embed.FS

// Handler 返回托管前端静态资源的 NoRoute 处理器。
// 本地开发未构建前端时返回 false，此时前端由 Vite 提供。
func Handler() (app.HandlerFunc, bool) {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		return nil, false
	}
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, false
	}

	return func(_ context.Context, c *app.RequestContext) {
		requestPath := strings.TrimPrefix(string(c.Path()), "/")
		if strings.HasPrefix(requestPath, "api/") || requestPath == "healthz" {
			response.Fail(c, apperr.NotFound("接口不存在"))
			return
		}
		if requestPath != "" {
			name := path.Clean(requestPath)
			if data, readErr := fs.ReadFile(sub, name); readErr == nil {
				serveFile(c, name, data)
				return
			}
		}
		// SPA 路由（如 /visits/7）回退到 index.html。
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	}, true
}

func serveFile(c *app.RequestContext, name string, data []byte) {
	contentType := mime.TypeByExtension(filepath.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if strings.HasPrefix(name, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	c.Data(http.StatusOK, contentType, data)
}
