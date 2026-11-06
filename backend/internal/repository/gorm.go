package repository

import (
	"context"
	"errors"

	"github.com/mrvs/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormRepository struct {
	db *gorm.DB
}

// New 基于 GORM 连接构造 Repository。
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Transaction(ctx context.Context, fn func(Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormRepository{db: tx})
	})
}

func (r *gormRepository) FindMR(ctx context.Context, id uint64) (*model.MR, error) {
	var row model.MR
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *gormRepository) FindHCP(ctx context.Context, id uint64) (*model.HCP, error) {
	var row model.HCP
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *gormRepository) FindHospital(ctx context.Context, id uint64) (*model.Hospital, error) {
	var row model.Hospital
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *gormRepository) FindProduct(ctx context.Context, id uint64) (*model.Product, error) {
	var row model.Product
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *gormRepository) ListMRs(ctx context.Context) ([]model.MR, error) {
	rows := make([]model.MR, 0)
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *gormRepository) ListHospitals(ctx context.Context) ([]model.Hospital, error) {
	rows := make([]model.Hospital, 0)
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *gormRepository) ListHCPs(ctx context.Context, hospitalID *uint64) ([]model.HCP, error) {
	rows := make([]model.HCP, 0)
	q := r.db.WithContext(ctx).Where("is_active = ?", true)
	if hospitalID != nil {
		q = q.Where("hospital_id = ?", *hospitalID)
	}
	err := q.Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *gormRepository) ListProducts(ctx context.Context) ([]model.Product, error) {
	rows := make([]model.Product, 0)
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *gormRepository) CreateVisit(ctx context.Context, visit *model.Visit) error {
	return r.db.WithContext(ctx).Create(visit).Error
}

func (r *gormRepository) GetVisit(ctx context.Context, id uint64) (*model.Visit, error) {
	var row model.Visit
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *gormRepository) LockVisit(ctx context.Context, id uint64) (*model.Visit, error) {
	var row model.Visit
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *gormRepository) UpdateVisit(ctx context.Context, id uint64, fromStatus string, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&model.Visit{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *gormRepository) ListVisits(ctx context.Context, f VisitListFilter) ([]VisitListRecord, int64, error) {
	base := r.db.WithContext(ctx).Table("visit AS v")
	base = applyVisitFilters(base, f)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]VisitListRecord, 0)
	err := base.
		Select(`v.id, v.visit_no, v.status, v.mr_id, v.hcp_id, v.hospital_id, v.product_id,
			v.planned_at, v.checkin_at AS check_in_at, v.checkout_at AS check_out_at,
			v.duration_seconds, v.deviation_meters, v.is_abnormal,
			m.name AS mr_name, h.name AS hcp_name, h.department AS hcp_department,
			hp.name AS hospital_name, p.name AS product_name`).
		Joins("JOIN mr m ON m.id = v.mr_id").
		Joins("JOIN hcp h ON h.id = v.hcp_id").
		Joins("JOIN hospital hp ON hp.id = v.hospital_id").
		Joins("JOIN product p ON p.id = v.product_id").
		Order("v.planned_at DESC, v.id DESC").
		Limit(f.PageSize).
		Offset((f.Page - 1) * f.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func applyVisitFilters(db *gorm.DB, f VisitListFilter) *gorm.DB {
	if f.MrID != nil {
		db = db.Where("v.mr_id = ?", *f.MrID)
	}
	if f.HcpID != nil {
		db = db.Where("v.hcp_id = ?", *f.HcpID)
	}
	if f.HospitalID != nil {
		db = db.Where("v.hospital_id = ?", *f.HospitalID)
	}
	if f.ProductID != nil {
		db = db.Where("v.product_id = ?", *f.ProductID)
	}
	if f.Status != nil {
		db = db.Where("v.status = ?", *f.Status)
	}
	if f.PlannedFrom != nil {
		db = db.Where("v.planned_at >= ?", *f.PlannedFrom)
	}
	if f.PlannedTo != nil {
		db = db.Where("v.planned_at < ?", *f.PlannedTo)
	}
	if f.Abnormal != nil {
		if *f.Abnormal {
			db = db.Where("v.is_abnormal = 1")
		} else {
			db = db.Where("v.is_abnormal = 0")
		}
	}
	return db
}

func (r *gormRepository) ListMaterials(ctx context.Context, visitID uint64) ([]model.VisitMaterial, error) {
	rows := make([]model.VisitMaterial, 0)
	err := r.db.WithContext(ctx).Where("visit_id = ?", visitID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *gormRepository) DeleteMaterials(ctx context.Context, visitID uint64) error {
	return r.db.WithContext(ctx).Where("visit_id = ?", visitID).Delete(&model.VisitMaterial{}).Error
}

func (r *gormRepository) CreateMaterials(ctx context.Context, materials []model.VisitMaterial) error {
	if len(materials) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&materials).Error
}

func (r *gormRepository) ProductStats(ctx context.Context, f ProductStatFilter) ([]ProductStatRow, error) {
	q := r.db.WithContext(ctx).
		Table("visit AS v").
		Select(`v.product_id,
			p.code AS product_code,
			p.name AS product_name,
			COUNT(*) AS total_visits,
			COALESCE(SUM(CASE WHEN v.is_abnormal = 0 THEN 1 ELSE 0 END), 0) AS normal_visits,
			COALESCE(SUM(CASE WHEN v.is_abnormal = 1 THEN 1 ELSE 0 END), 0) AS abnormal_visits`).
		Joins("JOIN product p ON p.id = v.product_id").
		Where("v.status IN ?", []string{model.StatusCheckedOut, model.StatusReported}).
		Where("v.checkout_at >= ? AND v.checkout_at < ?", f.From, f.To)
	if f.MrID != nil {
		q = q.Where("v.mr_id = ?", *f.MrID)
	}
	if f.ProductID != nil {
		q = q.Where("v.product_id = ?", *f.ProductID)
	}

	rows := make([]ProductStatRow, 0)
	err := q.Group("v.product_id, p.code, p.name").
		Order("total_visits DESC, v.product_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
