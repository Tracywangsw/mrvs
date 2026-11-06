package main

import (
	"log/slog"
	"os"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/config"
	"github.com/mrvs/backend/internal/database"
	"github.com/mrvs/backend/internal/repository"
	"github.com/mrvs/backend/internal/router"
	"github.com/mrvs/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("配置加载失败", "error", err)
		os.Exit(1)
	}
	logger := newLogger(cfg.LogLevel)

	db, err := database.Open(cfg)
	if err != nil {
		logger.Error("数据库连接失败", "error", err)
		os.Exit(1)
	}
	if cfg.AutoMigrate {
		if err := database.Migrate(db); err != nil {
			logger.Error("数据库迁移失败", "error", err)
			os.Exit(1)
		}
	}
	if cfg.SeedData {
		if err := database.Seed(db); err != nil {
			logger.Error("种子数据写入失败", "error", err)
			os.Exit(1)
		}
	}

	svc := service.New(repository.New(db), compliance.Rules{
		MinDurationSeconds: cfg.MinDurationSeconds,
		MaxDistanceMeters:  cfg.MaxDistanceMeters,
	}, cfg.AllowTimeOverride)

	hz := router.New(svc, logger, server.WithHostPorts(":"+cfg.AppPort))
	logger.Info("server starting", "port", cfg.AppPort)
	hz.Spin()
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
