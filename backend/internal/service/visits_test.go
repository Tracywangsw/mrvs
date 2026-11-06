package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/model"
	"github.com/mrvs/backend/internal/repository"
)

func floatPtr(v float64) *float64 { return &v }

func boolPtr(v bool) *bool { return &v }

func uint64Ptr(v uint64) *uint64 { return &v }

func strPtr(v string) *string { return &v }

func newTestService(f *fakeRepo, allowOverride bool) *VisitService {
	s := New(f, compliance.Rules{MinDurationSeconds: 300, MaxDistanceMeters: 500}, allowOverride)
	s.now = func() time.Time { return f.now }
	return s
}

func (f *fakeRepo) addVisit(status string) *model.Visit {
	id := f.nextID
	f.nextID++
	v := &model.Visit{
		ID: id, VisitNo: "V20261105-000001", MrID: 1, HcpID: 10, HospitalID: 100, ProductID: 500,
		PlannedAt: f.now.Add(-time.Hour), Status: status, CreatedAt: f.now, UpdatedAt: f.now,
	}
	if status == model.StatusCheckedIn || status == model.StatusCheckedOut || status == model.StatusReported {
		at := f.now.Add(-10 * time.Minute)
		v.CheckInAt = &at
		v.CheckInLat = floatPtr(f.hospitals[100].Latitude)
		v.CheckInLng = floatPtr(f.hospitals[100].Longitude)
		v.CheckInDistanceMeters = floatPtr(0)
	}
	if status == model.StatusCheckedOut || status == model.StatusReported {
		at := f.now
		v.CheckOutAt = &at
		v.CheckOutLat = floatPtr(f.hospitals[100].Latitude)
		v.CheckOutLng = floatPtr(f.hospitals[100].Longitude)
		v.CheckOutDistanceMeters = floatPtr(0)
		duration := uint(600)
		v.DurationSeconds = &duration
		v.DeviationMeters = floatPtr(0)
		v.IsAbnormal = boolPtr(false)
		reasons := "[]"
		v.AbnormalReasons = &reasons
	}
	f.visits[id] = v
	return v
}

func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected apperr.Error, got %v", err)
	}
	if ae.Code != code {
		t.Fatalf("error code = %s, want %s (message: %s)", ae.Code, code, ae.Message)
	}
}

func TestCreateVisit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		detail, err := s.Create(context.Background(), CreateVisitInput{
			MRID: 1, HCPID: 10, HospitalID: 100, ProductID: 500, PlannedAt: f.now.Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Status != model.StatusPlanned {
			t.Fatalf("status = %s", detail.Status)
		}
		if !strings.HasPrefix(detail.VisitNo, "V20261105-") {
			t.Fatalf("visit_no = %s", detail.VisitNo)
		}
		if detail.MR.ID != 1 || detail.HCP.ID != 10 || detail.Hospital.ID != 100 || detail.Product.ID != 500 {
			t.Fatalf("unexpected detail refs: %+v", detail)
		}
		if detail.IsAbnormal != nil {
			t.Fatal("is_abnormal should be nil before checkout")
		}
		if detail.Materials == nil || len(detail.Materials) != 0 {
			t.Fatal("materials should be an empty array")
		}
	})

	t.Run("missing ids", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		_, err := s.Create(context.Background(), CreateVisitInput{})
		requireCode(t, err, apperr.CodeValidation)
	})

	t.Run("inactive mr", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		_, err := s.Create(context.Background(), CreateVisitInput{MRID: 2, HCPID: 10, HospitalID: 100, ProductID: 500, PlannedAt: f.now})
		requireCode(t, err, apperr.CodeNotFound)
	})

	t.Run("hospital mismatch", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		_, err := s.Create(context.Background(), CreateVisitInput{MRID: 1, HCPID: 11, HospitalID: 100, ProductID: 500, PlannedAt: f.now})
		requireCode(t, err, apperr.CodeHCPHospitalMismatch)
	})

	t.Run("retry on duplicate visit_no", func(t *testing.T) {
		f := newFakeRepo()
		f.createErrors = []error{duplicateKey()}
		s := newTestService(f, false)
		detail, err := s.Create(context.Background(), CreateVisitInput{MRID: 1, HCPID: 10, HospitalID: 100, ProductID: 500, PlannedAt: f.now})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.ID == 0 {
			t.Fatal("visit should be created")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		f := newFakeRepo()
		f.createErrors = []error{errors.New("db down")}
		s := newTestService(f, false)
		_, err := s.Create(context.Background(), CreateVisitInput{MRID: 1, HCPID: 10, HospitalID: 100, ProductID: 500, PlannedAt: f.now})
		requireCode(t, err, apperr.CodeInternal)
	})
}

