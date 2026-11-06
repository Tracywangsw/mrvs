package service

import (
	"context"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/mrvs/backend/internal/model"
	"github.com/mrvs/backend/internal/repository"
)

type fakeRepo struct {
	mrs       map[uint64]*model.MR
	hcps      map[uint64]*model.HCP
	hospitals map[uint64]*model.Hospital
	products  map[uint64]*model.Product

	visits    map[uint64]*model.Visit
	materials map[uint64][]model.VisitMaterial

	stats []repository.ProductStatRow

	nextID       uint64
	now          time.Time
	createErrors []error
	getErr       error
	listErr      error
	statsErr     error
	lastFilter   repository.VisitListFilter
	lastStat     repository.ProductStatFilter
	calls        []string
}

func newFakeRepo() *fakeRepo {
	f := &fakeRepo{
		mrs:       map[uint64]*model.MR{},
		hcps:      map[uint64]*model.HCP{},
		hospitals: map[uint64]*model.Hospital{},
		products:  map[uint64]*model.Product{},
		visits:    map[uint64]*model.Visit{},
		materials: map[uint64][]model.VisitMaterial{},
		nextID:    1000,
		now:       time.Date(2026, 11, 5, 4, 0, 0, 0, time.UTC),
	}
	region := "华东大区"
	f.mrs[1] = &model.MR{ID: 1, EmployeeNo: "M001", Name: "张伟", Region: &region, IsActive: true}
	f.mrs[2] = &model.MR{ID: 2, EmployeeNo: "M002", Name: "李娜", IsActive: false}

	title := "主任医师"
	f.hcps[10] = &model.HCP{ID: 10, HospitalID: 100, Name: "王建国", Department: "心内科", Title: &title, IsActive: true}
	f.hcps[11] = &model.HCP{ID: 11, HospitalID: 200, Name: "刘洋", Department: "内分泌科", IsActive: true}
	f.hcps[12] = &model.HCP{ID: 12, HospitalID: 100, Name: "陈晓", Department: "呼吸科", IsActive: false}

	f.hospitals[100] = &model.Hospital{ID: 100, Name: "北京协和医院", Latitude: 39.912345, Longitude: 116.410123, IsActive: true}
	f.hospitals[200] = &model.Hospital{ID: 200, Name: "瑞金医院", Latitude: 31.211000, Longitude: 121.466000, IsActive: true}

	category := "心血管"
	f.products[500] = &model.Product{ID: 500, Code: "P001", Name: "立普妥", Category: &category, IsActive: true}
	f.products[501] = &model.Product{ID: 501, Code: "P002", Name: "拜唐苹", IsActive: false}
	return f
}

func (f *fakeRepo) Transaction(ctx context.Context, fn func(repository.Repository) error) error {
	f.calls = append(f.calls, "transaction")
	return fn(f)
}

func (f *fakeRepo) FindMR(_ context.Context, id uint64) (*model.MR, error) {
	return f.mrs[id], nil
}

func (f *fakeRepo) FindHCP(_ context.Context, id uint64) (*model.HCP, error) {
	return f.hcps[id], nil
}

func (f *fakeRepo) FindHospital(_ context.Context, id uint64) (*model.Hospital, error) {
	return f.hospitals[id], nil
}

func (f *fakeRepo) FindProduct(_ context.Context, id uint64) (*model.Product, error) {
	return f.products[id], nil
}

