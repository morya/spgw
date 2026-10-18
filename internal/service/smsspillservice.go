package service

import (
	"github.com/morya/spgw/internal/daox"
	"github.com/morya/spgw/internal/models"
)

func ListSpBill(req *models.ReqListSpBill) (int64, []*models.SpBillFile, error) {
	return daox.ListSpBill(req)
}

func GetSpBillByID(id uint) (*models.SpBillFile, error) {
	return daox.GetSpBillByID(id)
}
