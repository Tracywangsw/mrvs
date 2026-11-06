package router

import (
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/mrvs/backend/internal/handler"
	"github.com/mrvs/backend/internal/middleware"
	"github.com/mrvs/backend/internal/service"
	"github.com/mrvs/backend/internal/web"
)

// New 构造带中间件与全部路由的 Hertz 实例。
func New(svc service.Service, log *slog.Logger, opts ...config.Option) *server.Hertz {
	if log == nil {
		log = slog.Default()
	}
	hz := server.New(opts...)
	hz.Use(
		middleware.RequestID(),
		middleware.CORS(),
		middleware.Recover(log),
		middleware.AccessLog(log),
	)

	h := handler.New(svc)
	hz.GET("/healthz", h.Health)

	api := hz.Group("/api/v1")
	api.POST("/visits", h.Create)
	api.GET("/visits", h.List)
	api.GET("/visits/:id", h.Get)
	api.POST("/visits/:id/check-in", h.CheckIn)
	api.POST("/visits/:id/check-out", h.CheckOut)
	api.PUT("/visits/:id/report", h.Report)
	api.POST("/visits/:id/cancel", h.Cancel)
	api.GET("/dashboard/products", h.DashboardProducts)
	api.GET("/meta/mrs", h.MetaMRs)
	api.GET("/meta/hospitals", h.MetaHospitals)
	api.GET("/meta/hcps", h.MetaHCPs)
	api.GET("/meta/products", h.MetaProducts)

	// 生产镜像中前端静态资源被打包进二进制，未命中路由时回退到 index.html。
	if static, ok := web.Handler(); ok {
		hz.NoRoute(static)
	}

	return hz
}
