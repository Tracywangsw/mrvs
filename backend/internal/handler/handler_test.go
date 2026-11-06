package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/model"
	"github.com/mrvs/backend/internal/router"
	"github.com/mrvs/backend/internal/service"
)

type envelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"request_id"`
}

type fakeService struct {
	createFn    func(context.Context, service.CreateVisitInput) (*dto.VisitDetail, error)
	listFn      func(context.Context, service.ListVisitsInput) (*dto.VisitListResult, error)
	getFn       func(context.Context, uint64) (*dto.VisitDetail, error)
	checkInFn   func(context.Context, uint64, service.CheckInput) (*dto.VisitDetail, error)
	checkOutFn  func(context.Context, uint64, service.CheckInput) (*dto.VisitDetail, error)
	reportFn    func(context.Context, uint64, service.ReportInput) (*dto.VisitDetail, error)
	cancelFn    func(context.Context, uint64, *string) (*dto.VisitDetail, error)
	dashboardFn func(context.Context, service.DashboardInput) (*dto.ProductDashboard, error)
	metaFn      func(context.Context, string, *uint64) (any, error)

	lastList     *service.ListVisitsInput
	lastReport   *service.ReportInput
	lastCheckIn  *service.CheckInput
	lastCancel   *string
	lastMetaKind string
}

func emptyDetail() *dto.VisitDetail {
	planned := "2026-11-05T04:00:00Z"
	return &dto.VisitDetail{
		ID: 1, VisitNo: "V20261105-000001", Status: model.StatusPlanned, PlannedAt: planned,
		Materials: []dto.Material{}, AbnormalReasons: []compliance.Violation{},
	}
}

func (f *fakeService) Create(ctx context.Context, in service.CreateVisitInput) (*dto.VisitDetail, error) {
	if f.createFn != nil {
		return f.createFn(ctx, in)
	}
	return emptyDetail(), nil
}

func (f *fakeService) List(ctx context.Context, in service.ListVisitsInput) (*dto.VisitListResult, error) {
	f.lastList = &in
	if f.listFn != nil {
		return f.listFn(ctx, in)
	}
	return &dto.VisitListResult{Page: in.Page, PageSize: in.PageSize, Items: []dto.VisitListItem{}}, nil
}

func (f *fakeService) Get(ctx context.Context, id uint64) (*dto.VisitDetail, error) {
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return emptyDetail(), nil
}

func (f *fakeService) CheckIn(ctx context.Context, id uint64, in service.CheckInput) (*dto.VisitDetail, error) {
	copied := in
	f.lastCheckIn = &copied
	if f.checkInFn != nil {
		return f.checkInFn(ctx, id, in)
	}
	return emptyDetail(), nil
}

func (f *fakeService) CheckOut(ctx context.Context, id uint64, in service.CheckInput) (*dto.VisitDetail, error) {
	if f.checkOutFn != nil {
		return f.checkOutFn(ctx, id, in)
	}
	return emptyDetail(), nil
}

func (f *fakeService) Report(ctx context.Context, id uint64, in service.ReportInput) (*dto.VisitDetail, error) {
	copied := in
	f.lastReport = &copied
	if f.reportFn != nil {
		return f.reportFn(ctx, id, in)
	}
	return emptyDetail(), nil
}

func (f *fakeService) Cancel(ctx context.Context, id uint64, reason *string) (*dto.VisitDetail, error) {
	f.lastCancel = reason
	if f.cancelFn != nil {
		return f.cancelFn(ctx, id, reason)
	}
	return emptyDetail(), nil
}

func (f *fakeService) DashboardProducts(ctx context.Context, in service.DashboardInput) (*dto.ProductDashboard, error) {
	if f.dashboardFn != nil {
		return f.dashboardFn(ctx, in)
	}
	return &dto.ProductDashboard{Month: in.Month, Items: []dto.ProductDashboardItem{}}, nil
}

func (f *fakeService) Meta(ctx context.Context, kind string, hospitalID *uint64) (any, error) {
	f.lastMetaKind = kind
	if f.metaFn != nil {
		return f.metaFn(ctx, kind, hospitalID)
	}
	return dto.Items[dto.MRItem]{Items: []dto.MRItem{}}, nil
}

func newEngine(t *testing.T, svc service.Service) *route.Engine {
	t.Helper()
	hz := router.New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return hz.Engine
}

func perform(engine *route.Engine, method, path, body string, headers ...ut.Header) *ut.ResponseRecorder {
	var b *ut.Body
	if body != "" {
		b = &ut.Body{Body: strings.NewReader(body), Len: len(body)}
	}
	base := []ut.Header{{Key: "Content-Type", Value: "application/json"}}
	base = append(base, headers...)
	return ut.PerformRequest(engine, method, path, b, base...)
}

func decodeEnvelope(t *testing.T, w *ut.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response failed: %v, body=%s", err, w.Body.String())
	}
	return env
}

func TestHealth(t *testing.T) {
	engine := newEngine(t, &fakeService{})
	w := perform(engine, "GET", "/healthz", "")
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v", body)
	}
}

func TestCreateVisitHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &fakeService{}
		engine := newEngine(t, svc)
		w := perform(engine, "POST", "/api/v1/visits",
			`{"mr_id":1,"hcp_id":10,"hospital_id":100,"product_id":500,"planned_at":"2026-11-06T10:00:00+08:00"}`,
			ut.Header{Key: "X-Request-Id", Value: "req-test-1"})
		if w.Code != 201 {
			t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
		}
		env := decodeEnvelope(t, w)
		if env.Code != apperr.CodeOK || env.RequestID != "req-test-1" {
			t.Fatalf("envelope = %+v", env)
		}
		if w.Header().Get("X-Request-Id") != "req-test-1" {
			t.Fatalf("response header request id = %q", w.Header().Get("X-Request-Id"))
		}
		var detail dto.VisitDetail
		if err := json.Unmarshal(env.Data, &detail); err != nil {
			t.Fatalf("decode data failed: %v", err)
		}
		if detail.ID != 1 {
			t.Fatalf("detail = %+v", detail)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		w := perform(newEngine(t, &fakeService{}), "POST", "/api/v1/visits", `{`)
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
		env := decodeEnvelope(t, w)
		if env.Code != apperr.CodeValidation {
			t.Fatalf("code = %s", env.Code)
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		w := perform(newEngine(t, &fakeService{}), "POST", "/api/v1/visits", `{"mr_id":1}`)
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
		env := decodeEnvelope(t, w)
		var data struct {
			Fields map[string]string `json:"fields"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			t.Fatalf("decode fields failed: %v", err)
		}
		if data.Fields["hcp_id"] == "" || data.Fields["planned_at"] == "" {
			t.Fatalf("fields = %+v", data.Fields)
		}
	})

	t.Run("invalid planned_at", func(t *testing.T) {
		w := perform(newEngine(t, &fakeService{}), "POST", "/api/v1/visits",
			`{"mr_id":1,"hcp_id":10,"hospital_id":100,"product_id":500,"planned_at":"2026-11-06"}`)
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		svc := &fakeService{createFn: func(context.Context, service.CreateVisitInput) (*dto.VisitDetail, error) {
			return nil, apperr.NotFound("医生不存在")
		}}
		w := perform(newEngine(t, svc), "POST", "/api/v1/visits",
			`{"mr_id":1,"hcp_id":10,"hospital_id":100,"product_id":500,"planned_at":"2026-11-06T10:00:00Z"}`)
		if w.Code != 404 {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestListHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	w := perform(engine, "GET",
		"/api/v1/visits?mr_id=1&status=planned&date_from=2026-11-01&date_to=2026-11-05&abnormal=true&page=2&page_size=5", "")
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if svc.lastList == nil || svc.lastList.MrID == nil || *svc.lastList.MrID != 1 {
		t.Fatalf("mr_id not forwarded: %+v", svc.lastList)
	}
	if svc.lastList.Status == nil || *svc.lastList.Status != "planned" {
		t.Fatalf("status not forwarded: %+v", svc.lastList)
	}
	if svc.lastList.Abnormal == nil || !*svc.lastList.Abnormal {
		t.Fatalf("abnormal not forwarded: %+v", svc.lastList)
	}
	if svc.lastList.Page != 2 || svc.lastList.PageSize != 5 {
		t.Fatalf("pagination = %d/%d", svc.lastList.Page, svc.lastList.PageSize)
	}
	if svc.lastList.DateFrom == nil || svc.lastList.DateTo == nil {
		t.Fatal("date range not forwarded")
	}

	t.Run("invalid page", func(t *testing.T) {
		w := perform(engine, "GET", "/api/v1/visits?page=abc", "")
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("invalid abnormal", func(t *testing.T) {
		w := perform(engine, "GET", "/api/v1/visits?abnormal=yes", "")
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("invalid date", func(t *testing.T) {
		w := perform(engine, "GET", "/api/v1/visits?date_from=11/01/2026", "")
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		s := &fakeService{listFn: func(context.Context, service.ListVisitsInput) (*dto.VisitListResult, error) {
			return nil, apperr.Validation("bad")
		}}
		w := perform(newEngine(t, s), "GET", "/api/v1/visits", "")
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestGetHandler(t *testing.T) {
	svc := &fakeService{getFn: func(_ context.Context, id uint64) (*dto.VisitDetail, error) {
		if id == 404 {
			return nil, apperr.NotFound("拜访不存在")
		}
		return emptyDetail(), nil
	}}
	engine := newEngine(t, svc)

	if w := perform(engine, "GET", "/api/v1/visits/1", ""); w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w := perform(engine, "GET", "/api/v1/visits/404", ""); w.Code != 404 {
		t.Fatalf("status = %d", w.Code)
	}
	if w := perform(engine, "GET", "/api/v1/visits/abc", ""); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestCheckInHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	w := perform(engine, "POST", "/api/v1/visits/1/check-in",
		`{"latitude":39.912345,"longitude":116.410123,"occurred_at":"2026-11-05T12:00:00+08:00"}`)
	if w.Code != 200 {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if svc.lastCheckIn == nil || svc.lastCheckIn.OccurredAt == nil {
		t.Fatal("occurred_at not parsed")
	}

	t.Run("missing coordinates", func(t *testing.T) {
		w := perform(engine, "POST", "/api/v1/visits/1/check-in", `{}`)
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("invalid occurred_at", func(t *testing.T) {
		w := perform(engine, "POST", "/api/v1/visits/1/check-in", `{"latitude":39.9,"longitude":116.4,"occurred_at":"now"}`)
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("duplicate action", func(t *testing.T) {
		s := &fakeService{checkInFn: func(context.Context, uint64, service.CheckInput) (*dto.VisitDetail, error) {
			return nil, apperr.DuplicateAction("拜访已签到")
		}}
		w := perform(newEngine(t, s), "POST", "/api/v1/visits/1/check-in", `{"latitude":39.9,"longitude":116.4}`)
		if w.Code != 409 {
			t.Fatalf("status = %d", w.Code)
		}
		env := decodeEnvelope(t, w)
		if env.Code != apperr.CodeDuplicateAction {
			t.Fatalf("code = %s", env.Code)
		}
	})
}

func TestCheckOutHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	w := perform(engine, "POST", "/api/v1/visits/1/check-out", `{"latitude":39.912345,"longitude":116.410123}`)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w := perform(engine, "POST", "/api/v1/visits/abc/check-out", `{"latitude":39.9,"longitude":116.4}`); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
	if w := perform(engine, "POST", "/api/v1/visits/1/check-out", `{"latitude":39.9}`); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestReportHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	w := perform(engine, "PUT", "/api/v1/visits/1/report",
		`{"talking_points":"介绍适应症","hcp_feedback":"认可","materials_distributed":true,"follow_up":"下月跟进","materials":[{"material_name":"指南","material_type":"guideline","quantity":2}]}`)
	if w.Code != 200 {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if svc.lastReport == nil || svc.lastReport.TalkingPoints != "介绍适应症" || len(svc.lastReport.Materials) != 1 {
		t.Fatalf("report not forwarded: %+v", svc.lastReport)
	}
	if svc.lastReport.MaterialsDistributed != true {
		t.Fatal("materials_distributed not forwarded")
	}

	t.Run("missing required fields", func(t *testing.T) {
		w := perform(engine, "PUT", "/api/v1/visits/1/report", `{"hcp_feedback":"x"}`)
		if w.Code != 400 {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("material inconsistent", func(t *testing.T) {
		s := &fakeService{reportFn: func(context.Context, uint64, service.ReportInput) (*dto.VisitDetail, error) {
			return nil, apperr.MaterialInconsistent("资料与标记不一致")
		}}
		w := perform(newEngine(t, s), "PUT", "/api/v1/visits/1/report",
			`{"talking_points":"x","materials_distributed":true}`)
		if w.Code != 422 {
			t.Fatalf("status = %d", w.Code)
		}
	})
}

func TestCancelHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	w := perform(engine, "POST", "/api/v1/visits/1/cancel", "")
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if svc.lastCancel != nil {
		t.Fatalf("reason = %v", svc.lastCancel)
	}
	w = perform(engine, "POST", "/api/v1/visits/1/cancel", `{"reason":"临时取消"}`)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if svc.lastCancel == nil || *svc.lastCancel != "临时取消" {
		t.Fatalf("reason = %v", svc.lastCancel)
	}
	if w := perform(engine, "POST", "/api/v1/visits/1/cancel", `{`); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDashboardHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	w := perform(engine, "GET", "/api/v1/dashboard/products?month=2026-11&mr_id=1&product_id=500", "")
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if w := perform(engine, "GET", "/api/v1/dashboard/products?mr_id=1", ""); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
	if w := perform(engine, "GET", "/api/v1/dashboard/products?month=2026-11&mr_id=abc", ""); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestMetaHandler(t *testing.T) {
	svc := &fakeService{}
	engine := newEngine(t, svc)
	for _, path := range []string{
		"/api/v1/meta/mrs",
		"/api/v1/meta/hospitals",
		"/api/v1/meta/hcps?hospital_id=100",
		"/api/v1/meta/products",
	} {
		w := perform(engine, "GET", path, "")
		if w.Code != 200 {
			t.Fatalf("%s status = %d", path, w.Code)
		}
	}
	if w := perform(engine, "GET", "/api/v1/meta/hcps?hospital_id=abc", ""); w.Code != 400 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	engine := newEngine(t, &fakeService{})
	w := perform(engine, "OPTIONS", "/api/v1/visits", "")
	if w.Code != 204 {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("cors header missing: %v", w.Header())
	}
}

func TestRequestIDGenerated(t *testing.T) {
	engine := newEngine(t, &fakeService{})
	w := perform(engine, "GET", "/api/v1/visits", "")
	env := decodeEnvelope(t, w)
	if env.RequestID == "" || !strings.HasPrefix(env.RequestID, "req-") {
		t.Fatalf("request_id = %q", env.RequestID)
	}
}
