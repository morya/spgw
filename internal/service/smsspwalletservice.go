package service

import (
	"errors"

	"github.com/morya/spgw/internal/daox"
	"github.com/morya/spgw/internal/models"
)

func GetSpWalletByID(walletId uint) (*models.SpWallet, error) {
	return daox.GetSpWalletByID(walletId)
}

func AdjustSpWallet(req *models.ReqAdjustSpWallet) error {
	if req.ChangeValue == 0 {
		return errors.New("变化值不能为0")
	}
	return daox.AdjustSpWalletBalance(req)
}
