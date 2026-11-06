package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/model"
	"github.com/mrvs/backend/internal/repository"
	"github.com/mrvs/backend/internal/timeutil"
	"github.com/mrvs/backend/internal/visitno"
)

const maxVisitNoAttempts = 5

// Create 创建拜访计划。
func (s *VisitService) Create(ctx context.Context, in CreateVisitInput) (*dto.VisitDetail, error) {
	fields := map[string]string{}
	if in.MRID == 0 {
		fields["mr_id"] = "必填"
	}
	if in.HCPID == 0 {
		fields["hcp_id"] = "必填"
	}
	if in.HospitalID == 0 {
		fields["hospital_id"] = "必填"
	}
	if in.ProductID == 0 {
		fields["product_id"] = "必填"
	}
	if in.PlannedAt.IsZero() {
		fields["planned_at"] = "必填"
	}
	if len(fields) > 0 {
		return nil, apperr.ValidationFields("请求参数不合法", fields)
	}

	mr, err := s.repo.FindMR(ctx, in.MRID)
	if err != nil {
		return nil, internalError()
	}
	if mr == nil || !mr.IsActive {
		return nil, apperr.NotFound("医药代表不存在或已停用")
	}
	hcp, err := s.repo.FindHCP(ctx, in.HCPID)
	if err != nil {
		return nil, internalError()
	}
	if hcp == nil || !hcp.IsActive {
		return nil, apperr.NotFound("医生不存在或已停用")
	}
	hospital, err := s.repo.FindHospital(ctx, in.HospitalID)
	if err != nil {
		return nil, internalError()
	}
	if hospital == nil || !hospital.IsActive {
		return nil, apperr.NotFound("医院不存在或已停用")
	}
	product, err := s.repo.FindProduct(ctx, in.ProductID)
	if err != nil {
		return nil, internalError()
	}
	if product == nil || !product.IsActive {
		return nil, apperr.NotFound("产品不存在或已停用")
	}
	if hcp.HospitalID != hospital.ID {
		return nil, apperr.HCPHospitalMismatch("医生不属于所选医院")
	}

	visit := &model.Visit{
		MrID:       in.MRID,
		HcpID:      in.HCPID,
		HospitalID: in.HospitalID,
		ProductID:  in.ProductID,
		PlannedAt:  in.PlannedAt.UTC(),
		Status:     model.StatusPlanned,
	}

	var createErr error
	for attempt := 0; attempt < maxVisitNoAttempts; attempt++ {
		no, err := visitno.Generate(s.now(), nil)
		if err != nil {
			return nil, internalError()
		}
		visit.VisitNo = no
		createErr = s.repo.CreateVisit(ctx, visit)
		if createErr == nil {
			break
		}
		if !repository.IsDuplicateKey(createErr) {
			return nil, internalError()
		}
		visit.ID = 0
	}
	if createErr != nil {
		return nil, internalError()
	}
	return s.detail(ctx, visit)
}

// Get 查询拜访详情。
func (s *VisitService) Get(ctx context.Context, id uint64) (*dto.VisitDetail, error) {
	if id == 0 {
		return nil, apperr.NotFound("拜访不存在")
	}
	visit, err := s.repo.GetVisit(ctx, id)
	if err != nil {
		return nil, internalError()
	}
	if visit == nil {
		return nil, apperr.NotFound("拜访不存在")
	}
	return s.detail(ctx, visit)
}

