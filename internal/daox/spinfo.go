package daox

import (
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
)

func GetSpInfoByID(id uint) (*models.SpInfo, error) {
	var spInfo *models.SpInfo
	err := app.DB().Where("id = ?", id).First(&spInfo).Error
	return spInfo, err
}

func PutSpInfo(spInfo *models.SpInfo) error {
	err := app.DB().Save(spInfo).Error
	if err != nil {
		return err
	}

	if spInfo.WalletId == 0 {
		w := models.SpWallet{
			SmsBalance: 0,
		}
		app.DB().Create(&w)
		spInfo.WalletId = w.ID
		return app.DB().Save(spInfo).Error
	}

	return nil
}

func DeleteSpInfoByID(id uint) error {
	return app.DB().Where("id = ?", id).Delete(&models.SpInfo{}).Error
}

func ListSpInfo(req *models.ReqListSpInfo) (int64, []*models.SpInfo, error) {
	var total int64
	var spInfoList []*models.SpInfo
	stmt := app.DB().Model(&models.SpInfo{}).Preload("User").Preload("SpWallet")
	if req.Disabled != nil {
		stmt = stmt.Where("disabled = ?", *req.Disabled)
	}
	if req.SpName != "" {
		stmt = stmt.Where("sp_name like ?", "%"+req.SpName+"%")
	}

	err := stmt.Count(&total).Error
	if err != nil {
		return 0, nil, err
	}

	stmt = stmt.Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize).Order("id desc")
	if err = stmt.Find(&spInfoList).Error; err != nil {
		return 0, nil, err
	}
	return total, spInfoList, nil
}
