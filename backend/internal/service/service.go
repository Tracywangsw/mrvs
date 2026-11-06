package service

import (
	"context"
	"time"

	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/repository"
)

// 主数据类别。
const (
	MetaMRs       = "mrs"
	MetaHospitals = "hospitals"
	MetaHCPs      = "hcps"
	MetaProducts  = "products"
)

// CreateVisitInput 创建拜访计划的入参。
type CreateVisitInput struct {
	MRID       uint64
	HCPID      uint64
	HospitalID uint64
	ProductID  uint64
	PlannedAt  time.Time
}

// ListVisitsInput 拜访列表入参。
type ListVisitsInput struct {
	MrID       *uint64
	HcpID      *uint64
	HospitalID *uint64
	ProductID  *uint64
	Status     *string
	DateFrom   *time.Time
	DateTo     *time.Time
	Abnormal   *bool
	Page       int
	PageSize   int
}

// CheckInput 签到或签退入参。
type CheckInput struct {
	Latitude   float64
	Longitude  float64
	OccurredAt *time.Time
}

// MaterialInput 资料明细入参。
type MaterialInput struct {
	Name     string
	Type     *string
	Quantity int
}

// ReportInput 拜访记录入参。
type ReportInput struct {
	TalkingPoints        string
	HCPFeedback          *string
	MaterialsDistributed bool
	FollowUp             *string
	Materials            []MaterialInput
}

// DashboardInput 产品月度看板入参。
type DashboardInput struct {
	Month     string
	MrID      *uint64
	ProductID *uint64
}

// Service 是 Handler 层依赖的业务接口，便于用 fake 实现做 Handler 单测。
type Service interface {
	Create(ctx context.Context, in CreateVisitInput) (*dto.VisitDetail, error)
	List(ctx context.Context, in ListVisitsInput) (*dto.VisitListResult, error)
	Get(ctx context.Context, id uint64) (*dto.VisitDetail, error)
	CheckIn(ctx context.Context, id uint64, in CheckInput) (*dto.VisitDetail, error)
	CheckOut(ctx context.Context, id uint64, in CheckInput) (*dto.VisitDetail, error)
	Report(ctx context.Context, id uint64, in ReportInput) (*dto.VisitDetail, error)
	Cancel(ctx context.Context, id uint64, reason *string) (*dto.VisitDetail, error)
	DashboardProducts(ctx context.Context, in DashboardInput) (*dto.ProductDashboard, error)
	Meta(ctx context.Context, kind string, hospitalID *uint64) (any, error)
}

// VisitService 是 Service 的默认实现。
type VisitService struct {
	repo              repository.Repository
	rules             compliance.Rules
	allowTimeOverride bool
	now               func() time.Time
}

// New 构造 VisitService。
func New(repo repository.Repository, rules compliance.Rules, allowTimeOverride bool) *VisitService {
	return &VisitService{
		repo:              repo,
		rules:             rules,
		allowTimeOverride: allowTimeOverride,
		now:               func() time.Time { return time.Now().UTC() },
	}
}
