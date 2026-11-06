package database

import (
	"fmt"

	"github.com/mrvs/backend/internal/model"
	"gorm.io/gorm"
)

type seedHospital struct {
	Name      string
	Address   string
	Latitude  float64
	Longitude float64
}

var seedMRs = []model.MR{
	{EmployeeNo: "M001", Name: "张伟", Region: strPtr("华东大区")},
	{EmployeeNo: "M002", Name: "李娜", Region: strPtr("华南大区")},
}

var seedHospitals = []seedHospital{
	{Name: "北京协和医院", Address: "北京市东城区帅府园1号", Latitude: 39.912345, Longitude: 116.410123},
	{Name: "上海交通大学医学院附属瑞金医院", Address: "上海市黄浦区瑞金二路197号", Latitude: 31.211000, Longitude: 121.466000},
	{Name: "中山大学附属第一医院", Address: "广州市越秀区中山二路58号", Latitude: 23.129000, Longitude: 113.285000},
}

type seedHCP struct {
	HospitalName string
	Name         string
	Department   string
	Title        string
}

var seedHCPs = []seedHCP{
	{"北京协和医院", "王建国", "心内科", "主任医师"},
	{"北京协和医院", "陈晓", "呼吸科", "副主任医师"},
	{"上海交通大学医学院附属瑞金医院", "刘洋", "内分泌科", "主任医师"},
	{"上海交通大学医学院附属瑞金医院", "赵敏", "神经内科", "主治医师"},
	{"中山大学附属第一医院", "周杰", "肿瘤科", "主任医师"},
	{"中山大学附属第一医院", "孙丽", "消化内科", "副主任医师"},
}

var seedProducts = []model.Product{
	{Code: "P001", Name: "立普妥", Category: strPtr("心血管")},
	{Code: "P002", Name: "拜唐苹", Category: strPtr("内分泌")},
	{Code: "P003", Name: "希舒美", Category: strPtr("抗感染")},
}

// Seed 幂等写入演示用主数据。
func Seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for i := range seedMRs {
			mr := seedMRs[i]
			ok, err := recordExists(tx, &model.MR{}, "employee_no = ?", mr.EmployeeNo)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			if err := tx.Create(&mr).Error; err != nil {
				return fmt.Errorf("写入代表种子数据失败: %w", err)
			}
		}

		for _, h := range seedHospitals {
			ok, err := recordExists(tx, &model.Hospital{}, "name = ?", h.Name)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			row := model.Hospital{
				Name: h.Name, Address: strPtr(h.Address), Latitude: h.Latitude, Longitude: h.Longitude,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("写入医院种子数据失败: %w", err)
			}
		}

		for _, hcp := range seedHCPs {
			var hospital model.Hospital
			if err := tx.Where("name = ?", hcp.HospitalName).First(&hospital).Error; err != nil {
				return fmt.Errorf("查询医院 %s 失败: %w", hcp.HospitalName, err)
			}
			ok, err := recordExists(tx, &model.HCP{}, "hospital_id = ? AND name = ? AND department = ?",
				hospital.ID, hcp.Name, hcp.Department)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			row := model.HCP{
				HospitalID: hospital.ID, Name: hcp.Name, Department: hcp.Department, Title: strPtr(hcp.Title),
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("写入医生种子数据失败: %w", err)
			}
		}

		for i := range seedProducts {
			product := seedProducts[i]
			ok, err := recordExists(tx, &model.Product{}, "code = ?", product.Code)
			if err != nil {
				return err
			}
			if ok {
				continue
			}
			if err := tx.Create(&product).Error; err != nil {
				return fmt.Errorf("写入产品种子数据失败: %w", err)
			}
		}
		return nil
	})
}

func recordExists(tx *gorm.DB, dest any, query string, args ...any) (bool, error) {
	var count int64
	if err := tx.Model(dest).Where(query, args...).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func strPtr(s string) *string {
	return &s
}
