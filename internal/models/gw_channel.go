package models

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morya/spgw/internal/global/app"
)

type ReqPutOAccount struct {
	ID uint32 `json:"id" form:"id"`
	OAccountBase
	OAccountConfig

	ServiceCode string `json:"serviceCode" form:"serviceCode"`
	Spid        string `json:"spid" form:"spid"`
}

// ReqPutLinkInfo 下游渠道链路配置请求体，仅暴露可由管理后台配置的字段；
// IAccountID/INodeID/ONodeID 等运行时字段由 iagent/oagent 连接后填充，不在此暴露。
type ReqPutLinkInfo struct {
	ID            uint   `json:"id" form:"id"`
	OAccountID    uint32 `json:"oaccountId" form:"oaccountId" validate:"required"`
	Disabled      *bool  `json:"disabled" form:"disabled"`
	Spid          string `json:"spid" form:"spid" validate:"max=6"`
	Pswd          string `json:"pswd" form:"pswd" validate:"max=10"`
	Count         int    `json:"count" form:"count"`
	RemoteAddress string `json:"remoteAddress" form:"remoteAddress" validate:"required"`
	LocalAddress  string `json:"localAddress" form:"localAddress" validate:"max=128"`
	Proto         string `json:"proto" form:"proto" validate:"max=32,oneof=cmpp30 cmpp20"`
}

type ReqListOAccount struct {
	BasePaging
	Validator
	Name     string `json:"name" query:"name"`
	IsHttp   *bool  `json:"isHttp" query:"isHttp"`
	Disabled *bool  `json:"disabled" query:"disabled" queryDefault:"false" example:"false"`
}

func (r *ReqListOAccount) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

func (r *ReqListOAccount) Handle() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.Name != "" {
			db = db.Where("name like ?", "%"+r.Name+"%")
		}
		if r.IsHttp != nil {
			db = db.Where("is_http = ?", *r.IsHttp)
		}
		if r.Disabled != nil {
			db = db.Where("disabled = ?", *r.Disabled)
		}
		return db
	}
}

type OAccountList []*OAccount

func NewOAccountList() OAccountList {
	return OAccountList{}
}

func (list *OAccountList) Find(ctx context.Context, funcs ...func(*gorm.DB) *gorm.DB) error {
	return app.DB().WithContext(ctx).Scopes(funcs...).Find(list).Error
}

func (list *OAccountList) GetTotal(ctx context.Context, query ...func(*gorm.DB) *gorm.DB) (int64, error) {
	var total int64
	err := app.DB().WithContext(ctx).Model(&OAccount{}).Scopes(query...).Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

type OAccountWithConns struct {
	*OAccount
	ActiveConns uint `json:"activeConns"`
	TotalConns  uint `json:"totalConns"`
}

// OAccountWithConnsListData 通道账号(含连接数)列表响应数据
type OAccountWithConnsListData struct {
	List  []OAccountWithConns `json:"list"`
	Total int64               `json:"total"`
}

// GetLinkCounts 根据 OAccount ID 列表查询 LinkInfo 的总连接数和活跃连接数
func GetLinkCounts(ctx context.Context, ids []uint32) (totalMap map[uint32]uint, activeMap map[uint32]uint, err error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	type linkCount struct {
		OAccountID uint32
		Count      uint
	}

	// 查询总连接数
	var totalCounts []linkCount
	if err := app.DB().WithContext(ctx).Model(&LinkInfo{}).
		Select("oaccount_id, COUNT(*) as count").
		Where("oaccount_id IN ?", ids).
		Group("oaccount_id").
		Find(&totalCounts).Error; err != nil {
		return nil, nil, err
	}

	// 查询活跃连接数 (onode_id > 0 表示已成功建立连接)
	var activeCounts []linkCount
	if err := app.DB().WithContext(ctx).Model(&LinkInfo{}).
		Select("oaccount_id, COUNT(*) as count").
		Where("oaccount_id IN ? AND active = ?", ids, true).
		Group("oaccount_id").
		Find(&activeCounts).Error; err != nil {
		return nil, nil, err
	}

	totalMap = make(map[uint32]uint, len(totalCounts))
	for _, v := range totalCounts {
		totalMap[v.OAccountID] = v.Count
	}

	activeMap = make(map[uint32]uint, len(activeCounts))
	for _, v := range activeCounts {
		activeMap[v.OAccountID] = v.Count
	}

	return totalMap, activeMap, nil
}
