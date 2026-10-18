package daox

import (
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
)

func ListOAccountLinks(oaccountID uint) ([]*models.LinkInfo, error) {
	var list []*models.LinkInfo
	if err := app.DB().Model(&models.LinkInfo{}).Where("oaccount_id = ?", oaccountID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func PutLinkInfo(link *models.LinkInfo) error {
	return app.DB().Save(link).Error
}

func GetLinkInfoByID(id uint) (*models.LinkInfo, error) {
	var link models.LinkInfo
	err := app.DB().Where("id = ?", id).First(&link).Error
	return &link, err
}

func DeleteLinkInfoByID(id uint) error {
	return app.DB().Where("id = ?", id).Delete(&models.LinkInfo{}).Error
}
