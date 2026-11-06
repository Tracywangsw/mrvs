package config

import (
	"strings"
	"testing"
)

var envKeys = []string{
	"APP_PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME",
	"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "COMPLIANCE_MIN_DURATION_SECONDS",
	"COMPLIANCE_MAX_DISTANCE_METERS", "ALLOW_TIME_OVERRIDE", "TZ", "LOG_LEVEL",
	"AUTO_MIGRATE", "SEED_DATA",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range envKeys {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppPort != "8080" || cfg.DBHost != "mysql" || cfg.DBName != "mrvs" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.DBMaxOpenConns != 20 || cfg.DBMaxIdleConns != 10 {
		t.Fatalf("unexpected pool defaults: %+v", cfg)
	}
	if cfg.MinDurationSeconds != 300 || cfg.MaxDistanceMeters != 500 {
		t.Fatalf("unexpected compliance defaults: %+v", cfg)
	}
	if cfg.AllowTimeOverride {
		t.Fatal("ALLOW_TIME_OVERRIDE should default to false")
	}
	if !cfg.AutoMigrate || !cfg.SeedData {
		t.Fatal("AUTO_MIGRATE and SEED_DATA should default to true")
	}
}

func TestLoadOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_NAME", "mrvs_test")
	t.Setenv("DB_MAX_OPEN_CONNS", "5")
	t.Setenv("DB_MAX_IDLE_CONNS", "3")
	t.Setenv("COMPLIANCE_MIN_DURATION_SECONDS", "600")
	t.Setenv("COMPLIANCE_MAX_DISTANCE_METERS", "1000")
	t.Setenv("ALLOW_TIME_OVERRIDE", "true")
	t.Setenv("AUTO_MIGRATE", "0")
	t.Setenv("SEED_DATA", "no")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppPort != "9090" || cfg.DBHost != "127.0.0.1" || cfg.DBPort != "3307" || cfg.DBName != "mrvs_test" {
		t.Fatalf("unexpected overrides: %+v", cfg)
	}
	if cfg.DBMaxOpenConns != 5 || cfg.DBMaxIdleConns != 3 {
		t.Fatalf("unexpected pool: %+v", cfg)
	}
	if cfg.MinDurationSeconds != 600 || cfg.MaxDistanceMeters != 1000 {
		t.Fatalf("unexpected compliance: %+v", cfg)
	}
	if !cfg.AllowTimeOverride || cfg.AutoMigrate || cfg.SeedData {
		t.Fatalf("unexpected switches: %+v", cfg)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %s", cfg.LogLevel)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid DB_MAX_OPEN_CONNS")
	}

	clearEnv(t)
	t.Setenv("ALLOW_TIME_OVERRIDE", "maybe")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid ALLOW_TIME_OVERRIDE")
	}

	clearEnv(t)
	t.Setenv("COMPLIANCE_MIN_DURATION_SECONDS", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for zero min duration")
	}

	clearEnv(t)
	t.Setenv("COMPLIANCE_MAX_DISTANCE_METERS", "-1")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for negative distance")
	}

	clearEnv(t)
	t.Setenv("LOG_LEVEL", "trace")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid log level")
	}

	clearEnv(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "2")
	t.Setenv("DB_MAX_IDLE_CONNS", "5")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when idle > open")
	}
}

func TestValidate(t *testing.T) {
	cfg := &Config{
		AppPort: "8080", DBHost: "mysql", DBPort: "3306", DBUser: "mrvs", DBName: "mrvs",
		DBMaxOpenConns: 20, DBMaxIdleConns: 10, MinDurationSeconds: 300,
		MaxDistanceMeters: 500, LogLevel: "info",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg.AppPort = "abc"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for non-numeric port")
	}
}

func TestDSN(t *testing.T) {
	cfg := &Config{
		DBUser: "mrvs", DBPassword: "mrvs_pass", DBHost: "mysql", DBPort: "3306", DBName: "mrvs",
	}
	dsn := cfg.DSN()
	for _, want := range []string{"mrvs:mrvs_pass@tcp(mysql:3306)/mrvs", "parseTime=true", "loc=UTC", "charset=utf8mb4"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("DSN %q missing %q", dsn, want)
		}
	}
}