func TestGetVisit(t *testing.T) {
	f := newFakeRepo()
	s := newTestService(f, false)
	if _, err := s.Get(context.Background(), 999); err != nil {
		requireCode(t, err, apperr.CodeNotFound)
	}
	f.getErr = errors.New("db down")
	if _, err := s.Get(context.Background(), 1); err != nil {
		requireCode(t, err, apperr.CodeInternal)
	}
}

func TestListVisits(t *testing.T) {
	f := newFakeRepo()
	s := newTestService(f, false)
	first := f.addVisit(model.StatusPlanned)
	second := f.addVisit(model.StatusCheckedOut)
	second.PlannedAt = f.now.Add(-30 * time.Minute)

	t.Run("defaults", func(t *testing.T) {
		result, err := s.List(context.Background(), ListVisitsInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Page != 1 || result.PageSize != 20 || result.Total != 2 {
			t.Fatalf("unexpected pagination: %+v", result)
		}
		if len(result.Items) != 2 {
			t.Fatalf("items = %d", len(result.Items))
		}
		if result.Items[0].ID != second.ID {
			t.Fatalf("expected newest first, got %+v", result.Items[0])
		}
		if result.Items[0].HospitalName != "北京协和医院" || result.Items[0].MrName != "张伟" {
			t.Fatalf("join names missing: %+v", result.Items[0])
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		status := "unknown"
		_, err := s.List(context.Background(), ListVisitsInput{Status: &status})
		requireCode(t, err, apperr.CodeValidation)
	})

	t.Run("page size too large", func(t *testing.T) {
		_, err := s.List(context.Background(), ListVisitsInput{PageSize: 101})
		requireCode(t, err, apperr.CodeValidation)
	})

	t.Run("abnormal filter", func(t *testing.T) {
		_, err := s.List(context.Background(), ListVisitsInput{Abnormal: boolPtr(true)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.lastFilter.Abnormal == nil || !*f.lastFilter.Abnormal {
			t.Fatal("abnormal filter not forwarded")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		f.listErr = errors.New("db down")
		defer func() { f.listErr = nil }()
		_, err := s.List(context.Background(), ListVisitsInput{})
		requireCode(t, err, apperr.CodeInternal)
	})

	if first.Status != model.StatusPlanned {
		t.Fatal("sanity check failed")
	}
}

func TestCheckIn(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		detail, err := s.CheckIn(context.Background(), v.ID, CheckInput{
			Latitude: f.hospitals[100].Latitude, Longitude: f.hospitals[100].Longitude,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Status != model.StatusCheckedIn || detail.CheckIn == nil {
			t.Fatalf("unexpected detail: %+v", detail)
		}
		if detail.CheckIn.DistanceMeters != 0 {
			t.Fatalf("distance = %v", detail.CheckIn.DistanceMeters)
		}
		if detail.IsAbnormal != nil {
			t.Fatal("is_abnormal should stay nil before checkout")
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedIn)
		_, err := s.CheckIn(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 116.4})
		requireCode(t, err, apperr.CodeDuplicateAction)
	})

	t.Run("invalid state", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCancelled)
		_, err := s.CheckIn(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 116.4})
		requireCode(t, err, apperr.CodeInvalidState)
	})

	t.Run("not found", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		_, err := s.CheckIn(context.Background(), 999, CheckInput{Latitude: 39.9, Longitude: 116.4})
		requireCode(t, err, apperr.CodeNotFound)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		_, err := s.CheckIn(context.Background(), v.ID, CheckInput{Latitude: 100, Longitude: 116.4})
		requireCode(t, err, apperr.CodeBusinessRule)
	})

	t.Run("time override disabled", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		at := f.now.Add(-time.Minute)
		_, err := s.CheckIn(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 116.4, OccurredAt: &at})
		requireCode(t, err, apperr.CodeTimeOverrideDisabled)
	})

	t.Run("time override enabled", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, true)
		v := f.addVisit(model.StatusPlanned)
		at := f.now.Add(-5 * time.Minute)
		detail, err := s.CheckIn(context.Background(), v.ID, CheckInput{
			Latitude: f.hospitals[100].Latitude, Longitude: f.hospitals[100].Longitude, OccurredAt: &at,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.CheckIn.At != at.Format(time.RFC3339) {
			t.Fatalf("checkin at = %s", detail.CheckIn.At)
		}
	})
}

