package database

import (
	"os"
	"strings"
	"testing"
	"time"

	sqldriver "github.com/go-sql-driver/mysql"
	"github.com/mrvs/backend/internal/config"
	"github.com/mrvs/backend/internal/model"
)

// 本文件覆盖 Open / Migrate / Seed / CHECK 约束，需要真实 MySQL。
// 通过 TEST_DB_INT_DSN 指定独立测试库，未设置时跳过。

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dsn := os.Getenv("TEST_DB_INT_DSN")
	if dsn == "" {
		t.Skip("未设置 TEST_DB_INT_DSN，跳过 Database 集成测试")
	}
	parsed, err := sqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("解析 TEST_DB_INT_DSN 失败: %v", err)
	}
	host, port, ok := strings.Cut(parsed.Addr, ":")
	if !ok {
		t.Fatalf("DSN 地址缺少端口: %s", parsed.Addr)
	}
	return &config.Config{
		AppPort:        "8080",
		DBHost:         host,
		DBPort:         port,
		DBUser:         parsed.User,
		DBPassword:     parsed.Passwd,
		DBName:         parsed.DBName,
		DBMaxOpenConns: 5,
		DBMaxIdleConns: 2,
	}
}

func TestOpenInvalidHost(t *testing.T) {
	cfg := &config.Config{
		AppPort: "8080", DBHost: "127.0.0.1", DBPort: "1", DBUser: "root", DBName: "mrvs",
		DBMaxOpenConns: 1, DBMaxIdleConns: 1,
	}
	if db, err := Open(cfg); err == nil {
		t.Fatalf("预期连接失败，实际成功: %v", db)
	}
}

func TestIntegrationMigrateSeedAndConstraints(t *testing.T) {
	cfg := testConfig(t)
	db, err := Open(cfg)
	if err != nil {
		t.Fatalf("Open 失败: %v", err)
	}
	if err := db.Migrator().DropTable(
		&model.VisitMaterial{}, &model.Visit{}, &model.Product{},
		&model.HCP{}, &model.Hospital{}, &model.MR{},
	); err != nil {
		t.Fatalf("清理表失败: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate 失败: %v", err)
	}
	// 重复迁移必须幂等。
	if err := Migrate(db); err != nil {
		t.Fatalf("重复 Migrate 失败: %v", err)
	}

	if err := Seed(db); err != nil {
		t.Fatalf("Seed 失败: %v", err)
	}
	if err := Seed(db); err != nil {
		t.Fatalf("重复 Seed 失败: %v", err)
	}
	counts := []struct {
		model any
		want  int64
		name  string
	}{
		{&model.MR{}, 2, "mr"},
		{&model.Hospital{}, 3, "hospital"},
		{&model.HCP{}, 6, "hcp"},
		{&model.Product{}, 3, "product"},
	}
	for _, c := range counts {
		var got int64
		if err := db.Model(c.model).Count(&got).Error; err != nil {
			t.Fatalf("统计 %s 失败: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s 数量 = %d, want %d（重复 Seed 不应新增）", c.name, got, c.want)
		}
	}

	for _, c := range checkConstraints {
		exists, err := constraintExists(db, c.Name)
		if err != nil {
			t.Fatalf("查询约束 %s 失败: %v", c.Name, err)
		}
		if !exists {
			t.Fatalf("约束 %s 不存在", c.Name)
		}
	}

	var hospital model.Hospital
	if err := db.Where("name = ?", "北京协和医院").First(&hospital).Error; err != nil {
		t.Fatalf("查询种子医院失败: %v", err)
	}
	invalidHospital := model.Hospital{Name: "非法医院", Latitude: 91, Longitude: 0}
	if err := db.Create(&invalidHospital).Error; err == nil {
		t.Fatal("CHECK 约束未拒绝非法纬度")
	} else if !strings.Contains(err.Error(), "chk_hospital_latitude") && !strings.Contains(strings.ToLower(err.Error()), "check") {
		t.Fatalf("非预期错误: %v", err)
	}

	var hcp model.HCP
	if err := db.Where("hospital_id = ?", hospital.ID).First(&hcp).Error; err != nil {
		t.Fatalf("查询种子医生失败: %v", err)
	}
	var product model.Product
	if err := db.Where("code = ?", "P001").First(&product).Error; err != nil {
		t.Fatalf("查询种子产品失败: %v", err)
	}
	var mr model.MR
	if err := db.Where("employee_no = ?", "M001").First(&mr).Error; err != nil {
		t.Fatalf("查询种子代表失败: %v", err)
	}

	invalidVisit := model.Visit{
		VisitNo: "V-BAD-STATUS", MrID: mr.ID, HcpID: hcp.ID, HospitalID: hospital.ID,
		ProductID: product.ID, Status: "invalid", PlannedAt: time.Now().UTC(),
	}
	err = db.Create(&invalidVisit).Error
	if err == nil {
		t.Fatal("CHECK 约束未拒绝非法状态")
	}
	if !strings.Contains(err.Error(), "chk_visit_status") && !strings.Contains(strings.ToLower(err.Error()), "check") {
		t.Fatalf("非预期错误: %v", err)
	}

	invalidMaterial := model.VisitMaterial{VisitID: 1, MaterialName: "资料", Quantity: -1}
	if err := db.Create(&invalidMaterial).Error; err == nil {
		t.Fatal("CHECK 约束未拒绝非法资料数量")
	}
}
