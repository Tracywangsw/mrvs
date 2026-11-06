package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	sqldriver "github.com/go-sql-driver/mysql"
	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/database"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/model"
	"github.com/mrvs/backend/internal/repository"
	"github.com/mrvs/backend/internal/router"
	"github.com/mrvs/backend/internal/service"
	"github.com/mrvs/backend/internal/visitno"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type envelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
}

type fixture struct {
	mr1       uint64
	mr2       uint64
	hcp1      uint64
	hcp2      uint64
	hospital1 uint64
	hospital2 uint64
	product1  uint64
	product2  uint64
	product3  uint64
	lat       float64
	lng       float64
	lat2      float64
	lng2      float64
}

var (
	testDB     *gorm.DB
	baseURL    string
	fx         fixture
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

// resolveTestDSN 优先使用 TEST_DB_DSN；未显式设置时自动探测本机 3306 / 3307。
func resolveTestDSN() (string, bool) {
	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		return dsn, true
	}
	for _, port := range []string{"3306", "3307"} {
		dsn := fmt.Sprintf("root@tcp(127.0.0.1:%s)/mrvs_test?charset=utf8mb4&parseTime=true&loc=UTC", port)
		db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			continue
		}
		sqlDB, err := db.DB()
		if err != nil {
			continue
		}
		err = sqlDB.Ping()
		_ = sqlDB.Close()
		if err == nil {
			return dsn, true
		}
	}
	return "", false
}

func TestMain(m *testing.M) {
	dsn, ok := resolveTestDSN()
	if !ok {
		fmt.Fprintln(os.Stderr, "未发现可用的测试数据库，E2E 已跳过；请设置 TEST_DB_DSN 后重试，例如：")
		fmt.Fprintln(os.Stderr, "  TEST_DB_DSN='root@tcp(127.0.0.1:3306)/mrvs_test?charset=utf8mb4&parseTime=true&loc=UTC' go test ./test/e2e -v")
		os.Exit(0)
	}
	cfg, err := sqldriver.ParseDSN(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析 TEST_DB_DSN 失败: %v\n", err)
		os.Exit(1)
	}
	if !strings.Contains(strings.ToLower(cfg.DBName), "test") && os.Getenv("E2E_ALLOW_NON_TEST_DB") != "1" {
		fmt.Fprintf(os.Stderr, "拒绝在非测试库 %q 上执行 E2E；如确认可设置 E2E_ALLOW_NON_TEST_DB=1\n", cfg.DBName)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "E2E 使用测试数据库: %s\n", dsn)

	testDB, err = gorm.Open(gormmysql.Open(dsn), &gorm.Config{NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接测试数据库失败: %v\n", err)
		os.Exit(1)
	}
	if err := testDB.Migrator().DropTable(
		&model.VisitMaterial{}, &model.Visit{}, &model.Product{},
		&model.HCP{}, &model.Hospital{}, &model.MR{},
	); err != nil {
		fmt.Fprintf(os.Stderr, "清理测试表失败: %v\n", err)
		os.Exit(1)
	}
	if err := database.Migrate(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "迁移测试数据库失败: %v\n", err)
		os.Exit(1)
	}
	if err := database.Seed(testDB); err != nil {
		fmt.Fprintf(os.Stderr, "写入种子数据失败: %v\n", err)
		os.Exit(1)
	}
	if err := loadFixture(); err != nil {
		fmt.Fprintf(os.Stderr, "加载测试固件失败: %v\n", err)
		os.Exit(1)
	}

	svc := service.New(repository.New(testDB), compliance.Rules{
		MinDurationSeconds: 300, MaxDistanceMeters: 500,
	}, true)

	addr := freeAddr()
	hz := router.New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)), server.WithHostPorts(addr))
	baseURL = "http://" + addr
	go hz.Spin()

	if err := waitHealthy(baseURL, 15*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "E2E 服务启动失败: %v\n", err)
		_ = hz.Engine.Close()
		os.Exit(1)
	}

	code := m.Run()
	_ = hz.Engine.Close()
	os.Exit(code)
}

