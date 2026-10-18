package daox

import (
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
)

// ListIStat 查询北向账号统计列表（支持按 iaccount_id / node_id 过滤，分页）
func ListIStat(req *models.ReqListIStat) (int64, []*models.IStat, error) {
	var total int64
	var list []*models.IStat

	stmt := app.DB().Model(&models.IStat{}).Preload("SpInfo")
	if req.IAccountID != 0 {
		stmt = stmt.Where("iaccount_id = ?", req.IAccountID)
	}
	if req.NodeID != 0 {
		stmt = stmt.Where("node_id = ?", req.NodeID)
	}

	if err := stmt.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if req.PageSize == -1 {
		req.PageSize = 1000
	}
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	order := req.Order
	if order == "" {
		order = "id desc"
	}

	if err := stmt.Order(order).
		Offset((req.PageNum - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&list).Error; err != nil {
		return 0, nil, err
	}
	return total, list, nil
}

// ListOStat 查询南向账号统计列表（支持按 oaccount_id / node_id 过滤，分页）
func ListOStat(req *models.ReqListOStat) (int64, []*models.OStat, error) {
	var total int64
	var list []*models.OStat

	stmt := app.DB().Model(&models.OStat{}).Preload("OAccount")
	if req.OAccountID != 0 {
		stmt = stmt.Where("oaccount_id = ?", req.OAccountID)
	}
	if req.NodeID != 0 {
		stmt = stmt.Where("node_id = ?", req.NodeID)
	}

	if err := stmt.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	if req.PageSize == -1 {
		req.PageSize = 1000
	}
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	order := req.Order
	if order == "" {
		order = "id desc"
	}

	if err := stmt.Order(order).
		Offset((req.PageNum - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&list).Error; err != nil {
		return 0, nil, err
	}
	return total, list, nil
}