// List 查询拜访列表。
func (s *VisitService) List(ctx context.Context, in ListVisitsInput) (*dto.VisitListResult, error) {
	if in.Status != nil && !model.IsValidStatus(*in.Status) {
		return nil, apperr.Validation("status 不合法")
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		return nil, apperr.Validation("page_size 不能超过 100")
	}

	records, total, err := s.repo.ListVisits(ctx, repository.VisitListFilter{
		MrID:        in.MrID,
		HcpID:       in.HcpID,
		HospitalID:  in.HospitalID,
		ProductID:   in.ProductID,
		Status:      in.Status,
		PlannedFrom: in.DateFrom,
		PlannedTo:   in.DateTo,
		Abnormal:    in.Abnormal,
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		return nil, internalError()
	}

	items := make([]dto.VisitListItem, 0, len(records))
	for _, r := range records {
		items = append(items, dto.VisitListItem{
			ID:              r.ID,
			VisitNo:         r.VisitNo,
			Status:          r.Status,
			MrID:            r.MrID,
			MrName:          r.MrName,
			HcpID:           r.HcpID,
			HcpName:         r.HcpName,
			HcpDepartment:   r.HcpDepartment,
			HospitalID:      r.HospitalID,
			HospitalName:    r.HospitalName,
			ProductID:       r.ProductID,
			ProductName:     r.ProductName,
			PlannedAt:       formatTime(r.PlannedAt),
			CheckInAt:       formatTimePtr(r.CheckInAt),
			CheckOutAt:      formatTimePtr(r.CheckOutAt),
			DurationSeconds: r.DurationSeconds,
			DeviationMeters: r.DeviationMeters,
			IsAbnormal:      r.IsAbnormal,
		})
	}
	return &dto.VisitListResult{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

// CheckIn 现场签到。
func (s *VisitService) CheckIn(ctx context.Context, id uint64, in CheckInput) (*dto.VisitDetail, error) {
	if err := validateCoordinates(in.Latitude, in.Longitude); err != nil {
		return nil, err
	}
	at, aerr := s.resolveTime(in.OccurredAt)
	if aerr != nil {
		return nil, aerr
	}

	err := s.repo.Transaction(ctx, func(tx repository.Repository) error {
		visit, err := tx.LockVisit(ctx, id)
		if err != nil {
			return err
		}
		if visit == nil {
			return apperr.NotFound("拜访不存在")
		}
		switch visit.Status {
		case model.StatusPlanned:
		case model.StatusCheckedIn:
			return apperr.DuplicateAction("拜访已签到")
		default:
			return apperr.InvalidState("当前状态不允许签到")
		}

		hospital, err := tx.FindHospital(ctx, visit.HospitalID)
		if err != nil {
			return err
		}
		if hospital == nil {
			return apperr.Internal("医院信息缺失")
		}
		distance := roundMeters(compliance.HaversineMeters(in.Latitude, in.Longitude, hospital.Latitude, hospital.Longitude))

		affected, err := tx.UpdateVisit(ctx, id, model.StatusPlanned, map[string]any{
			"status":                  model.StatusCheckedIn,
			"checkin_at":              at,
			"checkin_lat":             in.Latitude,
			"checkin_lng":             in.Longitude,
			"checkin_distance_meters": distance,
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return apperr.InvalidState("当前状态不允许签到")
		}
		return nil
	})
	if err != nil {
		return nil, asAppError(err)
	}
	return s.Get(ctx, id)
}

// CheckOut 拜访签退，并执行合规判定。
func (s *VisitService) CheckOut(ctx context.Context, id uint64, in CheckInput) (*dto.VisitDetail, error) {
	if err := validateCoordinates(in.Latitude, in.Longitude); err != nil {
		return nil, err
	}
	at, aerr := s.resolveTime(in.OccurredAt)
	if aerr != nil {
		return nil, aerr
	}

	err := s.repo.Transaction(ctx, func(tx repository.Repository) error {
		visit, err := tx.LockVisit(ctx, id)
		if err != nil {
			return err
		}
		if visit == nil {
			return apperr.NotFound("拜访不存在")
		}
		switch visit.Status {
		case model.StatusCheckedIn:
		case model.StatusCheckedOut:
			return apperr.DuplicateAction("拜访已签退")
		default:
			return apperr.InvalidState("当前状态不允许签退")
		}
		if visit.CheckInAt == nil || visit.CheckInLat == nil || visit.CheckInLng == nil {
			return apperr.Internal("签到信息缺失")
		}
		if at.Before(*visit.CheckInAt) {
			return apperr.TimeOrderInvalid("签退时间不能早于签到时间")
		}

		hospital, err := tx.FindHospital(ctx, visit.HospitalID)
		if err != nil {
			return err
		}
		if hospital == nil {
			return apperr.Internal("医院信息缺失")
		}

		result := compliance.Evaluate(compliance.Snapshot{
			CheckInAt:   *visit.CheckInAt,
			CheckOutAt:  at,
			CheckInLat:  *visit.CheckInLat,
			CheckInLng:  *visit.CheckInLng,
			CheckOutLat: in.Latitude,
			CheckOutLng: in.Longitude,
		}, compliance.HospitalAnchor{
			Latitude:  hospital.Latitude,
			Longitude: hospital.Longitude,
		}, s.rules)

		reasons, err := json.Marshal(result.Violations)
		if err != nil {
			return err
		}
		duration := uint(result.DurationSeconds)
		affected, err := tx.UpdateVisit(ctx, id, model.StatusCheckedIn, map[string]any{
			"status":                   model.StatusCheckedOut,
			"checkout_at":              at,
			"checkout_lat":             in.Latitude,
			"checkout_lng":             in.Longitude,
			"checkout_distance_meters": result.CheckOutDistanceMeters,
			"duration_seconds":         duration,
			"deviation_meters":         result.DeviationMeters,
			"is_abnormal":              result.IsAbnormal,
			"abnormal_reasons":         string(reasons),
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return apperr.InvalidState("当前状态不允许签退")
		}
		return nil
	})
	if err != nil {
		return nil, asAppError(err)
	}
	return s.Get(ctx, id)
}

// Report 提交或更新拜访记录。
func (s *VisitService) Report(ctx context.Context, id uint64, in ReportInput) (*dto.VisitDetail, error) {
	if len(in.Materials) > 20 {
		return nil, apperr.Validation("materials 最多 20 条")
	}
	materials, aerr := normalizeMaterials(in.Materials)
	if aerr != nil {
		return nil, aerr
	}
	if aerr := validateReportText(in); aerr != nil {
		return nil, aerr
	}
	if in.MaterialsDistributed && len(materials) == 0 {
		return nil, apperr.MaterialInconsistent("materials_distributed=true 时至少需要 1 条资料")
	}
	if !in.MaterialsDistributed && len(materials) > 0 {
		return nil, apperr.MaterialInconsistent("materials_distributed=false 时资料必须为空")
	}

	err := s.repo.Transaction(ctx, func(tx repository.Repository) error {
		visit, err := tx.LockVisit(ctx, id)
		if err != nil {
			return err
		}
		if visit == nil {
			return apperr.NotFound("拜访不存在")
		}
		if visit.Status != model.StatusCheckedOut && visit.Status != model.StatusReported {
			return apperr.InvalidState("当前状态不允许提交拜访记录")
		}

		fields := map[string]any{
			"status":                model.StatusReported,
			"talking_points":        in.TalkingPoints,
			"hcp_feedback":          emptyToNil(in.HCPFeedback),
			"materials_distributed": in.MaterialsDistributed,
			"follow_up":             emptyToNil(in.FollowUp),
		}
		if visit.Status == model.StatusCheckedOut {
			fields["report_submitted_at"] = s.now()
		}

		affected, err := tx.UpdateVisit(ctx, id, visit.Status, fields)
		if err != nil {
			return err
		}
		if affected == 0 {
			return apperr.InvalidState("当前状态不允许提交拜访记录")
		}

		if err := tx.DeleteMaterials(ctx, id); err != nil {
			return err
		}
		if len(materials) > 0 {
			for i := range materials {
				materials[i].VisitID = id
			}
			if err := tx.CreateMaterials(ctx, materials); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, asAppError(err)
	}
	return s.Get(ctx, id)
}

// Cancel 取消拜访。
func (s *VisitService) Cancel(ctx context.Context, id uint64, reason *string) (*dto.VisitDetail, error) {
	if reason != nil && utf8.RuneCountInString(*reason) > 255 {
		return nil, apperr.Validation("reason 最多 255 字符")
	}

	err := s.repo.Transaction(ctx, func(tx repository.Repository) error {
		visit, err := tx.LockVisit(ctx, id)
		if err != nil {
			return err
		}
		if visit == nil {
			return apperr.NotFound("拜访不存在")
		}
		if visit.Status != model.StatusPlanned {
			return apperr.InvalidState("当前状态不允许取消")
		}

		affected, err := tx.UpdateVisit(ctx, id, model.StatusPlanned, map[string]any{
			"status":        model.StatusCancelled,
			"cancelled_at":  s.now(),
			"cancel_reason": emptyToNil(reason),
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return apperr.InvalidState("当前状态不允许取消")
		}
		return nil
	})
	if err != nil {
		return nil, asAppError(err)
	}
	return s.Get(ctx, id)
}

func (s *VisitService) resolveTime(occurredAt *time.Time) (time.Time, *apperr.Error) {
	if occurredAt == nil {
		return s.now(), nil
	}
	if !s.allowTimeOverride {
		return time.Time{}, apperr.TimeOverrideDisabled("演示时间覆盖开关未开启")
	}
	return occurredAt.UTC(), nil
}

func validateCoordinates(lat, lng float64) *apperr.Error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return apperr.BusinessRule("latitude 必须在 -90 到 90 之间")
	}
	if math.IsNaN(lng) || math.IsInf(lng, 0) || lng < -180 || lng > 180 {
		return apperr.BusinessRule("longitude 必须在 -180 到 180 之间")
	}
	return nil
}

func validateReportText(in ReportInput) *apperr.Error {
	if strings.TrimSpace(in.TalkingPoints) == "" {
		return apperr.ValidationFields("请求参数不合法", map[string]string{"talking_points": "必填"})
	}
	if utf8.RuneCountInString(in.TalkingPoints) > 5000 {
		return apperr.ValidationFields("请求参数不合法", map[string]string{"talking_points": "最多 5000 字符"})
	}
	if in.HCPFeedback != nil && utf8.RuneCountInString(*in.HCPFeedback) > 5000 {
		return apperr.ValidationFields("请求参数不合法", map[string]string{"hcp_feedback": "最多 5000 字符"})
	}
	if in.FollowUp != nil && utf8.RuneCountInString(*in.FollowUp) > 5000 {
		return apperr.ValidationFields("请求参数不合法", map[string]string{"follow_up": "最多 5000 字符"})
	}
	return nil
}

var allowedMaterialTypes = map[string]bool{
	"literature": true,
	"guideline":  true,
	"slide":      true,
	"other":      true,
}

func normalizeMaterials(in []MaterialInput) ([]model.VisitMaterial, *apperr.Error) {
	out := make([]model.VisitMaterial, 0, len(in))
	for _, m := range in {
		name := strings.TrimSpace(m.Name)
		if name == "" || utf8.RuneCountInString(name) > 128 {
			return nil, apperr.ValidationFields("请求参数不合法", map[string]string{
				"materials": "material_name 长度必须为 1 至 128 字符",
			})
		}
		quantity := m.Quantity
		if quantity == 0 {
			quantity = 1
		}
		if quantity < 0 || quantity > 1000 {
			return nil, apperr.ValidationFields("请求参数不合法", map[string]string{
				"materials": "quantity 必须在 1 到 1000 之间",
			})
		}
		var materialType *string
		if m.Type != nil && *m.Type != "" {
			if !allowedMaterialTypes[*m.Type] {
				return nil, apperr.ValidationFields("请求参数不合法", map[string]string{
					"materials": "material_type 不合法",
				})
			}
			t := *m.Type
			materialType = &t
		}
		out = append(out, model.VisitMaterial{
			MaterialName: name,
			MaterialType: materialType,
			Quantity:     quantity,
		})
	}
	return out, nil
}

func internalError() *apperr.Error {
	return apperr.Internal("服务端内部错误")
}

func asAppError(err error) *apperr.Error {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae
	}
	return internalError()
}

func roundMeters(v float64) float64 {
	return math.Round(v*100) / 100
}

func emptyToNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func formatTime(t time.Time) string {
	return timeutil.Format(t)
}

func formatTimePtr(t *time.Time) *string {
	return timeutil.FormatPtr(t)
}
