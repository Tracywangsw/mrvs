package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/mrvs/backend/internal/database"
	"github.com/mrvs/backend/internal/model"
)

// 本文件是针对 GORM 实现的集成测试，需要真实 MySQL。
// 通过 TEST_REPO_DB_DSN 指定独立测试库，未设置时整组用例跳过，避免误删业务数据。

type repoFixture struct {
	mrActive     uint64
	mrOther      uint64
	mrInactive   uint64
	hospitalA    uint64
	hospitalB    uint64
	hospitalOff  uint64
	hcpA1        uint64
	hcpA2        uint64
	hcpB1        uint64
	hcpOff       uint64
	productA     uint64
	productB     uint64
	productOff   uint64
	hospitalALat float64
	hospitalALng float64
	hospitalBLat float64
	hospitalBLng float64
}

var (
	repoOnce sync.Once
	repoInst Repository
	repoDB   *gorm.DB
	repoErr  error
	fx       repoFixture
)

func setupRepo(t *testing.T) Repository {
	t.Helper()
	dsn := os.Getenv("TEST_REPO_DB_DSN")
	if dsn == "" {
		t.Skip("未设置 TEST_REPO_DB_DSN，跳过 Repository 集成测试")
	}
	repoOnce.Do(func() { repoErr = bootstrapRepo(dsn) })
	if repoErr != nil {
		t.Fatalf("初始化测试数据库失败: %v", repoErr)
	}
	return repoInst
}

func bootstrapRepo(dsn string) error {
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return err
	}
	if err := db.Migrator().DropTable(
		&model.VisitMaterial{}, &model.Visit{}, &model.Product{},
		&model.HCP{}, &model.Hospital{}, &model.MR{},
	); err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return err
	}
	repoDB = db
	repoInst = New(db)
	return seedRepoFixture(db)
}

func seedRepoFixture(db *gorm.DB) error {
	region := "华东"
	addr := "测试地址"
	title := "主任医师"

	mrActive := model.MR{EmployeeNo: "M-A", Name: "代表A", Region: &region}
	mrOther := model.MR{EmployeeNo: "M-B", Name: "代表B"}
	mrOff := model.MR{EmployeeNo: "M-I", Name: "停用代表"}
	hospitalA := model.Hospital{Name: "医院A", Address: &addr, Latitude: 39.912345, Longitude: 116.410123}
	hospitalB := model.Hospital{Name: "医院B", Address: &addr, Latitude: 31.211000, Longitude: 121.466000}
	hospitalOff := model.Hospital{Name: "停用医院", Address: &addr, Latitude: 23.129000, Longitude: 113.285000}
	productA := model.Product{Code: "PA", Name: "产品A"}
	productB := model.Product{Code: "PB", Name: "产品B"}
	productOff := model.Product{Code: "PI", Name: "停用产品"}

	for _, row := range []any{
		&mrActive, &mrOther, &mrOff,
		&hospitalA, &hospitalB, &hospitalOff,
		&productA, &productB, &productOff,
	} {
		if err := db.Create(row).Error; err != nil {
			return err
		}
	}

	hcpA1 := model.HCP{HospitalID: hospitalA.ID, Name: "医生A1", Department: "心内科", Title: &title}
	hcpA2 := model.HCP{HospitalID: hospitalA.ID, Name: "医生A2", Department: "呼吸科"}
	hcpB1 := model.HCP{HospitalID: hospitalB.ID, Name: "医生B1", Department: "内分泌科"}
	hcpOff := model.HCP{HospitalID: hospitalA.ID, Name: "停用医生", Department: "心内科"}
	for _, row := range []any{&hcpA1, &hcpA2, &hcpB1, &hcpOff} {
		if err := db.Create(row).Error; err != nil {
			return err
		}
	}

	for _, row := range []any{
		&model.MR{ID: mrOff.ID, IsActive: false},
		&model.Hospital{ID: hospitalOff.ID, IsActive: false},
		&model.Product{ID: productOff.ID, IsActive: false},
		&model.HCP{ID: hcpOff.ID, IsActive: false},
	} {
		if err := db.Model(row).Update("is_active", false).Error; err != nil {
			return err
		}
	}

	fx = repoFixture{
		mrActive: mrActive.ID, mrOther: mrOther.ID, mrInactive: mrOff.ID,
		hospitalA: hospitalA.ID, hospitalB: hospitalB.ID, hospitalOff: hospitalOff.ID,
		hcpA1: hcpA1.ID, hcpA2: hcpA2.ID, hcpB1: hcpB1.ID, hcpOff: hcpOff.ID,
		productA: productA.ID, productB: productB.ID, productOff: productOff.ID,
		hospitalALat: hospitalA.Latitude, hospitalALng: hospitalA.Longitude,
		hospitalBLat: hospitalB.Latitude, hospitalBLng: hospitalB.Longitude,
	}
	return nil
}

