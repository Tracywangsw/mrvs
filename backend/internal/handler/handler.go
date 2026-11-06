package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/response"
	"github.com/mrvs/backend/internal/service"
	"github.com/mrvs/backend/internal/timeutil"
)

// Handler 负责参数绑定、基础校验与响应输出。
type Handler struct {
	svc service.Service
}

// New 构造 Handler。
func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// Health 健康检查。
func (h *Handler) Health(_ context.Context, c *app.RequestContext) {
	response.OKHealth(c)
}

type createVisitRequest struct {
	MrID       int64  `json:"mr_id"`
	HcpID      int64  `json:"hcp_id"`
	HospitalID int64  `json:"hospital_id"`
	ProductID  int64  `json:"product_id"`
	PlannedAt  string `json:"planned_at"`
}

// Create 创建拜访计划。
func (h *Handler) Create(ctx context.Context, c *app.RequestContext) {
	var req createVisitRequest
	if err := c.BindJSON(&req); err != nil {
		writeError(c, apperr.Validation("请求体格式不合法"))
		return
	}
	fields := map[string]string{}
	if req.MrID <= 0 {
		fields["mr_id"] = "必填且必须为正整数"
	}
	if req.HcpID <= 0 {
		fields["hcp_id"] = "必填且必须为正整数"
	}
	if req.HospitalID <= 0 {
		fields["hospital_id"] = "必填且必须为正整数"
	}
	if req.ProductID <= 0 {
		fields["product_id"] = "必填且必须为正整数"
	}
	plannedAt, err := parseRequiredTime(req.PlannedAt)
	if err != nil {
		fields["planned_at"] = "必须为 RFC3339 格式"
	}
	if len(fields) > 0 {
		writeError(c, apperr.ValidationFields("请求参数不合法", fields))
		return
	}

	detail, err := h.svc.Create(ctx, service.CreateVisitInput{
		MRID:       uint64(req.MrID),
		HCPID:      uint64(req.HcpID),
		HospitalID: uint64(req.HospitalID),
		ProductID:  uint64(req.ProductID),
		PlannedAt:  plannedAt,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusCreated, detail)
}

// List 拜访列表。
func (h *Handler) List(ctx context.Context, c *app.RequestContext) {
	fields := map[string]string{}
	mrID, err := optionalUint(c, "mr_id")
	if err != nil {
		fields["mr_id"] = err.Error()
	}
	hcpID, err := optionalUint(c, "hcp_id")
	if err != nil {
		fields["hcp_id"] = err.Error()
	}
	hospitalID, err := optionalUint(c, "hospital_id")
	if err != nil {
		fields["hospital_id"] = err.Error()
	}
	productID, err := optionalUint(c, "product_id")
	if err != nil {
		fields["product_id"] = err.Error()
	}
	abnormal, err := optionalBool(c, "abnormal")
	if err != nil {
		fields["abnormal"] = err.Error()
	}
	page, err := optionalInt(c, "page")
	if err != nil {
		fields["page"] = err.Error()
	}
	pageSize, err := optionalInt(c, "page_size")
	if err != nil {
		fields["page_size"] = err.Error()
	}

	var status *string
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		status = &v
	}

	dateFrom, dateTo, err := parseDateRange(c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		fields["date_from"] = err.Error()
	}
	if len(fields) > 0 {
		writeError(c, apperr.ValidationFields("请求参数不合法", fields))
		return
	}

	result, err := h.svc.List(ctx, service.ListVisitsInput{
		MrID:       mrID,
		HcpID:      hcpID,
		HospitalID: hospitalID,
		ProductID:  productID,
		Status:     status,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Abnormal:   abnormal,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, result)
}

// Get 拜访详情。
func (h *Handler) Get(ctx context.Context, c *app.RequestContext) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	detail, err := h.svc.Get(ctx, id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, detail)
}

type checkRequest struct {
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	OccurredAt *string  `json:"occurred_at"`
}

// CheckIn 现场签到。
func (h *Handler) CheckIn(ctx context.Context, c *app.RequestContext) {
	h.check(ctx, c, h.svc.CheckIn)
}

// CheckOut 拜访签退。
func (h *Handler) CheckOut(ctx context.Context, c *app.RequestContext) {
	h.check(ctx, c, h.svc.CheckOut)
}

type checkAction func(context.Context, uint64, service.CheckInput) (*dto.VisitDetail, error)

func (h *Handler) check(
	ctx context.Context,
	c *app.RequestContext,
	action checkAction,
) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var req checkRequest
	if err := c.BindJSON(&req); err != nil {
		writeError(c, apperr.Validation("请求体格式不合法"))
		return
	}
	fields := map[string]string{}
	if req.Latitude == nil {
		fields["latitude"] = "必填"
	}
	if req.Longitude == nil {
		fields["longitude"] = "必填"
	}
	var occurredAt *time.Time
	if req.OccurredAt != nil && *req.OccurredAt != "" {
		t, err := timeutil.ParseRFC3339(*req.OccurredAt)
		if err != nil {
			fields["occurred_at"] = "必须为 RFC3339 格式"
		} else {
			occurredAt = &t
		}
	}
	if len(fields) > 0 {
		writeError(c, apperr.ValidationFields("请求参数不合法", fields))
		return
	}

	detail, err := action(ctx, id, service.CheckInput{
		Latitude:   *req.Latitude,
		Longitude:  *req.Longitude,
		OccurredAt: occurredAt,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, detail)
}

type materialRequest struct {
	MaterialName string  `json:"material_name"`
	MaterialType *string `json:"material_type"`
	Quantity     int     `json:"quantity"`
}

