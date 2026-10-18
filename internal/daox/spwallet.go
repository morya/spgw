package daox

import (
	"errors"

	"gorm.io/gorm"

	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
)

func GetSpWalletByID(walletId uint) (*models.SpWallet, error) {
	var wallet models.SpWallet
	err := app.DB().Where("id = ?", walletId).First(&wallet).Error
	return &wallet, err
}

func GetSpWalletBySpTableId(spTableId uint) (*models.SpWallet, error) {
	return GetSpWalletByID(spTableId)
}

// AdjustSpWalletBalance 事务: 更新余额 + 写入变动记录
func AdjustSpWalletBalance(req *models.ReqAdjustSpWallet) error {
	return app.DB().Transaction(func(tx *gorm.DB) error {
		var wallet models.SpWallet
		if err := tx.Where("id = ?", req.WalletId).First(&wallet).Error; err != nil {
			return err
		}

		currValue := wallet.SmsBalance
		newBalance := wallet.SmsBalance + req.ChangeValue
		if newBalance < 0 {
			return errors.New("余额不足")
		}

		if err := tx.Model(&wallet).Update("sms_balance", newBalance).Error; err != nil {
			return err
		}

		logEntry := &models.SpWalletLog{
			WalletId:    wallet.ID,
			BeforeValue: currValue,
			ChangeValue: req.ChangeValue,
			Reason:      req.Reason,
		}
		return tx.Create(logEntry).Error
	})
}