func loadFixture() error {
	var mr1, mr2 model.MR
	if err := testDB.Where("employee_no = ?", "M001").First(&mr1).Error; err != nil {
		return err
	}
	if err := testDB.Where("employee_no = ?", "M002").First(&mr2).Error; err != nil {
		return err
	}
	var h1, h2 model.Hospital
	if err := testDB.Where("name = ?", "北京协和医院").First(&h1).Error; err != nil {
		return err
	}
	if err := testDB.Where("name = ?", "上海交通大学医学院附属瑞金医院").First(&h2).Error; err != nil {
		return err
	}
	var hcp1, hcp2 model.HCP
	if err := testDB.Where("hospital_id = ?", h1.ID).Order("id ASC").First(&hcp1).Error; err != nil {
		return err
	}
	if err := testDB.Where("hospital_id = ?", h2.ID).Order("id ASC").First(&hcp2).Error; err != nil {
		return err
	}
	var p1, p2, p3 model.Product
	if err := testDB.Where("code = ?", "P001").First(&p1).Error; err != nil {
		return err
	}
	if err := testDB.Where("code = ?", "P002").First(&p2).Error; err != nil {
		return err
	}
	if err := testDB.Where("code = ?", "P003").First(&p3).Error; err != nil {
		return err
	}
	fx = fixture{
		mr1: mr1.ID, mr2: mr2.ID, hcp1: hcp1.ID, hcp2: hcp2.ID,
		hospital1: h1.ID, hospital2: h2.ID,
		product1: p1.ID, product2: p2.ID, product3: p3.ID,
		lat: h1.Latitude, lng: h1.Longitude, lat2: h2.Latitude, lng2: h2.Longitude,
	}
	return nil
}

func freeAddr() string {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "127.0.0.1:18080"
	}
	addr := lis.Addr().String()
	_ = lis.Close()
	return addr
}

func waitHealthy(base string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(base + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("healthz 未在 %s 内就绪", timeout)
}

func resetVisits(t *testing.T) {
	t.Helper()
	exec := func(query string) {
		if err := testDB.Exec(query).Error; err != nil {
			t.Fatalf("执行 %q 失败: %v", query, err)
		}
	}
	exec("SET FOREIGN_KEY_CHECKS=0")
	exec("TRUNCATE TABLE visit_material")
	exec("TRUNCATE TABLE visit")
	exec("SET FOREIGN_KEY_CHECKS=1")
}

func call(t *testing.T, method, path string, body any, headers map[string]string) (int, envelope, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求失败: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("请求 %s %s 失败: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}
	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, envelope{}, raw
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, string(raw))
	}
	if env.RequestID == "" {
		t.Fatalf("响应缺少 request_id: %s", string(raw))
	}
	return resp.StatusCode, env, raw
}

func decode[T any](t *testing.T, env envelope) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		t.Fatalf("解析 data 失败: %v, data=%s", err, string(env.Data))
	}
	return out
}

func expectStatus(t *testing.T, got, want int, env envelope) {
	t.Helper()
	if got != want {
		t.Fatalf("status = %d, want %d, code=%s message=%s", got, want, env.Code, env.Message)
	}
}

func expectCode(t *testing.T, env envelope, code string) {
	t.Helper()
	if env.Code != code {
		t.Fatalf("code = %s, want %s (message=%s)", env.Code, code, env.Message)
	}
}

func createVisit(t *testing.T, mrID, hcpID, hospitalID, productID uint64, plannedAt string) dto.VisitDetail {
	t.Helper()
	status, env, _ := call(t, http.MethodPost, "/api/v1/visits", map[string]any{
		"mr_id": mrID, "hcp_id": hcpID, "hospital_id": hospitalID, "product_id": productID,
		"planned_at": plannedAt,
	}, nil)
	expectStatus(t, status, http.StatusCreated, env)
	expectCode(t, env, "OK")
	detail := decode[dto.VisitDetail](t, env)
	if !visitno.Valid(detail.VisitNo) {
		t.Fatalf("visit_no 格式错误: %s", detail.VisitNo)
	}
	return detail
}

