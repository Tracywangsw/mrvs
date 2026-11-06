package repository

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/mrvs/backend/internal/model"
)

// VisitListFilter 拜访列表查询条件。
type VisitListFilter struct {
	MrID        *uint64
	HcpID       *uint64
	HospitalID  *uint64
	ProductID   *uint64
	Status      *string
	PlannedFrom *time.Time
	PlannedTo   *time.Time
	Abnormal    *bool
	Page        int
	PageSize    int
}

// VisitListRecord 是带主数据名称的列表行。
type VisitListRecord struct {
	ID              uint64
	VisitNo         string
	Status          string
	MrID            uint64
	MrName          string
	HcpID           uint64
	HcpName         string
	HcpDepartment   string
	HospitalID      uint64
	HospitalName    string
	ProductID       uint64
	ProductName     string
	PlannedAt       time.Time
	CheckInAt       *time.Time
	CheckOutAt      *time.Time
	DurationSeconds *uint
	DeviationMeters *float64
	IsAbnormal      *bool
}

// ProductStatFilter 产品看板统计条件。
type ProductStatFilter struct {
	From      time.Time
	To        time.Time
	MrID      *uint64
	ProductID *uint64
}

// ProductStatRow 是单个产品的统计结果。
type ProductStatRow struct {
	ProductID      uint64
	ProductCode    string
	ProductName    string
	TotalVisits    int64
	NormalVisits   int64
	AbnormalVisits int64
}

// Repository 定义 Service 层所需的全部数据访问能力。
// Transaction 会把当前实例替换为事务内的实例，因此所有写操作必须通过回调参数执行。
type Repository interface {
	Transaction(ctx context.Context, fn func(Repository) error) error

	FindMR(ctx context.Context, id uint64) (*model.MR, error)
	FindHCP(ctx context.Context, id uint64) (*model.HCP, error)
	FindHospital(ctx context.Context, id uint64) (*model.Hospital, error)
	FindProduct(ctx context.Context, id uint64) (*model.Product, error)
	ListMRs(ctx context.Context) ([]model.MR, error)
	ListHospitals(ctx context.Context) ([]model.Hospital, error)
	ListHCPs(ctx context.Context, hospitalID *uint64) ([]model.HCP, error)
	ListProducts(ctx context.Context) ([]model.Product, error)

	CreateVisit(ctx context.Context, visit *model.Visit) error
	GetVisit(ctx context.Context, id uint64) (*model.Visit, error)
	LockVisit(ctx context.Context, id uint64) (*model.Visit, error)
	UpdateVisit(ctx context.Context, id uint64, fromStatus string, fields map[string]any) (int64, error)
	ListVisits(ctx context.Context, filter VisitListFilter) ([]VisitListRecord, int64, error)
	ListMaterials(ctx context.Context, visitID uint64) ([]model.VisitMaterial, error)
	DeleteMaterials(ctx context.Context, visitID uint64) error
	CreateMaterials(ctx context.Context, materials []model.VisitMaterial) error
	ProductStats(ctx context.Context, filter ProductStatFilter) ([]ProductStatRow, error)
}

// IsDuplicateKey 判断错误是否为 MySQL 唯一键冲突（错误码 1062）。
func IsDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
