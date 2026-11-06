package service

import (
	"context"

	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/dto"
)

// Meta 返回主数据下拉列表。
func (s *VisitService) Meta(ctx context.Context, kind string, hospitalID *uint64) (any, error) {
	switch kind {
	case MetaMRs:
		rows, err := s.repo.ListMRs(ctx)
		if err != nil {
			return nil, internalError()
		}
		items := make([]dto.MRItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, dto.MRItem{ID: r.ID, EmployeeNo: r.EmployeeNo, Name: r.Name, Region: r.Region})
		}
		return dto.Items[dto.MRItem]{Items: items}, nil
	case MetaHospitals:
		rows, err := s.repo.ListHospitals(ctx)
		if err != nil {
			return nil, internalError()
		}
		items := make([]dto.HospitalItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, dto.HospitalItem{
				ID: r.ID, Name: r.Name, Address: r.Address, Latitude: r.Latitude, Longitude: r.Longitude,
			})
		}
		return dto.Items[dto.HospitalItem]{Items: items}, nil
	case MetaHCPs:
		rows, err := s.repo.ListHCPs(ctx, hospitalID)
		if err != nil {
			return nil, internalError()
		}
		items := make([]dto.HCPItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, dto.HCPItem{
				ID: r.ID, HospitalID: r.HospitalID, Name: r.Name, Department: r.Department, Title: r.Title,
			})
		}
		return dto.Items[dto.HCPItem]{Items: items}, nil
	case MetaProducts:
		rows, err := s.repo.ListProducts(ctx)
		if err != nil {
			return nil, internalError()
		}
		items := make([]dto.ProductItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, dto.ProductItem{ID: r.ID, Code: r.Code, Name: r.Name, Category: r.Category})
		}
		return dto.Items[dto.ProductItem]{Items: items}, nil
	default:
		return nil, apperr.Validation("不支持的主数据类型")
	}
}