func checkIn(t *testing.T, id uint64, at string, lat, lng float64) dto.VisitDetail {
	t.Helper()
	status, env, _ := call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-in", id), map[string]any{
		"latitude": lat, "longitude": lng, "occurred_at": at,
	}, nil)
	expectStatus(t, status, http.StatusOK, env)
	return decode[dto.VisitDetail](t, env)
}

func checkOut(t *testing.T, id uint64, at string, lat, lng float64) dto.VisitDetail {
	t.Helper()
	status, env, _ := call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-out", id), map[string]any{
		"latitude": lat, "longitude": lng, "occurred_at": at,
	}, nil)
	expectStatus(t, status, http.StatusOK, env)
	return decode[dto.VisitDetail](t, env)
}

func report(t *testing.T, id uint64, body map[string]any) dto.VisitDetail {
	t.Helper()
	status, env, _ := call(t, http.MethodPut, fmt.Sprintf("/api/v1/visits/%d/report", id), body, nil)
	expectStatus(t, status, http.StatusOK, env)
	return decode[dto.VisitDetail](t, env)
}

func TestHealthAndMeta(t *testing.T) {
	healthResp, err := httpClient.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("healthz 请求失败: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != 200 {
		t.Fatalf("healthz status = %d", healthResp.StatusCode)
	}

	_, env, _ := call(t, http.MethodGet, "/api/v1/meta/mrs", nil, nil)
	mrs := decode[dto.Items[dto.MRItem]](t, env)
	if len(mrs.Items) != 2 || mrs.Items[0].EmployeeNo != "M001" {
		t.Fatalf("mrs = %+v", mrs.Items)
	}

	_, env, _ = call(t, http.MethodGet, "/api/v1/meta/hospitals", nil, nil)
	hospitals := decode[dto.Items[dto.HospitalItem]](t, env)
	if len(hospitals.Items) != 3 {
		t.Fatalf("hospitals = %+v", hospitals.Items)
	}

	_, env, _ = call(t, http.MethodGet, fmt.Sprintf("/api/v1/meta/hcps?hospital_id=%d", fx.hospital1), nil, nil)
	hcps := decode[dto.Items[dto.HCPItem]](t, env)
	if len(hcps.Items) != 2 {
		t.Fatalf("hcps = %+v", hcps.Items)
	}

	_, env, _ = call(t, http.MethodGet, "/api/v1/meta/products", nil, nil)
	products := decode[dto.Items[dto.ProductItem]](t, env)
	if len(products.Items) != 3 {
		t.Fatalf("products = %+v", products.Items)
	}
}

func TestCreateVisitValidation(t *testing.T) {
	resetVisits(t)
	planned := "2026-11-05T10:00:00+08:00"

	status, env, _ := call(t, http.MethodPost, "/api/v1/visits", map[string]any{
		"mr_id": fx.mr1, "hcp_id": fx.hcp2, "hospital_id": fx.hospital1, "product_id": fx.product1,
		"planned_at": planned,
	}, nil)
	expectStatus(t, status, http.StatusUnprocessableEntity, env)
	expectCode(t, env, "HCP_HOSPITAL_MISMATCH")

	status, env, _ = call(t, http.MethodPost, "/api/v1/visits", map[string]any{
		"mr_id": 999999, "hcp_id": fx.hcp1, "hospital_id": fx.hospital1, "product_id": fx.product1,
		"planned_at": planned,
	}, nil)
	expectStatus(t, status, http.StatusNotFound, env)
	expectCode(t, env, "NOT_FOUND")

	status, env, _ = call(t, http.MethodPost, "/api/v1/visits", map[string]any{
		"mr_id": fx.mr1, "hcp_id": fx.hcp1, "hospital_id": fx.hospital1,
	}, nil)
	expectStatus(t, status, http.StatusBadRequest, env)
	expectCode(t, env, "VALIDATION_ERROR")

	status, env, _ = call(t, http.MethodPost, "/api/v1/visits", map[string]any{
		"mr_id": fx.mr1, "hcp_id": fx.hcp1, "hospital_id": fx.hospital1, "product_id": fx.product1,
		"planned_at": "2026-11-05",
	}, nil)
	expectStatus(t, status, http.StatusBadRequest, env)

	created := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, planned)
	status, env, _ = call(t, http.MethodGet, fmt.Sprintf("/api/v1/visits/%d", created.ID), nil, nil)
	expectStatus(t, status, http.StatusOK, env)

	status, env, _ = call(t, http.MethodGet, "/api/v1/visits/999999", nil, nil)
	expectStatus(t, status, http.StatusNotFound, env)
	expectCode(t, env, "NOT_FOUND")
}

