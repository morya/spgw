package service

import (
	"github.com/morya/spgw/internal/daox"
	"github.com/morya/spgw/internal/models"
)

// ListIStat 查询北向账号统计列表
func ListIStat(req *models.ReqListIStat) (int64, []*models.IStat, error) {
	return daox.ListIStat(req)
}

// ListOStat 查询南向账号统计列表
func ListOStat(req *models.ReqListOStat) (int64, []*models.OStat, error) {
	return daox.ListOStat(req)
}
