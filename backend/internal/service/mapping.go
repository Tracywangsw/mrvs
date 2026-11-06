package service

import (
	"context"
	"encoding/json"

	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/compliance"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/model"
)

func (s *VisitService) detail(ctx context.Context, visit *model.Visit) (*dto.VisitDetail, error) {
	mr, err := s.repo.FindMR(ctx, visit.MrID)
	if err != nil {
		return nil, internalError()
	}
	hcp, err := s.repo.FindHCP(ctx, visit.HcpID)
	if err != nil {
		return nil, internalError()
	}
	hospital, err := s.repo.FindHospital(ctx, visit.HospitalID)
	if err != nil {
		return nil, internalError()
	}
	product, err := s.repo.FindProduct(ctx, visit.ProductID)
	if err != nil {
		return nil, internalError()
	}
	if mr == nil || hcp == nil || hospital == nil || product == nil {
		return nil, apperr.Internal("拜访关联的主数据缺失")
	}
	materials, err := s.repo.ListMaterials(ctx, visit.ID)
	if err != nil {
		return nil, internalError()
	}

	detail := &dto.VisitDetail{
		ID:      visit.ID,
		VisitNo: visit.VisitNo,
		Status:  visit.Status,
		MR: dto.MRRef{
			ID:         mr.ID,
			EmployeeNo: mr.EmployeeNo,
			Name:       mr.Name,
		},
		HCP: dto.HCPRef{
			ID:         hcp.ID,
			Name:       hcp.Name,
			Department: hcp.Department,
			Title:      hcp.Title,
			HospitalID: hcp.HospitalID,
		},
		Hospital: dto.HospitalRef{
			ID:        hospital.ID,
			Name:      hospital.Name,
			Address:   hospital.Address,
			Latitude:  hospital.Latitude,
			Longitude: hospital.Longitude,
		},
		Product: dto.ProductRef{
			ID:       product.ID,
			Code:     product.Code,
			Name:     product.Name,
			Category: product.Category,
		},
		PlannedAt:       formatTime(visit.PlannedAt),
		DurationSeconds: visit.DurationSeconds,
		DeviationMeters: visit.DeviationMeters,
		IsAbnormal:      visit.IsAbnormal,
		Report: dto.Report{
			TalkingPoints:        visit.TalkingPoints,
			HCPFeedback:          visit.HCPFeedback,
			MaterialsDistributed: visit.MaterialsDistributed,
			FollowUp:             visit.FollowUp,
			ReportSubmittedAt:    formatTimePtr(visit.ReportSubmittedAt),
		},
		Materials:    make([]dto.Material, 0, len(materials)),
		CancelledAt:  formatTimePtr(visit.CancelledAt),
		CancelReason: visit.CancelReason,
		CreatedAt:    formatTime(visit.CreatedAt),
		UpdatedAt:    formatTime(visit.UpdatedAt),
	}

	if visit.CheckInAt != nil {
		detail.CheckIn = &dto.CheckPoint{
			At:             formatTime(*visit.CheckInAt),
			Latitude:       derefFloat(visit.CheckInLat),
			Longitude:      derefFloat(visit.CheckInLng),
			DistanceMeters: derefFloat(visit.CheckInDistanceMeters),
		}
	}
	if visit.CheckOutAt != nil {
		detail.CheckOut = &dto.CheckPoint{
			At:             formatTime(*visit.CheckOutAt),
			Latitude:       derefFloat(visit.CheckOutLat),
			Longitude:      derefFloat(visit.CheckOutLng),
			DistanceMeters: derefFloat(visit.CheckOutDistanceMeters),
		}
	}

	if visit.AbnormalReasons != nil {
		var reasons []compliance.Violation
		if err := json.Unmarshal([]byte(*visit.AbnormalReasons), &reasons); err != nil {
			return nil, apperr.Internal("异常原因数据损坏")
		}
		detail.AbnormalReasons = reasons
	}

	for _, m := range materials {
		detail.Materials = append(detail.Materials, dto.Material{
			ID:           m.ID,
			MaterialName: m.MaterialName,
			MaterialType: m.MaterialType,
			Quantity:     m.Quantity,
		})
	}
	return detail, nil
}

func derefFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