func TestVisitLifecycle(t *testing.T) {
	resetVisits(t)
	created := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T10:00:00+08:00")

	checkedIn := checkIn(t, created.ID, "2026-11-05T10:05:00+08:00", fx.lat, fx.lng)
	if checkedIn.Status != model.StatusCheckedIn || checkedIn.CheckIn == nil {
		t.Fatalf("checkin detail = %+v", checkedIn)
	}

	status, env, _ := call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-in", created.ID), map[string]any{
		"latitude": fx.lat, "longitude": fx.lng,
	}, nil)
	expectStatus(t, status, http.StatusConflict, env)
	expectCode(t, env, "DUPLICATE_ACTION")

	checkedOut := checkOut(t, created.ID, "2026-11-05T10:15:00+08:00", fx.lat, fx.lng)
	if checkedOut.Status != model.StatusCheckedOut {
		t.Fatalf("status = %s", checkedOut.Status)
	}
	if checkedOut.DurationSeconds == nil || *checkedOut.DurationSeconds != 600 {
		t.Fatalf("duration = %v", checkedOut.DurationSeconds)
	}
	if checkedOut.IsAbnormal == nil || *checkedOut.IsAbnormal {
		t.Fatalf("is_abnormal = %v", checkedOut.IsAbnormal)
	}
	if checkedOut.AbnormalReasons == nil || len(checkedOut.AbnormalReasons) != 0 {
		t.Fatalf("reasons = %+v", checkedOut.AbnormalReasons)
	}
	if checkedOut.CheckOut == nil || checkedOut.CheckOut.DistanceMeters != 0 {
		t.Fatalf("checkout = %+v", checkedOut.CheckOut)
	}

	status, env, _ = call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-out", created.ID), map[string]any{
		"latitude": fx.lat, "longitude": fx.lng,
	}, nil)
	expectStatus(t, status, http.StatusConflict, env)
	expectCode(t, env, "DUPLICATE_ACTION")

	reported := report(t, created.ID, map[string]any{
		"talking_points":        "介绍产品适应症与临床数据",
		"hcp_feedback":          "医生认可疗效",
		"materials_distributed": true,
		"follow_up":             "下月再次拜访",
		"materials": []map[string]any{
			{"material_name": "临床指南", "material_type": "guideline", "quantity": 2},
		},
	})
	if reported.Status != model.StatusReported {
		t.Fatalf("status = %s", reported.Status)
	}
	if reported.Report.ReportSubmittedAt == nil {
		t.Fatal("report_submitted_at 未设置")
	}
	if len(reported.Materials) != 1 || reported.Materials[0].Quantity != 2 {
		t.Fatalf("materials = %+v", reported.Materials)
	}
	if reported.IsAbnormal == nil || *reported.IsAbnormal {
		t.Fatal("合规结果不应被拜访记录覆盖")
	}

	firstSubmitted := *reported.Report.ReportSubmittedAt
	updated := report(t, created.ID, map[string]any{
		"talking_points":        "更新后的谈话要点",
		"materials_distributed": false,
	})
	if updated.Report.TalkingPoints == nil || *updated.Report.TalkingPoints != "更新后的谈话要点" {
		t.Fatalf("talking points = %v", updated.Report.TalkingPoints)
	}
	if updated.Report.ReportSubmittedAt == nil || *updated.Report.ReportSubmittedAt != firstSubmitted {
		t.Fatalf("report_submitted_at 被覆盖: %v", updated.Report.ReportSubmittedAt)
	}
	if len(updated.Materials) != 0 {
		t.Fatalf("materials 未清空: %+v", updated.Materials)
	}

	status, env, _ = call(t, http.MethodPut, fmt.Sprintf("/api/v1/visits/%d/report", created.ID), map[string]any{
		"talking_points":        "x",
		"materials_distributed": true,
	}, nil)
	expectStatus(t, status, http.StatusUnprocessableEntity, env)
	expectCode(t, env, "MATERIAL_INCONSISTENT")

	status, env, _ = call(t, http.MethodPut, fmt.Sprintf("/api/v1/visits/%d/report", created.ID), map[string]any{
		"talking_points":        "x",
		"materials_distributed": false,
		"materials":             []map[string]any{{"material_name": "A"}},
	}, nil)
	expectStatus(t, status, http.StatusUnprocessableEntity, env)
	expectCode(t, env, "MATERIAL_INCONSISTENT")
}