func TestCheckOut(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedIn)
		detail, err := s.CheckOut(context.Background(), v.ID, CheckInput{
			Latitude: f.hospitals[100].Latitude, Longitude: f.hospitals[100].Longitude,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Status != model.StatusCheckedOut {
			t.Fatalf("status = %s", detail.Status)
		}
		if detail.DurationSeconds == nil || *detail.DurationSeconds != 600 {
			t.Fatalf("duration = %v", detail.DurationSeconds)
		}
		if detail.IsAbnormal == nil || *detail.IsAbnormal {
			t.Fatalf("is_abnormal = %v", detail.IsAbnormal)
		}
		if detail.AbnormalReasons == nil || len(detail.AbnormalReasons) != 0 {
			t.Fatalf("reasons = %+v", detail.AbnormalReasons)
		}
	})

	t.Run("short duration", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedIn)
		short := f.now.Add(-120 * time.Second)
		v.CheckInAt = &short
		detail, err := s.CheckOut(context.Background(), v.ID, CheckInput{
			Latitude: f.hospitals[100].Latitude, Longitude: f.hospitals[100].Longitude,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.IsAbnormal == nil || !*detail.IsAbnormal {
			t.Fatal("visit should be abnormal")
		}
		if len(detail.AbnormalReasons) != 1 || detail.AbnormalReasons[0].Code != "SHORT_DURATION" {
			t.Fatalf("reasons = %+v", detail.AbnormalReasons)
		}
	})

	t.Run("location deviation", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedIn)
		detail, err := s.CheckOut(context.Background(), v.ID, CheckInput{
			Latitude: f.hospitals[100].Latitude + 0.01, Longitude: f.hospitals[100].Longitude,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.DeviationMeters == nil || *detail.DeviationMeters < 500 {
			t.Fatalf("deviation = %v", detail.DeviationMeters)
		}
		found := false
		for _, r := range detail.AbnormalReasons {
			if r.Code == "LOCATION_DEVIATION" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing LOCATION_DEVIATION: %+v", detail.AbnormalReasons)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedOut)
		_, err := s.CheckOut(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 116.4})
		requireCode(t, err, apperr.CodeDuplicateAction)
	})

	t.Run("invalid state", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		_, err := s.CheckOut(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 116.4})
		requireCode(t, err, apperr.CodeInvalidState)
	})

	t.Run("time order invalid", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, true)
		v := f.addVisit(model.StatusCheckedIn)
		at := v.CheckInAt.Add(-time.Minute)
		_, err := s.CheckOut(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 116.4, OccurredAt: &at})
		requireCode(t, err, apperr.CodeTimeOrderInvalid)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedIn)
		_, err := s.CheckOut(context.Background(), v.ID, CheckInput{Latitude: 39.9, Longitude: 200})
		requireCode(t, err, apperr.CodeBusinessRule)
	})
}