func (f *fakeRepo) ListMRs(context.Context) ([]model.MR, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	rows := make([]model.MR, 0, len(f.mrs))
	for id := uint64(1); id <= 10; id++ {
		if row, ok := f.mrs[id]; ok && row.IsActive {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}

func (f *fakeRepo) ListHospitals(context.Context) ([]model.Hospital, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	rows := make([]model.Hospital, 0, len(f.hospitals))
	for id := uint64(1); id <= 1000; id++ {
		if row, ok := f.hospitals[id]; ok {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}

func (f *fakeRepo) ListHCPs(_ context.Context, hospitalID *uint64) ([]model.HCP, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	rows := make([]model.HCP, 0, len(f.hcps))
	for id := uint64(1); id <= 1000; id++ {
		row, ok := f.hcps[id]
		if !ok || !row.IsActive {
			continue
		}
		if hospitalID != nil && row.HospitalID != *hospitalID {
			continue
		}
		rows = append(rows, *row)
	}
	return rows, nil
}

func (f *fakeRepo) ListProducts(context.Context) ([]model.Product, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	rows := make([]model.Product, 0, len(f.products))
	for id := uint64(1); id <= 1000; id++ {
		if row, ok := f.products[id]; ok && row.IsActive {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}

func (f *fakeRepo) CreateVisit(_ context.Context, visit *model.Visit) error {
	if len(f.createErrors) > 0 {
		err := f.createErrors[0]
		f.createErrors = f.createErrors[1:]
		if err != nil {
			return err
		}
	}
	if visit.ID == 0 {
		visit.ID = f.nextID
		f.nextID++
	}
	visit.CreatedAt = f.now
	visit.UpdatedAt = f.now
	cp := *visit
	f.visits[visit.ID] = &cp
	return nil
}

func (f *fakeRepo) GetVisit(_ context.Context, id uint64) (*model.Visit, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	row, ok := f.visits[id]
	if !ok {
		return nil, nil
	}
	cp := *row
	return &cp, nil
}

func (f *fakeRepo) LockVisit(ctx context.Context, id uint64) (*model.Visit, error) {
	return f.GetVisit(ctx, id)
}

func (f *fakeRepo) UpdateVisit(_ context.Context, id uint64, fromStatus string, fields map[string]any) (int64, error) {
	row, ok := f.visits[id]
	if !ok || row.Status != fromStatus {
		return 0, nil
	}
	applyFields(row, fields)
	row.UpdatedAt = f.now
	return 1, nil
}

func applyFields(v *model.Visit, fields map[string]any) {
	for key, value := range fields {
		switch key {
		case "status":
			v.Status = value.(string)
		case "checkin_at":
			t := value.(time.Time)
			v.CheckInAt = &t
		case "checkin_lat":
			f := value.(float64)
			v.CheckInLat = &f
		case "checkin_lng":
			f := value.(float64)
			v.CheckInLng = &f
		case "checkin_distance_meters":
			f := value.(float64)
			v.CheckInDistanceMeters = &f
		case "checkout_at":
			t := value.(time.Time)
			v.CheckOutAt = &t
		case "checkout_lat":
			f := value.(float64)
			v.CheckOutLat = &f
		case "checkout_lng":
			f := value.(float64)
			v.CheckOutLng = &f
		case "checkout_distance_meters":
			f := value.(float64)
			v.CheckOutDistanceMeters = &f
		case "duration_seconds":
			u := value.(uint)
			v.DurationSeconds = &u
		case "deviation_meters":
			f := value.(float64)
			v.DeviationMeters = &f
		case "is_abnormal":
			b := value.(bool)
			v.IsAbnormal = &b
		case "abnormal_reasons":
			s := value.(string)
			v.AbnormalReasons = &s
		case "talking_points":
			s := value.(string)
			v.TalkingPoints = &s
		case "hcp_feedback":
			v.HCPFeedback = value.(*string)
		case "materials_distributed":
			b := value.(bool)
			v.MaterialsDistributed = &b
		case "follow_up":
			v.FollowUp = value.(*string)
		case "report_submitted_at":
			t := value.(time.Time)
			v.ReportSubmittedAt = &t
		case "cancelled_at":
			t := value.(time.Time)
			v.CancelledAt = &t
		case "cancel_reason":
			v.CancelReason = value.(*string)
		}
	}
}

func (f *fakeRepo) ListVisits(_ context.Context, filter repository.VisitListFilter) ([]repository.VisitListRecord, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	f.lastFilter = filter
	matched := make([]*model.Visit, 0)
	for id := uint64(0); id < f.nextID; id++ {
		v, ok := f.visits[id]
		if !ok {
			continue
		}
		if filter.MrID != nil && v.MrID != *filter.MrID {
			continue
		}
		if filter.HcpID != nil && v.HcpID != *filter.HcpID {
			continue
		}
		if filter.HospitalID != nil && v.HospitalID != *filter.HospitalID {
			continue
		}
		if filter.ProductID != nil && v.ProductID != *filter.ProductID {
			continue
		}
		if filter.Status != nil && v.Status != *filter.Status {
			continue
		}
		if filter.PlannedFrom != nil && v.PlannedAt.Before(*filter.PlannedFrom) {
			continue
		}
		if filter.PlannedTo != nil && !v.PlannedAt.Before(*filter.PlannedTo) {
			continue
		}
		if filter.Abnormal != nil {
			if v.IsAbnormal == nil || *v.IsAbnormal != *filter.Abnormal {
				continue
			}
		}
		matched = append(matched, v)
	}
	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].PlannedAt.After(matched[i].PlannedAt) {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}
	start := min((filter.Page-1)*filter.PageSize, len(matched))
	end := min(start+filter.PageSize, len(matched))
	rows := make([]repository.VisitListRecord, 0, end-start)
	for _, v := range matched[start:end] {
		rows = append(rows, f.row(v))
	}
	return rows, int64(len(matched)), nil
}

func (f *fakeRepo) row(v *model.Visit) repository.VisitListRecord {
	row := repository.VisitListRecord{
		ID: v.ID, VisitNo: v.VisitNo, Status: v.Status, MrID: v.MrID, HcpID: v.HcpID,
		HospitalID: v.HospitalID, ProductID: v.ProductID, PlannedAt: v.PlannedAt,
		CheckInAt: v.CheckInAt, CheckOutAt: v.CheckOutAt, DurationSeconds: v.DurationSeconds,
		DeviationMeters: v.DeviationMeters, IsAbnormal: v.IsAbnormal,
	}
	if mr, ok := f.mrs[v.MrID]; ok {
		row.MrName = mr.Name
	}
	if hcp, ok := f.hcps[v.HcpID]; ok {
		row.HcpName = hcp.Name
		row.HcpDepartment = hcp.Department
	}
	if h, ok := f.hospitals[v.HospitalID]; ok {
		row.HospitalName = h.Name
	}
	if p, ok := f.products[v.ProductID]; ok {
		row.ProductName = p.Name
	}
	return row
}

func (f *fakeRepo) ListMaterials(_ context.Context, visitID uint64) ([]model.VisitMaterial, error) {
	rows := append([]model.VisitMaterial(nil), f.materials[visitID]...)
	return rows, nil
}

func (f *fakeRepo) DeleteMaterials(_ context.Context, visitID uint64) error {
	delete(f.materials, visitID)
	return nil
}

func (f *fakeRepo) CreateMaterials(_ context.Context, materials []model.VisitMaterial) error {
	if len(materials) == 0 {
		return nil
	}
	visitID := materials[0].VisitID
	f.materials[visitID] = append(f.materials[visitID], materials...)
	return nil
}

func (f *fakeRepo) ProductStats(_ context.Context, filter repository.ProductStatFilter) ([]repository.ProductStatRow, error) {
	f.lastStat = filter
	if f.statsErr != nil {
		return nil, f.statsErr
	}
	return append([]repository.ProductStatRow(nil), f.stats...), nil
}

func duplicateKey() error {
	return &mysql.MySQLError{Number: 1062}
}
