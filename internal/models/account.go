package models

// OAccountBase OAccount 基础字段，OAccount 与 ReqPutOAccount 共享
type OAccountBase struct {
	Name      string `gorm:"column:name;uniqueIndex;type:varchar(20)" json:"name" form:"name" validate:"omitempty,max=20"`
	Disabled  *bool  `gorm:"column:disabled;type:boolean" json:"disabled" form:"disabled"`
	NotBefore int64  `gorm:"column:not_before" json:"notBefore" form:"notBefore"` // 如果为0，则表示一直生效
	Province  string `gorm:"column:province;type:varchar(20)" json:"province" form:"province" validate:"max=20"`
	HlrCode   string `gorm:"column:hlr_code;type:varchar(20)" json:"hlrCode" form:"hlrCode" validate:"max=20"`
	Desc      string `gorm:"column:desc;type:varchar(256)" json:"desc" form:"desc" validate:"max=256"` // 备注信息
}

// OAccountConfig OAccount HTTP/协议配置字段，OAccount 与 ReqPutOAccount 共享
type OAccountConfig struct {
	IsHttp     *bool  `gorm:"column:is_http;type:boolean;default:0" json:"isHttp" form:"isHttp"`                           // 是否使用 HTTP 发送消息到南向账号，默认 false
	Appid      string `gorm:"column:appid;type:varchar(32)" json:"appid" form:"appid" validate:"max=32"`                   // 应用 ID
	ApiKey     string `gorm:"column:api_key;type:varchar(32)" json:"apiKey" form:"apiKey" validate:"max=32"`               // HTTP API 签名密钥
	ApiAddress string `gorm:"column:api_address;type:varchar(256)" json:"apiAddress" form:"apiAddress" validate:"max=256"` // HTTP API 地址
	HttpVendor string `gorm:"column:http_vendor;type:varchar(20)" json:"httpVendor" form:"httpVendor" validate:"max=20"`   // HTTP API 供应商类型
}

// OAccount 南向账号类型
type OAccount struct {
	ID uint32 `json:"id" gorm:"primaryKey"`
	OAccountBase
	OAccountConfig

	ChannelType   string `gorm:"column:channel_type;type:varchar(26)" json:"channelType" validate:"max=26"` // 渠道类型 SELF, THIRD, 完全未用到
	ForceClosedAt int64  `gorm:"column:force_closed_at;type:int(11)" json:"forceClosedAt"`                  // 手动关闭时间 0 表示未关闭
	AutoClosedAt  int64  `gorm:"column:auto_closed_at;type:int(11)" json:"autoClosedAt"`                    // 自动关闭时间 0 表示未关闭
}

func (OAccount) TableName() string {
	return "gw_oaccount"
}
