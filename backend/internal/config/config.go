package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config 保存服务运行所需的全部配置，全部来自环境变量。
type Config struct {
	AppPort string

	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBMaxOpenConns int
	DBMaxIdleConns int

	MinDurationSeconds int
	MaxDistanceMeters  float64

	AllowTimeOverride bool
	Timezone          string
	LogLevel          string
	AutoMigrate       bool
	SeedData          bool
}

// Load 从环境变量读取配置，缺失项使用设计文档中的默认值。
func Load() (*Config, error) {
	cfg := &Config{
		AppPort:            getenv("APP_PORT", "8080"),
		DBHost:             getenv("DB_HOST", "mysql"),
		DBPort:             getenv("DB_PORT", "3306"),
		DBUser:             getenv("DB_USER", "mrvs"),
		DBPassword:         getenv("DB_PASSWORD", "mrvs_pass"),
		DBName:             getenv("DB_NAME", "mrvs"),
		Timezone:           getenv("TZ", "Asia/Shanghai"),
		LogLevel:           strings.ToLower(getenv("LOG_LEVEL", "info")),
		MinDurationSeconds: 300,
		MaxDistanceMeters:  500,
	}

	var err error
	if cfg.DBMaxOpenConns, err = getenvInt("DB_MAX_OPEN_CONNS", 20); err != nil {
		return nil, err
	}
	if cfg.DBMaxIdleConns, err = getenvInt("DB_MAX_IDLE_CONNS", 10); err != nil {
		return nil, err
	}
	if cfg.MinDurationSeconds, err = getenvInt("COMPLIANCE_MIN_DURATION_SECONDS", 300); err != nil {
		return nil, err
	}
	if cfg.MaxDistanceMeters, err = getenvFloat("COMPLIANCE_MAX_DISTANCE_METERS", 500); err != nil {
		return nil, err
	}
	if cfg.AllowTimeOverride, err = getenvBool("ALLOW_TIME_OVERRIDE", false); err != nil {
		return nil, err
	}
	if cfg.AutoMigrate, err = getenvBool("AUTO_MIGRATE", true); err != nil {
		return nil, err
	}
	if cfg.SeedData, err = getenvBool("SEED_DATA", true); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 校验关键配置的取值范围。
func (c *Config) Validate() error {
	if strings.TrimSpace(c.AppPort) == "" {
		return fmt.Errorf("APP_PORT 不能为空")
	}
	if _, err := strconv.Atoi(c.AppPort); err != nil {
		return fmt.Errorf("APP_PORT 必须是数字: %w", err)
	}
	if c.DBHost == "" || c.DBPort == "" || c.DBUser == "" || c.DBName == "" {
		return fmt.Errorf("数据库连接配置不完整")
	}
	if c.DBMaxOpenConns <= 0 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS 必须大于 0")
	}
	if c.DBMaxIdleConns <= 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS 必须大于 0")
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS 不能大于 DB_MAX_OPEN_CONNS")
	}
	if c.MinDurationSeconds <= 0 {
		return fmt.Errorf("COMPLIANCE_MIN_DURATION_SECONDS 必须大于 0")
	}
	if c.MaxDistanceMeters <= 0 {
		return fmt.Errorf("COMPLIANCE_MAX_DISTANCE_METERS 必须大于 0")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL 不合法: %s", c.LogLevel)
	}
	return nil
}

// DSN 构造 MySQL 连接串，时间统一按 UTC 解析。
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s 必须是整数: %w", key, err)
	}
	return v, nil
}

func getenvFloat(key string, def float64) (float64, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是数字: %w", key, err)
	}
	return v, nil
}

func getenvBool(key string, def bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return def, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("%s 必须是布尔值: %s", key, raw)
	}
}