func TestReport(t *testing.T) {
	feedback := "反馈良好"
	t.Run("first submission", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedOut)
		detail, err := s.Report(context.Background(), v.ID, ReportInput{
			TalkingPoints:        "介绍产品适应症",
			HCPFeedback:          &feedback,
			MaterialsDistributed: true,
			Materials: []MaterialInput{
				{Name: "临床指南", Type: strPtr("guideline"), Quantity: 2},
				{Name: "产品手册", Quantity: 0},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Status != model.StatusReported {
			t.Fatalf("status = %s", detail.Status)
		}
		if detail.Report.ReportSubmittedAt == nil {
			t.Fatal("report_submitted_at should be set")
		}
		if len(detail.Materials) != 2 || detail.Materials[1].Quantity != 1 {
			t.Fatalf("materials = %+v", detail.Materials)
		}
		if detail.Report.MaterialsDistributed == nil || !*detail.Report.MaterialsDistributed {
			t.Fatal("materials_distributed should be true")
		}
	})

	t.Run("update keeps submitted at and replaces materials", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedOut)
		first, err := s.Report(context.Background(), v.ID, ReportInput{
			TalkingPoints: "第一次", MaterialsDistributed: true,
			Materials: []MaterialInput{{Name: "A", Quantity: 1}, {Name: "B", Quantity: 1}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		second, err := s.Report(context.Background(), v.ID, ReportInput{
			TalkingPoints: "第二次", MaterialsDistributed: true,
			Materials: []MaterialInput{{Name: "C", Quantity: 3}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if second.Report.TalkingPoints == nil || *second.Report.TalkingPoints != "第二次" {
			t.Fatalf("talking points = %v", second.Report.TalkingPoints)
		}
		if second.Report.ReportSubmittedAt == nil || first.Report.ReportSubmittedAt == nil ||
			*second.Report.ReportSubmittedAt != *first.Report.ReportSubmittedAt {
			t.Fatalf("submitted_at changed: %v -> %v", first.Report.ReportSubmittedAt, second.Report.ReportSubmittedAt)
		}
		if len(second.Materials) != 1 || second.Materials[0].MaterialName != "C" {
			t.Fatalf("materials = %+v", second.Materials)
		}
	})

	t.Run("distributed without materials", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedOut)
		_, err := s.Report(context.Background(), v.ID, ReportInput{TalkingPoints: "x", MaterialsDistributed: true})
		requireCode(t, err, apperr.CodeMaterialInconsistent)
	})

	t.Run("materials without distributed flag", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedOut)
		_, err := s.Report(context.Background(), v.ID, ReportInput{
			TalkingPoints: "x", MaterialsDistributed: false,
			Materials: []MaterialInput{{Name: "A", Quantity: 1}},
		})
		requireCode(t, err, apperr.CodeMaterialInconsistent)
	})

	t.Run("wrong state", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		_, err := s.Report(context.Background(), v.ID, ReportInput{TalkingPoints: "x", MaterialsDistributed: false})
		requireCode(t, err, apperr.CodeInvalidState)
	})

	t.Run("validation", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedOut)
		cases := []ReportInput{
			{TalkingPoints: "  ", MaterialsDistributed: false},
			{TalkingPoints: strings.Repeat("a", 5001), MaterialsDistributed: false},
			{TalkingPoints: "x", MaterialsDistributed: true, Materials: []MaterialInput{{Name: "A", Type: strPtr("bad"), Quantity: 1}}},
			{TalkingPoints: "x", MaterialsDistributed: true, Materials: []MaterialInput{{Name: "A", Quantity: 1001}}},
		}
		for _, in := range cases {
			if _, err := s.Report(context.Background(), v.ID, in); err != nil {
				requireCode(t, err, apperr.CodeValidation)
			}
		}
	})
}

func TestCancel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		reason := "患者临时有事"
		detail, err := s.Cancel(context.Background(), v.ID, &reason)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Status != model.StatusCancelled {
			t.Fatalf("status = %s", detail.Status)
		}
		if detail.CancelReason == nil || *detail.CancelReason != reason {
			t.Fatalf("cancel reason = %v", detail.CancelReason)
		}
		if detail.CancelledAt == nil {
			t.Fatal("cancelled_at should be set")
		}
	})

	t.Run("invalid state", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusCheckedIn)
		if _, err := s.Cancel(context.Background(), v.ID, nil); err != nil {
			requireCode(t, err, apperr.CodeInvalidState)
		}
	})

	t.Run("reason too long", func(t *testing.T) {
		f := newFakeRepo()
		s := newTestService(f, false)
		v := f.addVisit(model.StatusPlanned)
		reason := strings.Repeat("原", 256)
		if _, err := s.Cancel(context.Background(), v.ID, &reason); err != nil {
			requireCode(t, err, apperr.CodeValidation)
		}
	})
}