func TestAbnormalCompliance(t *testing.T) {
	resetVisits(t)
	shortVisit := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T10:00:00+08:00")
	checkIn(t, shortVisit.ID, "2026-11-05T10:00:00+08:00", fx.lat, fx.lng)
	detail := checkOut(t, shortVisit.ID, "2026-11-05T10:02:00+08:00", fx.lat, fx.lng)
	if detail.IsAbnormal == nil || !*detail.IsAbnormal {
		t.Fatal("停留时长不足应判定异常")
	}
	if len(detail.AbnormalReasons) != 1 || detail.AbnormalReasons[0].Code != "SHORT_DURATION" {
		t.Fatalf("reasons = %+v", detail.AbnormalReasons)
	}

	farVisit := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T11:00:00+08:00")
	checkIn(t, farVisit.ID, "2026-11-05T11:00:00+08:00", fx.lat, fx.lng)
	far := checkOut(t, farVisit.ID, "2026-11-05T11:15:00+08:00", fx.lat+0.01, fx.lng)
	if far.IsAbnormal == nil || !*far.IsAbnormal {
		t.Fatal("坐标偏离应判定异常")
	}
	if far.DeviationMeters == nil || *far.DeviationMeters < 500 {
		t.Fatalf("deviation = %v", far.DeviationMeters)
	}
	if len(far.AbnormalReasons) != 1 || far.AbnormalReasons[0].Code != "LOCATION_DEVIATION" {
		t.Fatalf("reasons = %+v", far.AbnormalReasons)
	}

	invalidState := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T12:00:00+08:00")
	status, env, _ := call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-out", invalidState.ID), map[string]any{
		"latitude": fx.lat, "longitude": fx.lng,
	}, nil)
	expectStatus(t, status, http.StatusConflict, env)
	expectCode(t, env, "INVALID_STATE")

	timeOrder := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T13:00:00+08:00")
	checkIn(t, timeOrder.ID, "2026-11-05T13:00:00+08:00", fx.lat, fx.lng)
	status, env, _ = call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-out", timeOrder.ID), map[string]any{
		"latitude": fx.lat, "longitude": fx.lng, "occurred_at": "2026-11-05T12:50:00+08:00",
	}, nil)
	expectStatus(t, status, http.StatusUnprocessableEntity, env)
	expectCode(t, env, "TIME_ORDER_INVALID")
}

