package daox

import (
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
)

func PutOAccount(oa *models.OAccount) error {
	return app.DB().Save(oa).Error
}

func GetOAccountByID(id uint) (*models.OAccount, error) {
	var oa models.OAccount
	err := app.DB().Where("id = ?", id).First(&oa).Error
	return &oa, err
}

func DeleteOAccountByID(id uint) error {
	return app.DB().Where("id = ?", id).Delete(&models.OAccount{}).Error
}

func ListOAccount(req *models.ReqListOAccount) (int64, []*models.OAccount, error) {
	var total int64
	var list []*models.OAccount
	stmt := app.DB().Model(&models.OAccount{})
	if req.Name != "" {
		stmt = stmt.Where("name like ?", "%"+req.Name+"%")
	}
	if req.IsHttp != nil {
		stmt = stmt.Where("is_http = ?", *req.IsHttp)
	}
	if req.Disabled != nil {
		stmt = stmt.Where("disabled = ?", *req.Disabled)
	}

	if err := stmt.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	stmt = stmt.Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize).Order("id desc")
	if err := stmt.Find(&list).Error; err != nil {
		return 0, nil, err
	}
	return total, list, nil
}
