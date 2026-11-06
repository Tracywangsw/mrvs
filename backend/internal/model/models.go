package model

import "time"

// 拜访状态枚举。
const (
	StatusPlanned    = "planned"
	StatusCheckedIn  = "checked_in"
	StatusCheckedOut = "checked_out"
	StatusReported   = "reported"
	StatusCancelled  = "cancelled"
)

// IsValidStatus 判断状态值是否属于状态机枚举。
func IsValidStatus(s string) bool {
	switch s {
	case StatusPlanned, StatusCheckedIn, StatusCheckedOut, StatusReported, StatusCancelled:
		return true
	default:
		return false
	}
}

// MR 医药代表。
type MR struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	EmployeeNo string    `gorm:"type:varchar(32);not null;uniqueIndex:uk_employee_no" json:"employee_no"`
	Name       string    `gorm:"type:varchar(64);not null" json:"name"`
	Region     *string   `gorm:"type:varchar(64)" json:"region"`
	IsActive   bool      `gorm:"not null;default:1" json:"is_active"`
	CreatedAt  time.Time `gorm:"type:datetime(3);not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime(3);not null" json:"updated_at"`
}

func (MR) TableName() string { return "mr" }

// Hospital 医院，经纬度是合规判定基准。
type Hospital struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(128);not null;index:idx_name" json:"name"`
	Address   *string   `gorm:"type:varchar(255)" json:"address"`
	Latitude  float64   `gorm:"type:decimal(9,6);not null" json:"latitude"`
	Longitude float64   `gorm:"type:decimal(9,6);not null" json:"longitude"`
	IsActive  bool      `gorm:"not null;default:1" json:"is_active"`
	CreatedAt time.Time `gorm:"type:datetime(3);not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null" json:"updated_at"`
}

func (Hospital) TableName() string { return "hospital" }

// HCP 医生。
type HCP struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HospitalID uint64    `gorm:"not null;index:idx_hospital_department,priority:1" json:"hospital_id"`
	Name       string    `gorm:"type:varchar(64);not null;index:idx_name" json:"name"`
	Department string    `gorm:"type:varchar(64);not null;index:idx_hospital_department,priority:2" json:"department"`
	Title      *string   `gorm:"type:varchar(32)" json:"title"`
	IsActive   bool      `gorm:"not null;default:1" json:"is_active"`
	CreatedAt  time.Time `gorm:"type:datetime(3);not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime(3);not null" json:"updated_at"`
}

func (HCP) TableName() string { return "hcp" }

// Product 产品。
type Product struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"type:varchar(32);not null;uniqueIndex:uk_code" json:"code"`
	Name      string    `gorm:"type:varchar(128);not null" json:"name"`
	Category  *string   `gorm:"type:varchar(64)" json:"category"`
	IsActive  bool      `gorm:"not null;default:1" json:"is_active"`
	CreatedAt time.Time `gorm:"type:datetime(3);not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null" json:"updated_at"`
}

func (Product) TableName() string { return "product" }

// Visit 拜访主表，合并了 call_report 的 1:1 记录字段。
type Visit struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VisitNo    string    `gorm:"type:varchar(32);not null;uniqueIndex:uk_visit_no" json:"visit_no"`
	MrID       uint64    `gorm:"not null;index:idx_mr_planned,priority:1" json:"mr_id"`
	HcpID      uint64    `gorm:"not null;index:idx_hcp" json:"hcp_id"`
	HospitalID uint64    `gorm:"not null" json:"hospital_id"`
	ProductID  uint64    `gorm:"not null;index:idx_product_checkout,priority:1" json:"product_id"`
	PlannedAt  time.Time `gorm:"type:datetime(3);not null;index:idx_mr_planned,priority:2" json:"planned_at"`
	Status     string    `gorm:"type:varchar(20);not null;default:'planned';index:idx_product_checkout,priority:2" json:"status"`

	CheckInAt             *time.Time `gorm:"type:datetime(3);column:checkin_at" json:"checkin_at"`
	CheckInLat            *float64   `gorm:"type:decimal(9,6);column:checkin_lat" json:"checkin_lat"`
	CheckInLng            *float64   `gorm:"type:decimal(9,6);column:checkin_lng" json:"checkin_lng"`
	CheckInDistanceMeters *float64   `gorm:"type:decimal(10,2);column:checkin_distance_meters" json:"checkin_distance_meters"`

	CheckOutAt             *time.Time `gorm:"type:datetime(3);column:checkout_at;index:idx_product_checkout,priority:3" json:"checkout_at"`
	CheckOutLat            *float64   `gorm:"type:decimal(9,6);column:checkout_lat" json:"checkout_lat"`
	CheckOutLng            *float64   `gorm:"type:decimal(9,6);column:checkout_lng" json:"checkout_lng"`
	CheckOutDistanceMeters *float64   `gorm:"type:decimal(10,2);column:checkout_distance_meters" json:"checkout_distance_meters"`

	DurationSeconds *uint    `gorm:"column:duration_seconds" json:"duration_seconds"`
	DeviationMeters *float64 `gorm:"type:decimal(10,2);column:deviation_meters" json:"deviation_meters"`
	IsAbnormal      *bool    `gorm:"column:is_abnormal" json:"is_abnormal"`
	AbnormalReasons *string  `gorm:"type:json;column:abnormal_reasons" json:"abnormal_reasons"`

	TalkingPoints        *string    `gorm:"type:text;column:talking_points" json:"talking_points"`
	HCPFeedback          *string    `gorm:"type:text;column:hcp_feedback" json:"hcp_feedback"`
	MaterialsDistributed *bool      `gorm:"column:materials_distributed" json:"materials_distributed"`
	FollowUp             *string    `gorm:"type:text;column:follow_up" json:"follow_up"`
	ReportSubmittedAt    *time.Time `gorm:"type:datetime(3);column:report_submitted_at" json:"report_submitted_at"`

	CancelledAt  *time.Time `gorm:"type:datetime(3);column:cancelled_at" json:"cancelled_at"`
	CancelReason *string    `gorm:"type:varchar(255);column:cancel_reason" json:"cancel_reason"`

	CreatedAt time.Time `gorm:"type:datetime(3);not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime(3);not null" json:"updated_at"`
}

func (Visit) TableName() string { return "visit" }

// VisitMaterial 资料派发明细。
type VisitMaterial struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VisitID      uint64    `gorm:"not null;index:idx_visit_id" json:"visit_id"`
	MaterialName string    `gorm:"type:varchar(128);not null" json:"material_name"`
	MaterialType *string   `gorm:"type:varchar(32)" json:"material_type"`
	Quantity     int       `gorm:"not null;default:1" json:"quantity"`
	CreatedAt    time.Time `gorm:"type:datetime(3);not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:datetime(3);not null" json:"updated_at"`
}

func (VisitMaterial) TableName() string { return "visit_material" }