func TestCancelAndInvalidState(t *testing.T) {
	resetVisits(t)
	created := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-06T10:00:00+08:00")
	status, env, _ := call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/cancel", created.ID), map[string]any{
		"reason": "医生临时有手术",
	}, nil)
	expectStatus(t, status, http.StatusOK, env)
	cancelled := decode[dto.VisitDetail](t, env)
	if cancelled.Status != model.StatusCancelled || cancelled.CancelReason == nil {
		t.Fatalf("cancel detail = %+v", cancelled)
	}

	status, env, _ = call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/cancel", created.ID), map[string]any{}, nil)
	expectStatus(t, status, http.StatusConflict, env)
	expectCode(t, env, "INVALID_STATE")

	status, env, _ = call(t, http.MethodPost, fmt.Sprintf("/api/v1/visits/%d/check-in", created.ID), map[string]any{
		"latitude": fx.lat, "longitude": fx.lng,
	}, nil)
	expectStatus(t, status, http.StatusConflict, env)
	expectCode(t, env, "INVALID_STATE")

	status, env, _ = call(t, http.MethodPut, fmt.Sprintf("/api/v1/visits/%d/report", created.ID), map[string]any{
		"talking_points": "x", "materials_distributed": false,
	}, nil)
	expectStatus(t, status, http.StatusConflict, env)
	expectCode(t, env, "INVALID_STATE")
}

