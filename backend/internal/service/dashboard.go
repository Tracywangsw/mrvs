package service

import (
	"context"
	"math"

	"github.com/mrvs/backend/internal/apperr"
	"github.com/mrvs/backend/internal/dto"
	"github.com/mrvs/backend/internal/model"
	"github.com/mrvs/backend/internal/repository"
	"github.com/mrvs/backend/internal/timeutil"
)

// DashboardProducts 返回产品维度月度看板。
func (s *VisitService) DashboardProducts(ctx context.Context, in DashboardInput) (*dto.ProductDashboard, error) {
	from, to, err := timeutil.MonthRange(in.Month)
	if err != nil {
		return nil, apperr.Validation(err.Error())
	}

	var product *model.Product
	if in.ProductID != nil {
		product, err = s.repo.FindProduct(ctx, *in.ProductID)
		if err != nil {
			return nil, internalError()
		}
		if product == nil {
			return nil, apperr.NotFound("产品不存在")
		}
	}

	rows, err := s.repo.ProductStats(ctx, repository.ProductStatFilter{
		From:      from,
		To:        to,
		MrID:      in.MrID,
		ProductID: in.ProductID,
	})
	if err != nil {
		return nil, internalError()
	}

	items := make([]dto.ProductDashboardItem, 0, len(rows)+1)
	var total, normal, abnormal int64
	for _, r := range rows {
		items = append(items, dto.ProductDashboardItem{
			ProductID:      r.ProductID,
			ProductCode:    r.ProductCode,
			ProductName:    r.ProductName,
			TotalVisits:    r.TotalVisits,
			NormalVisits:   r.NormalVisits,
			AbnormalVisits: r.AbnormalVisits,
			AbnormalRate:   abnormalRate(r.AbnormalVisits, r.TotalVisits),
		})
		total += r.TotalVisits
		normal += r.NormalVisits
		abnormal += r.AbnormalVisits
	}

	if in.ProductID != nil && len(items) == 0 {
		items = append(items, dto.ProductDashboardItem{
			ProductID:   product.ID,
			ProductCode: product.Code,
			ProductName: product.Name,
		})
	}

	result := &dto.ProductDashboard{
		Month:          in.Month,
		ProductID:      in.ProductID,
		TotalVisits:    total,
		NormalVisits:   normal,
		AbnormalVisits: abnormal,
		AbnormalRate:   abnormalRate(abnormal, total),
		Items:          items,
	}
	return result, nil
}

func abnormalRate(abnormal, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(abnormal)/float64(total)*10000) / 10000
}
