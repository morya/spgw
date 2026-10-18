package daox

import (
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
)

func ListSpBill(req *models.ReqListSpBill) (int64, []*models.SpBillFile, error) {
	var total int64
	var list []*models.SpBillFile

	stmt := app.DB().Model(&models.SpBillFile{})
	if req.Spid != "" {
		stmt = stmt.Where("spid = ?", req.Spid)
	}
	if req.StartTime > 0 {
		stmt = stmt.Where("start_time >= ?", req.StartTime)
	}
	if req.EndTime > 0 {
		stmt = stmt.Where("end_time <= ?", req.EndTime)
	}

	if err := stmt.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if err := stmt.Scopes(req.Paginate()).Order("id desc").Find(&list).Error; err != nil {
		return 0, nil, err
	}
	return total, list, nil
}

func GetSpBillByID(id uint) (*models.SpBillFile, error) {
	var f models.SpBillFile
	if err := app.DB().First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}