func TestDashboardProducts(t *testing.T) {
	f := newFakeRepo()
	s := newTestService(f, false)

	t.Run("invalid month", func(t *testing.T) {
		_, err := s.DashboardProducts(context.Background(), DashboardInput{Month: "2026-13"})
		requireCode(t, err, apperr.CodeValidation)
	})

	t.Run("aggregate", func(t *testing.T) {
		f.stats = []repository.ProductStatRow{
			{ProductID: 500, ProductCode: "P001", ProductName: "立普妥", TotalVisits: 3, NormalVisits: 2, AbnormalVisits: 1},
			{ProductID: 501, ProductCode: "P002", ProductName: "拜唐苹", TotalVisits: 1, NormalVisits: 1},
		}
		result, err := s.DashboardProducts(context.Background(), DashboardInput{Month: "2026-11", MrID: uint64Ptr(1)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.TotalVisits != 4 || result.NormalVisits != 3 || result.AbnormalVisits != 1 {
			t.Fatalf("unexpected totals: %+v", result)
		}
		if math.Abs(result.AbnormalRate-0.25) > 1e-9 {
			t.Fatalf("rate = %v", result.AbnormalRate)
		}
		if len(result.Items) != 2 || math.Abs(result.Items[0].AbnormalRate-0.3333) > 1e-9 {
			t.Fatalf("items = %+v", result.Items)
		}
		if f.lastStat.MrID == nil || *f.lastStat.MrID != 1 {
			t.Fatal("mr filter not forwarded")
		}
	})

	t.Run("product filter with zero rows", func(t *testing.T) {
		f.stats = nil
		result, err := s.DashboardProducts(context.Background(), DashboardInput{Month: "2026-11", ProductID: uint64Ptr(501)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("items = %+v", result.Items)
		}
		item := result.Items[0]
		if item.ProductID != 501 || item.ProductName != "拜唐苹" || item.TotalVisits != 0 || item.AbnormalRate != 0 {
			t.Fatalf("zero item = %+v", item)
		}
		if result.TotalVisits != 0 || result.AbnormalRate != 0 {
			t.Fatalf("unexpected totals: %+v", result)
		}
	})

	t.Run("product not found", func(t *testing.T) {
		f.stats = nil
		_, err := s.DashboardProducts(context.Background(), DashboardInput{Month: "2026-11", ProductID: uint64Ptr(999)})
		requireCode(t, err, apperr.CodeNotFound)
	})

	t.Run("repo error", func(t *testing.T) {
		f.statsErr = errors.New("db down")
		defer func() { f.statsErr = nil }()
		_, err := s.DashboardProducts(context.Background(), DashboardInput{Month: "2026-11"})
		requireCode(t, err, apperr.CodeInternal)
	})
}

func TestMeta(t *testing.T) {
	f := newFakeRepo()
	s := newTestService(f, false)

	t.Run("mrs", func(t *testing.T) {
		data, err := s.Meta(context.Background(), MetaMRs, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		items := data.(dto.Items[dto.MRItem])
		if len(items.Items) != 1 || items.Items[0].Name != "张伟" {
			t.Fatalf("items = %+v", items.Items)
		}
	})

	t.Run("hcps filtered", func(t *testing.T) {
		data, err := s.Meta(context.Background(), MetaHCPs, uint64Ptr(100))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		items := data.(dto.Items[dto.HCPItem])
		if len(items.Items) != 1 || items.Items[0].ID != 10 {
			t.Fatalf("items = %+v", items.Items)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := s.Meta(context.Background(), "unknown", nil)
		requireCode(t, err, apperr.CodeValidation)
	})

	t.Run("repo error", func(t *testing.T) {
		f.listErr = errors.New("db down")
		defer func() { f.listErr = nil }()
		_, err := s.Meta(context.Background(), MetaProducts, nil)
		requireCode(t, err, apperr.CodeInternal)
	})
}
