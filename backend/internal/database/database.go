package database

import (
	"fmt"
	"time"

	"github.com/mrvs/backend/internal/config"
	"github.com/mrvs/backend/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open 建立 MySQL 连接并配置连接池。
func Open(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Warn),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库不可用: %w", err)
	}
	return db, nil
}

type checkConstraint struct {
	Table string
	Name  string
	Expr  string
}

var checkConstraints = []checkConstraint{
	{"hospital", "chk_hospital_latitude", "latitude BETWEEN -90 AND 90"},
	{"hospital", "chk_hospital_longitude", "longitude BETWEEN -180 AND 180"},
	{"visit", "chk_visit_status", "status IN ('planned','checked_in','checked_out','reported','cancelled')"},
	{"visit", "chk_visit_checkin_lat", "checkin_lat IS NULL OR checkin_lat BETWEEN -90 AND 90"},
	{"visit", "chk_visit_checkin_lng", "checkin_lng IS NULL OR checkin_lng BETWEEN -180 AND 180"},
	{"visit", "chk_visit_checkout_lat", "checkout_lat IS NULL OR checkout_lat BETWEEN -90 AND 90"},
	{"visit", "chk_visit_checkout_lng", "checkout_lng IS NULL OR checkout_lng BETWEEN -180 AND 180"},
	{"visit", "chk_visit_duration", "duration_seconds IS NULL OR duration_seconds >= 0"},
	{"visit", "chk_visit_deviation", "deviation_meters IS NULL OR deviation_meters >= 0"},
	{"visit_material", "chk_visit_material_quantity", "quantity > 0"},
}

// Migrate 执行 AutoMigrate 并补充 CHECK 约束。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.MR{},
		&model.Hospital{},
		&model.HCP{},
		&model.Product{},
		&model.Visit{},
		&model.VisitMaterial{},
	); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	return addCheckConstraints(db)
}

func addCheckConstraints(db *gorm.DB) error {
	for _, c := range checkConstraints {
		exists, err := constraintExists(db, c.Name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` CHECK (%s)", c.Table, c.Name, c.Expr)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("添加约束 %s 失败: %w", c.Name, err)
		}
	}
	return nil
}

func constraintExists(db *gorm.DB, name string) (bool, error) {
	var count int64
	err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.CHECK_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND CONSTRAINT_NAME = ?",
		name,
	).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询约束 %s 失败: %w", name, err)
	}
	return count > 0, nil
}
