package dto

import "github.com/mrvs/backend/internal/compliance"

// MRRef 代表精简信息。
type MRRef struct {
	ID         uint64 `json:"id"`
	EmployeeNo string `json:"employee_no"`
	Name       string `json:"name"`
}

// HCPRef 医生精简信息。
type HCPRef struct {
	ID         uint64  `json:"id"`
	Name       string  `json:"name"`
	Department string  `json:"department"`
	Title      *string `json:"title"`
	HospitalID uint64  `json:"hospital_id"`
}

// HospitalRef 医院精简信息。
type HospitalRef struct {
	ID        uint64  `json:"id"`
	Name      string  `json:"name"`
	Address   *string `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ProductRef 产品精简信息。
type ProductRef struct {
	ID       uint64  `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Category *string `json:"category"`
}

// CheckPoint 签到或签退的坐标与时间快照。
type CheckPoint struct {
	At             string  `json:"at"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	DistanceMeters float64 `json:"distance_meters"`
}

// Report 拜访记录。
type Report struct {
	TalkingPoints        *string `json:"talking_points"`
	HCPFeedback          *string `json:"hcp_feedback"`
	MaterialsDistributed *bool   `json:"materials_distributed"`
	FollowUp             *string `json:"follow_up"`
	ReportSubmittedAt    *string `json:"report_submitted_at"`
}

// Material 资料明细。
type Material struct {
	ID           uint64  `json:"id"`
	MaterialName string  `json:"material_name"`
	MaterialType *string `json:"material_type"`
	Quantity     int     `json:"quantity"`
}

// VisitDetail 拜访详情响应对象。
type VisitDetail struct {
	ID              uint64                 `json:"id"`
	VisitNo         string                 `json:"visit_no"`
	Status          string                 `json:"status"`
	MR              MRRef                  `json:"mr"`
	HCP             HCPRef                 `json:"hcp"`
	Hospital        HospitalRef            `json:"hospital"`
	Product         ProductRef             `json:"product"`
	PlannedAt       string                 `json:"planned_at"`
	CheckIn         *CheckPoint            `json:"checkin"`
	CheckOut        *CheckPoint            `json:"checkout"`
	DurationSeconds *uint                  `json:"duration_seconds"`
	DeviationMeters *float64               `json:"deviation_meters"`
	IsAbnormal      *bool                  `json:"is_abnormal"`
	AbnormalReasons []compliance.Violation `json:"abnormal_reasons"`
	Report          Report                 `json:"report"`
	Materials       []Material             `json:"materials"`
	CancelledAt     *string                `json:"cancelled_at"`
	CancelReason    *string                `json:"cancel_reason"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

// VisitListItem 拜访列表行。
type VisitListItem struct {
	ID              uint64   `json:"id"`
	VisitNo         string   `json:"visit_no"`
	Status          string   `json:"status"`
	MrID            uint64   `json:"mr_id"`
	MrName          string   `json:"mr_name"`
	HcpID           uint64   `json:"hcp_id"`
	HcpName         string   `json:"hcp_name"`
	HcpDepartment   string   `json:"hcp_department"`
	HospitalID      uint64   `json:"hospital_id"`
	HospitalName    string   `json:"hospital_name"`
	ProductID       uint64   `json:"product_id"`
	ProductName     string   `json:"product_name"`
	PlannedAt       string   `json:"planned_at"`
	CheckInAt       *string  `json:"checkin_at"`
	CheckOutAt      *string  `json:"checkout_at"`
	DurationSeconds *uint    `json:"duration_seconds"`
	DeviationMeters *float64 `json:"deviation_meters"`
	IsAbnormal      *bool    `json:"is_abnormal"`
}

// VisitListResult 拜访列表分页结果。
type VisitListResult struct {
	Items    []VisitListItem `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
}

// ProductDashboardItem 单个产品的月度统计。
type ProductDashboardItem struct {
	ProductID      uint64  `json:"product_id"`
	ProductCode    string  `json:"product_code"`
	ProductName    string  `json:"product_name"`
	TotalVisits    int64   `json:"total_visits"`
	NormalVisits   int64   `json:"normal_visits"`
	AbnormalVisits int64   `json:"abnormal_visits"`
	AbnormalRate   float64 `json:"abnormal_rate"`
}

// ProductDashboard 产品维度月度看板响应。
type ProductDashboard struct {
	Month          string                 `json:"month"`
	ProductID      *uint64                `json:"product_id"`
	TotalVisits    int64                  `json:"total_visits"`
	NormalVisits   int64                  `json:"normal_visits"`
	AbnormalVisits int64                  `json:"abnormal_visits"`
	AbnormalRate   float64                `json:"abnormal_rate"`
	Items          []ProductDashboardItem `json:"items"`
}

// MRItem 代表下拉项。
type MRItem struct {
	ID         uint64  `json:"id"`
	EmployeeNo string  `json:"employee_no"`
	Name       string  `json:"name"`
	Region     *string `json:"region"`
}

// HospitalItem 医院下拉项。
type HospitalItem struct {
	ID        uint64  `json:"id"`
	Name      string  `json:"name"`
	Address   *string `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// HCPItem 医生下拉项。
type HCPItem struct {
	ID         uint64  `json:"id"`
	HospitalID uint64  `json:"hospital_id"`
	Name       string  `json:"name"`
	Department string  `json:"department"`
	Title      *string `json:"title"`
}

// ProductItem 产品下拉项。
type ProductItem struct {
	ID       uint64  `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Category *string `json:"category"`
}

// Items 是主数据接口的通用 data 结构。
type Items[T any] struct {
	Items []T `json:"items"`
}