func TestListVisits(t *testing.T) {
	resetVisits(t)
	planned := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T09:00:00+08:00")
	checked := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T10:00:00+08:00")
	checkIn(t, checked.ID, "2026-11-05T10:05:00+08:00", fx.lat, fx.lng)
	abnormal := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T11:00:00+08:00")
	checkIn(t, abnormal.ID, "2026-11-05T11:00:00+08:00", fx.lat, fx.lng)
	checkOut(t, abnormal.ID, "2026-11-05T11:01:00+08:00", fx.lat, fx.lng)

	status, env, _ := call(t, http.MethodGet, "/api/v1/visits?status=planned&page=1&page_size=10", nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	plannedList := decode[dto.VisitListResult](t, env)
	if plannedList.Total != 1 || len(plannedList.Items) != 1 || plannedList.Items[0].ID != planned.ID {
		t.Fatalf("planned list = %+v", plannedList)
	}
	if plannedList.Items[0].HospitalName == "" || plannedList.Items[0].MrName == "" {
		t.Fatalf("list join fields missing: %+v", plannedList.Items[0])
	}

	status, env, _ = call(t, http.MethodGet, "/api/v1/visits?abnormal=true", nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	abnormalList := decode[dto.VisitListResult](t, env)
	if abnormalList.Total != 1 || abnormalList.Items[0].ID != abnormal.ID {
		t.Fatalf("abnormal list = %+v", abnormalList)
	}
	if abnormalList.Items[0].IsAbnormal == nil || !*abnormalList.Items[0].IsAbnormal {
		t.Fatal("is_abnormal 应为 true")
	}

	status, env, _ = call(t, http.MethodGet, fmt.Sprintf("/api/v1/visits?mr_id=%d&date_from=2026-11-05&date_to=2026-11-05&page=1&page_size=2", fx.mr1), nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	paged := decode[dto.VisitListResult](t, env)
	if paged.Total != 3 || len(paged.Items) != 2 || paged.PageSize != 2 {
		t.Fatalf("paged list = %+v", paged)
	}

	status, env, _ = call(t, http.MethodGet, "/api/v1/visits?page=abc", nil, nil)
	expectStatus(t, status, http.StatusBadRequest, env)
	expectCode(t, env, "VALIDATION_ERROR")
}

func TestDashboardProducts(t *testing.T) {
	resetVisits(t)

	normal := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T08:00:00+08:00")
	checkIn(t, normal.ID, "2026-11-05T08:05:00+08:00", fx.lat, fx.lng)
	checkOut(t, normal.ID, "2026-11-05T08:20:00+08:00", fx.lat, fx.lng)
	report(t, normal.ID, map[string]any{"talking_points": "正常拜访", "materials_distributed": false})

	abnormal := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product1, "2026-11-05T09:00:00+08:00")
	checkIn(t, abnormal.ID, "2026-11-05T09:00:00+08:00", fx.lat, fx.lng)
	checkOut(t, abnormal.ID, "2026-11-05T09:02:00+08:00", fx.lat, fx.lng)

	otherProduct := createVisit(t, fx.mr1, fx.hcp1, fx.hospital1, fx.product2, "2026-11-05T10:00:00+08:00")
	checkIn(t, otherProduct.ID, "2026-11-05T10:00:00+08:00", fx.lat, fx.lng)
	checkOut(t, otherProduct.ID, "2026-11-05T10:15:00+08:00", fx.lat, fx.lng)

	otherMR := createVisit(t, fx.mr2, fx.hcp2, fx.hospital2, fx.product2, "2026-11-05T11:00:00+08:00")
	checkIn(t, otherMR.ID, "2026-11-05T11:00:00+08:00", fx.lat2, fx.lng2)
	checkOut(t, otherMR.ID, "2026-11-05T11:15:00+08:00", fx.lat2, fx.lng2)

	status, env, _ := call(t, http.MethodGet, "/api/v1/dashboard/products?month=2026-11", nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	board := decode[dto.ProductDashboard](t, env)
	if board.TotalVisits != 4 || board.NormalVisits != 3 || board.AbnormalVisits != 1 {
		t.Fatalf("dashboard = %+v", board)
	}
	if len(board.Items) != 2 {
		t.Fatalf("items = %+v", board.Items)
	}
	if board.AbnormalRate != 0.25 {
		t.Fatalf("abnormal_rate = %v", board.AbnormalRate)
	}

	status, env, _ = call(t, http.MethodGet, fmt.Sprintf("/api/v1/dashboard/products?month=2026-11&product_id=%d", fx.product1), nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	filtered := decode[dto.ProductDashboard](t, env)
	if filtered.TotalVisits != 2 || filtered.AbnormalVisits != 1 || len(filtered.Items) != 1 {
		t.Fatalf("filtered dashboard = %+v", filtered)
	}
	if filtered.ProductID == nil || *filtered.ProductID != fx.product1 {
		t.Fatalf("product_id = %v", filtered.ProductID)
	}

	status, env, _ = call(t, http.MethodGet, fmt.Sprintf("/api/v1/dashboard/products?month=2026-11&product_id=%d", fx.product3), nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	zero := decode[dto.ProductDashboard](t, env)
	if zero.TotalVisits != 0 || len(zero.Items) != 1 || zero.Items[0].ProductID != fx.product3 || zero.Items[0].TotalVisits != 0 {
		t.Fatalf("zero dashboard = %+v", zero)
	}

	status, env, _ = call(t, http.MethodGet, fmt.Sprintf("/api/v1/dashboard/products?month=2026-11&mr_id=%d", fx.mr1), nil, nil)
	expectStatus(t, status, http.StatusOK, env)
	byMR := decode[dto.ProductDashboard](t, env)
	if byMR.TotalVisits != 3 {
		t.Fatalf("mr dashboard = %+v", byMR)
	}

	status, env, _ = call(t, http.MethodGet, "/api/v1/dashboard/products?month=2026-13", nil, nil)
	expectStatus(t, status, http.StatusBadRequest, env)
	expectCode(t, env, "VALIDATION_ERROR")

	status, env, _ = call(t, http.MethodGet, "/api/v1/dashboard/products?month=2026-11&product_id=999999", nil, nil)
	expectStatus(t, status, http.StatusNotFound, env)
	expectCode(t, env, "NOT_FOUND")
}

func TestCORSAndRequestID(t *testing.T) {
	resp, err := httpClient.Do(mustRequest(t, http.MethodOptions, "/api/v1/visits", nil))
	if err != nil {
		t.Fatalf("OPTIONS 请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("CORS header = %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}

	_, env, _ := call(t, http.MethodGet, "/api/v1/meta/mrs", nil, map[string]string{"X-Request-Id": "req-e2e-fixed"})
	if env.RequestID != "req-e2e-fixed" {
		t.Fatalf("request_id = %q", env.RequestID)
	}
}

func mustRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