func truncateVisits(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		"SET FOREIGN_KEY_CHECKS=0",
		"TRUNCATE TABLE visit_material",
		"TRUNCATE TABLE visit",
		"SET FOREIGN_KEY_CHECKS=1",
	} {
		if err := repoDB.Exec(q).Error; err != nil {
			t.Fatalf("执行 %q 失败: %v", q, err)
		}
	}
}

var visitSeq uint64

func newVisit(status string, planned time.Time, mrID, hcpID, hospitalID, productID uint64) *model.Visit {
	visitSeq++
	return &model.Visit{
		VisitNo:    fmt.Sprintf("V-REPO-%06d", visitSeq),
		MrID:       mrID,
		HcpID:      hcpID,
		HospitalID: hospitalID,
		ProductID:  productID,
		PlannedAt:  planned,
		Status:     status,
	}
}

func insertVisit(t *testing.T, repo Repository, v *model.Visit) *model.Visit {
	t.Helper()
	if err := repo.CreateVisit(context.Background(), v); err != nil {
		t.Fatalf("创建拜访失败: %v", err)
	}
	return v
}

func ptr[T any](v T) *T { return &v }

func TestGormMasterData(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	mr, err := repo.FindMR(ctx, fx.mrActive)
	if err != nil || mr == nil || mr.Name != "代表A" || !mr.IsActive {
		t.Fatalf("FindMR = %+v, err = %v", mr, err)
	}
	if mr, err := repo.FindMR(ctx, 999999); err != nil || mr != nil {
		t.Fatalf("missing FindMR = %+v, err = %v", mr, err)
	}

	hcp, err := repo.FindHCP(ctx, fx.hcpA1)
	if err != nil || hcp == nil || hcp.Name != "医生A1" {
		t.Fatalf("FindHCP = %+v, err = %v", hcp, err)
	}
	if hcp, err := repo.FindHCP(ctx, 999999); err != nil || hcp != nil {
		t.Fatalf("missing FindHCP = %+v, err = %v", hcp, err)
	}

	hospital, err := repo.FindHospital(ctx, fx.hospitalA)
	if err != nil || hospital == nil || hospital.Name != "医院A" {
		t.Fatalf("FindHospital = %+v, err = %v", hospital, err)
	}
	if hospital, err := repo.FindHospital(ctx, 999999); err != nil || hospital != nil {
		t.Fatalf("missing FindHospital = %+v, err = %v", hospital, err)
	}

	product, err := repo.FindProduct(ctx, fx.productA)
	if err != nil || product == nil || product.Code != "PA" {
		t.Fatalf("FindProduct = %+v, err = %v", product, err)
	}
	if product, err := repo.FindProduct(ctx, 999999); err != nil || product != nil {
		t.Fatalf("missing FindProduct = %+v, err = %v", product, err)
	}

	mrs, err := repo.ListMRs(ctx)
	if err != nil || len(mrs) != 2 {
		t.Fatalf("ListMRs = %+v, err = %v", mrs, err)
	}
	for _, row := range mrs {
		if !row.IsActive {
			t.Fatalf("ListMRs 包含停用数据: %+v", row)
		}
	}

	hospitals, err := repo.ListHospitals(ctx)
	if err != nil || len(hospitals) != 2 {
		t.Fatalf("ListHospitals = %+v, err = %v", hospitals, err)
	}

	allHCPs, err := repo.ListHCPs(ctx, nil)
	if err != nil || len(allHCPs) != 3 {
		t.Fatalf("ListHCPs(nil) = %+v, err = %v", allHCPs, err)
	}
	hospitalAHCPs, err := repo.ListHCPs(ctx, &fx.hospitalA)
	if err != nil || len(hospitalAHCPs) != 2 {
		t.Fatalf("ListHCPs(A) = %+v, err = %v", hospitalAHCPs, err)
	}
	for _, row := range hospitalAHCPs {
		if row.HospitalID != fx.hospitalA || !row.IsActive {
			t.Fatalf("ListHCPs(A) 过滤错误: %+v", row)
		}
	}

	products, err := repo.ListProducts(ctx)
	if err != nil || len(products) != 2 {
		t.Fatalf("ListProducts = %+v, err = %v", products, err)
	}
}