type reportRequest struct {
	TalkingPoints        *string           `json:"talking_points"`
	HCPFeedback          *string           `json:"hcp_feedback"`
	MaterialsDistributed *bool             `json:"materials_distributed"`
	FollowUp             *string           `json:"follow_up"`
	Materials            []materialRequest `json:"materials"`
}

// Report 提交或更新拜访记录。
func (h *Handler) Report(ctx context.Context, c *app.RequestContext) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var req reportRequest
	if err := c.BindJSON(&req); err != nil {
		writeError(c, apperr.Validation("请求体格式不合法"))
		return
	}
	fields := map[string]string{}
	if req.TalkingPoints == nil {
		fields["talking_points"] = "必填"
	}
	if req.MaterialsDistributed == nil {
		fields["materials_distributed"] = "必填"
	}
	if len(fields) > 0 {
		writeError(c, apperr.ValidationFields("请求参数不合法", fields))
		return
	}

	materials := make([]service.MaterialInput, 0, len(req.Materials))
	for _, m := range req.Materials {
		materials = append(materials, service.MaterialInput{Name: m.MaterialName, Type: m.MaterialType, Quantity: m.Quantity})
	}
	detail, err := h.svc.Report(ctx, id, service.ReportInput{
		TalkingPoints:        *req.TalkingPoints,
		HCPFeedback:          req.HCPFeedback,
		MaterialsDistributed: *req.MaterialsDistributed,
		FollowUp:             req.FollowUp,
		Materials:            materials,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, detail)
}

type cancelRequest struct {
	Reason *string `json:"reason"`
}

// Cancel 取消拜访。
func (h *Handler) Cancel(ctx context.Context, c *app.RequestContext) {
	id, err := pathID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var req cancelRequest
	if len(c.Request.Body()) > 0 {
		if err := c.BindJSON(&req); err != nil {
			writeError(c, apperr.Validation("请求体格式不合法"))
			return
		}
	}
	detail, err := h.svc.Cancel(ctx, id, req.Reason)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, detail)
}

// DashboardProducts 产品维度月度看板。
func (h *Handler) DashboardProducts(ctx context.Context, c *app.RequestContext) {
	fields := map[string]string{}
	month := strings.TrimSpace(c.Query("month"))
	if month == "" {
		fields["month"] = "必填"
	}
	mrID, err := optionalUint(c, "mr_id")
	if err != nil {
		fields["mr_id"] = err.Error()
	}
	productID, err := optionalUint(c, "product_id")
	if err != nil {
		fields["product_id"] = err.Error()
	}
	if len(fields) > 0 {
		writeError(c, apperr.ValidationFields("请求参数不合法", fields))
		return
	}
	result, err := h.svc.DashboardProducts(ctx, service.DashboardInput{
		Month: month, MrID: mrID, ProductID: productID,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, result)
}

// MetaMRs 代表下拉。
func (h *Handler) MetaMRs(ctx context.Context, c *app.RequestContext) {
	h.meta(ctx, c, service.MetaMRs, nil)
}

// MetaHospitals 医院下拉。
func (h *Handler) MetaHospitals(ctx context.Context, c *app.RequestContext) {
	h.meta(ctx, c, service.MetaHospitals, nil)
}

// MetaHCPs 医生下拉，支持按医院过滤。
func (h *Handler) MetaHCPs(ctx context.Context, c *app.RequestContext) {
	var hospitalID *uint64
	v, err := optionalUint(c, "hospital_id")
	if err != nil {
		writeError(c, apperr.ValidationFields("请求参数不合法", map[string]string{"hospital_id": err.Error()}))
		return
	}
	hospitalID = v
	h.meta(ctx, c, service.MetaHCPs, hospitalID)
}

// MetaProducts 产品下拉。
func (h *Handler) MetaProducts(ctx context.Context, c *app.RequestContext) {
	h.meta(ctx, c, service.MetaProducts, nil)
}

func (h *Handler) meta(ctx context.Context, c *app.RequestContext, kind string, hospitalID *uint64) {
	data, err := h.svc.Meta(ctx, kind, hospitalID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, http.StatusOK, data)
}

func writeError(c *app.RequestContext, err error) {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		response.Fail(c, ae)
		return
	}
	response.Fail(c, apperr.Internal("服务端内部错误"))
}

func pathID(c *app.RequestContext) (uint64, error) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, apperr.ValidationFields("请求参数不合法", map[string]string{"id": "必须为正整数"})
	}
	return id, nil
}

func parseRequiredTime(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, errors.New("planned_at 必填")
	}
	return timeutil.ParseRFC3339(raw)
}

func optionalUint(c *app.RequestContext, key string) (*uint64, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		return nil, errors.New(key + " 必须为正整数")
	}
	return &v, nil
}

func optionalInt(c *app.RequestContext, key string) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(key + " 必须为整数")
	}
	return v, nil
}

func optionalBool(c *app.RequestContext, key string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	switch strings.ToLower(raw) {
	case "true", "1":
		v := true
		return &v, nil
	case "false", "0":
		v := false
		return &v, nil
	default:
		return nil, errors.New(key + " 必须为 true 或 false")
	}
}

func parseDateRange(from, to string) (start, end *time.Time, err error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" && to == "" {
		return nil, nil, nil
	}
	if from != "" {
		s, parseErr := timeutil.ParseDateStart(from)
		if parseErr != nil {
			return nil, nil, errors.New("date_from 必须为 YYYY-MM-DD 格式")
		}
		start = &s
	}
	if to != "" {
		e, parseErr := timeutil.ParseDateEnd(to)
		if parseErr != nil {
			return nil, nil, errors.New("date_to 必须为 YYYY-MM-DD 格式")
		}
		end = &e
	}
	if start != nil && end != nil && !end.After(*start) {
		return nil, nil, errors.New("date_to 不能早于 date_from")
	}
	return start, end, nil
}
