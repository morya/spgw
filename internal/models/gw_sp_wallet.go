package models

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morya/spgw/internal/global/app"
)

type SpWallet struct {
	BaseModel

	SmsBalance int64 `gorm:"column:sms_balance" json:"smsBalance"` // 短信余额
}

func (w *SpWallet) TableName() string {
	return "gw_sp_wallet"
}

type SpWalletLog struct {
	BaseModel
	WalletId    uint   `gorm:"column:wallet_id;index" json:"walletId"` // 关联的SpWallet id
	BeforeValue int64  `gorm:"column:before_value" json:"beforeValue"` // 变化前的余额
	ChangeValue int64  `gorm:"column:change_value" json:"changeValue"` // 变化值(正=充值, 负=消耗)
	Reason      string `gorm:"column:reason" json:"reason"`            // 变化原因(自助充值、手动充值、消耗)
}

func (l *SpWalletLog) TableName() string {
	return "gw_sp_wallet_log"
}

// ReqAdjustSpWallet 钱包余额调整(每次调整自动记录到SpWalletLog)
type ReqAdjustSpWallet struct {
	WalletId    uint   `json:"walletId" form:"walletId" validate:"required"`       // 钱包id
	ChangeValue int64  `json:"changeValue" form:"changeValue" validate:"required"` // 变化值(正=增加, 负=扣减)
	Reason      string `json:"reason" form:"reason" validate:"required"`           // 变化原因
}

// ReqListSpWalletLog 钱包变动记录列表查询
type ReqListSpWalletLog struct {
	BasePaging
	Validator
	WalletId uint `json:"walletId" form:"walletId" validate:"required"` // 钱包id
}

func (r *ReqListSpWalletLog) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

func (r *ReqListSpWalletLog) Handle() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.WalletId > 0 {
			db = db.Where("wallet_id = ?", r.WalletId)
		}
		return db
	}
}

type SpWalletLogList []*SpWalletLog

func NewSpWalletLogList() SpWalletLogList {
	return SpWalletLogList{}
}

func (list *SpWalletLogList) Find(ctx context.Context, funcs ...func(*gorm.DB) *gorm.DB) error {
	return app.DB().WithContext(ctx).Scopes(funcs...).Find(list).Error
}

func (list *SpWalletLogList) GetTotal(ctx context.Context, query ...func(*gorm.DB) *gorm.DB) (int64, error) {
	var total int64
	err := app.DB().WithContext(ctx).Model(&SpWalletLog{}).Scopes(query...).Count(&total).Error
	return total, err
}

// SpWalletLogListData 钱包变动记录列表响应数据
type SpWalletLogListData struct {
	List  []*SpWalletLog `json:"list"`
	Total int64          `json:"total"`
}