func TestGormVisitLifecycleAndMaterials(t *testing.T) {
	repo := setupRepo(t)
	truncateVisits(t)
	ctx := context.Background()

	visit := insertVisit(t, repo, newVisit(model.StatusPlanned,
		time.Date(2026, 11, 5, 1, 0, 0, 0, time.UTC),
		fx.mrActive, fx.hcpA1, fx.hospitalA, fx.productA))
	if visit.ID == 0 {
		t.Fatal("CreateVisit 未回填主键")
	}

	got, err := repo.GetVisit(ctx, visit.ID)
	if err != nil || got == nil || got.VisitNo != visit.VisitNo {
		t.Fatalf("GetVisit = %+v, err = %v", got, err)
	}
	if got, err := repo.GetVisit(ctx, 999999); err != nil || got != nil {
		t.Fatalf("missing GetVisit = %+v, err = %v", got, err)
	}

	if err := repo.Transaction(ctx, func(tx Repository) error {
		locked, err := tx.LockVisit(ctx, visit.ID)
		if err != nil || locked == nil || locked.Status != model.StatusPlanned {
			return fmt.Errorf("LockVisit = %+v, err = %v", locked, err)
		}
		if locked, err := tx.LockVisit(ctx, 999999); err != nil || locked != nil {
			return fmt.Errorf("missing LockVisit = %+v, err = %v", locked, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	affected, err := repo.UpdateVisit(ctx, visit.ID, model.StatusPlanned, map[string]any{
		"status":                  model.StatusCheckedIn,
		"checkin_at":              time.Date(2026, 11, 5, 1, 5, 0, 0, time.UTC),
		"checkin_lat":             fx.hospitalALat,
		"checkin_lng":             fx.hospitalALng,
		"checkin_distance_meters": 0.0,
	})
	if err != nil || affected != 1 {
		t.Fatalf("UpdateVisit affected = %d, err = %v", affected, err)
	}
	affected, err = repo.UpdateVisit(ctx, visit.ID, model.StatusPlanned, map[string]any{"status": model.StatusCancelled})
	if err != nil || affected != 0 {
		t.Fatalf("条件更新 affected = %d, err = %v", affected, err)
	}
	got, _ = repo.GetVisit(ctx, visit.ID)
	if got.Status != model.StatusCheckedIn {
		t.Fatalf("status = %s", got.Status)
	}

	materials := []model.VisitMaterial{
		{VisitID: visit.ID, MaterialName: "临床指南", MaterialType: ptr("guideline"), Quantity: 2},
		{VisitID: visit.ID, MaterialName: "产品手册", Quantity: 1},
	}
	if err := repo.CreateMaterials(ctx, materials); err != nil {
		t.Fatalf("CreateMaterials = %v", err)
	}
	if err := repo.CreateMaterials(ctx, nil); err != nil {
		t.Fatalf("CreateMaterials(nil) = %v", err)
	}
	listed, err := repo.ListMaterials(ctx, visit.ID)
	if err != nil || len(listed) != 2 || listed[0].MaterialName != "临床指南" {
		t.Fatalf("ListMaterials = %+v, err = %v", listed, err)
	}
	if err := repo.DeleteMaterials(ctx, visit.ID); err != nil {
		t.Fatalf("DeleteMaterials = %v", err)
	}
	listed, err = repo.ListMaterials(ctx, visit.ID)
	if err != nil || len(listed) != 0 {
		t.Fatalf("删除后 ListMaterials = %+v, err = %v", listed, err)
	}

	rollback := errors.New("rollback")
	if err := repo.Transaction(ctx, func(tx Repository) error {
		if _, err := tx.UpdateVisit(ctx, visit.ID, model.StatusCheckedIn, map[string]any{"status": model.StatusCancelled}); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("Transaction err = %v", err)
	}
	got, _ = repo.GetVisit(ctx, visit.ID)
	if got.Status != model.StatusCheckedIn {
		t.Fatalf("事务未回滚, status = %s", got.Status)
	}
}

func TestGormListVisits(t *testing.T) {
	repo := setupRepo(t)
	truncateVisits(t)
	ctx := context.Background()

	base := func(status string, hour int, productID uint64) *model.Visit {
		return newVisit(status, time.Date(2026, 11, 5, hour, 0, 0, 0, time.UTC),
			fx.mrActive, fx.hcpA1, fx.hospitalA, productID)
	}
	planned := insertVisit(t, repo, base(model.StatusPlanned, 9, fx.productA))
	if err := repoDB.Model(&model.Visit{}).Where("id = ?", planned.ID).Update("is_abnormal", false).Error; err != nil {
		t.Fatal(err)
	}
	checkedIn := insertVisit(t, repo, base(model.StatusCheckedIn, 10, fx.productA))
	abnormal := insertVisit(t, repo, base(model.StatusCheckedOut, 11, fx.productA))
	if err := repoDB.Model(&model.Visit{}).Where("id = ?", abnormal.ID).
		Updates(map[string]any{"is_abnormal": true, "checkout_at": time.Date(2026, 11, 5, 11, 10, 0, 0, time.UTC), "duration_seconds": 600}).
		Error; err != nil {
		t.Fatal(err)
	}
	cancelled := insertVisit(t, repo, base(model.StatusCancelled, 8, fx.productB))

	ctxPage := func(filter VisitListFilter) ([]VisitListRecord, int64) {
		rows, total, err := repo.ListVisits(ctx, filter)
		if err != nil {
			t.Fatalf("ListVisits(%+v) = %v", filter, err)
		}
		return rows, total
	}

	rows, total := ctxPage(VisitListFilter{Status: ptr(model.StatusPlanned), Page: 1, PageSize: 20})
	if total != 1 || len(rows) != 1 || rows[0].ID != planned.ID {
		t.Fatalf("status 过滤 total = %d, rows = %+v", total, rows)
	}

	rows, total = ctxPage(VisitListFilter{Abnormal: ptr(true), Page: 1, PageSize: 20})
	if total != 1 || len(rows) != 1 || rows[0].ID != abnormal.ID {
		t.Fatalf("abnormal 过滤 total = %d, rows = %+v", total, rows)
	}
	rows, total = ctxPage(VisitListFilter{Abnormal: ptr(false), Page: 1, PageSize: 20})
	if total != 1 || len(rows) != 1 || rows[0].ID != planned.ID {
		t.Fatalf("normal 过滤 total = %d, rows = %+v", total, rows)
	}

	from := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 5, 11, 0, 0, 0, time.UTC)
	rows, total = ctxPage(VisitListFilter{PlannedFrom: &from, PlannedTo: &to, Page: 1, PageSize: 20})
	if total != 2 {
		t.Fatalf("日期过滤 total = %d, rows = %+v", total, rows)
	}

	rows, total = ctxPage(VisitListFilter{MrID: &fx.mrActive, HcpID: &fx.hcpA1, HospitalID: &fx.hospitalA,
		ProductID: &fx.productA, Page: 1, PageSize: 20})
	if total != 3 {
		t.Fatalf("组合过滤 total = %d", total)
	}

	rows, total = ctxPage(VisitListFilter{Page: 1, PageSize: 2})
	if total != 4 || len(rows) != 2 {
		t.Fatalf("分页 total = %d, len = %d", total, len(rows))
	}
	if rows[0].ID != abnormal.ID || rows[1].ID != checkedIn.ID {
		t.Fatalf("排序错误: %+v", rows)
	}
	if rows[0].MrName != "代表A" || rows[0].HcpName != "医生A1" ||
		rows[0].HospitalName != "医院A" || rows[0].ProductName != "产品A" {
		t.Fatalf("join 字段错误: %+v", rows[0])
	}
	if rows[0].CheckOutAt == nil || rows[0].DurationSeconds == nil || *rows[0].DurationSeconds != 600 {
		t.Fatalf("签退字段错误: %+v", rows[0])
	}

	rows, total = ctxPage(VisitListFilter{ProductID: &fx.productB, Page: 1, PageSize: 20})
	if total != 1 || rows[0].ID != cancelled.ID {
		t.Fatalf("产品过滤 total = %d, rows = %+v", total, rows)
	}
}

func TestGormProductStats(t *testing.T) {
	repo := setupRepo(t)
	truncateVisits(t)
	ctx := context.Background()

	checkoutA1 := time.Date(2026, 11, 5, 2, 0, 0, 0, time.UTC)
	checkoutA2 := time.Date(2026, 11, 5, 3, 0, 0, 0, time.UTC)
	checkoutB := time.Date(2026, 11, 5, 4, 0, 0, 0, time.UTC)
	checkoutOld := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	insert := func(productID uint64, status string, checkoutAt time.Time, abnormal bool) {
		v := newVisit(status, checkoutAt, fx.mrActive, fx.hcpA1, fx.hospitalA, productID)
		v.CheckOutAt = &checkoutAt
		v.IsAbnormal = ptr(abnormal)
		insertVisit(t, repo, v)
	}
	insert(fx.productA, model.StatusCheckedOut, checkoutA1, false)
	insert(fx.productA, model.StatusReported, checkoutA2, true)
	insert(fx.productA, model.StatusPlanned, checkoutA2, false)
	insert(fx.productB, model.StatusCheckedOut, checkoutB, false)
	otherMR := newVisit(model.StatusReported, checkoutOld, fx.mrOther, fx.hcpB1, fx.hospitalB, fx.productB)
	otherMR.CheckOutAt = &checkoutOld
	otherMR.IsAbnormal = ptr(false)
	insertVisit(t, repo, otherMR)

	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	rows, err := repo.ProductStats(ctx, ProductStatFilter{From: from, To: to})
	if err != nil {
		t.Fatalf("ProductStats = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	byID := map[uint64]ProductStatRow{}
	for _, row := range rows {
		byID[row.ProductID] = row
	}
	a := byID[fx.productA]
	if a.TotalVisits != 2 || a.NormalVisits != 1 || a.AbnormalVisits != 1 || a.ProductCode != "PA" || a.ProductName != "产品A" {
		t.Fatalf("productA stat = %+v", a)
	}
	b := byID[fx.productB]
	if b.TotalVisits != 1 || b.NormalVisits != 1 || b.AbnormalVisits != 0 {
		t.Fatalf("productB stat = %+v", b)
	}
	if rows[0].ProductID != fx.productA {
		t.Fatalf("排序错误: %+v", rows)
	}

	rows, err = repo.ProductStats(ctx, ProductStatFilter{From: from, To: to, MrID: &fx.mrActive})
	if err != nil || len(rows) != 2 || rows[0].TotalVisits != 2 {
		t.Fatalf("mr 过滤 rows = %+v, err = %v", rows, err)
	}

	rows, err = repo.ProductStats(ctx, ProductStatFilter{From: from, To: to, ProductID: &fx.productB})
	if err != nil || len(rows) != 1 || rows[0].ProductID != fx.productB || rows[0].TotalVisits != 1 {
		t.Fatalf("product 过滤 rows = %+v, err = %v", rows, err)
	}

	rows, err = repo.ProductStats(ctx, ProductStatFilter{From: from, To: from})
	if err != nil || len(rows) != 0 {
		t.Fatalf("空区间 rows = %+v, err = %v", rows, err)
	}
}
