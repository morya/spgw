package models

// SpInfoBase SP 基础信息字段（BaseSpInfo 与 ReqPutSpInfo 共享）
type SpInfoBase struct {
	UserId      uint   `gorm:"column:user_id;index" json:"userId" form:"userId"`            // 用户id
	WalletId    uint   `gorm:"column:wallet_id;index" json:"walletId" form:"walletId"`      // 钱包id
	SpName      string `gorm:"column:sp_name" json:"spName" form:"spName"`                  // 企业名称
	Spid        string `gorm:"column:spid;type:varchar(32);unique" json:"spid" form:"spid"` // 企业id
	ServiceCode string `gorm:"column:service_code" json:"serviceCode" form:"serviceCode"`   // 服务编码
	Signature   string `gorm:"column:signature" json:"signature" form:"signature"`          // 短信签名
	Remark      string `gorm:"column:remark" json:"remark" form:"remark"`                   // 备注
}

type BaseSpInfo struct {
	BaseModel
	SpInfoBase

	User     User     `gorm:"foreignKey:UserId;references:ID" json:"user"`       // 用户信息
	SpWallet SpWallet `gorm:"foreignKey:WalletId;references:ID" json:"spWallet"` // 钱包信息
}

// SpInfoConfig SP 配置公共字段（SpInfo 与 ReqPutSpInfo 共享）
type SpInfoConfig struct {
	Pswd        string  `gorm:"column:pswd" json:"pswd" form:"pswd"`                                                                                 // cmpp 协议的密码 明文存储，因为需要用来做 md5 加密
	Disabled    *bool   `gorm:"column:disabled;type:boolean" json:"disabled" form:"disabled"`                                                        // 是否禁用 true=禁用 false=正常
	Protocol    StrList `gorm:"column:protocol;type:varchar(64)" json:"protocol" form:"protocol" validate:"required,min=1,dive,oneof=cmpp20 cmpp30"` // cmpp20, cmpp30
	AllowHttp   *bool   `gorm:"column:allow_http;type:boolean" json:"allowHttp" form:"allowHttp"`                                                    // 是否允许 HTTP 访问
	Appid       string  `gorm:"column:appid;type:varchar(32)" json:"appid" form:"appid"`                                                             // HTTP 应用 ID
	ApiKey      string  `gorm:"column:api_key;type:varchar(32)" json:"apiKey" form:"apiKey"`                                                         // HTTP API 签名密钥
	CallbackUrl string  `gorm:"column:callback_url;type:varchar(256)" json:"callbackUrl" form:"callbackUrl"`                                         // 回调 URL
	SignType    string  `gorm:"column:sign_type;type:varchar(10)" json:"signType" form:"signType" default:"md5" validate:"omitempty,oneof=md5"`      // 签名类型, md5, sha1, 实际，仅支持 md5
	SignTmpl    string  `gorm:"column:sign_tmpl;type:varchar(256)" json:"signTmpl" form:"signTmpl"`                                                  // http API 签名模板
	MaxConn     uint    `gorm:"column:max_conn;type:int(11)" json:"maxConn" form:"maxConn" validate:"min=1,max=100"`                                 // 最大连接数
	MaxSpeed    int     `gorm:"column:max_speed;type:int(11);default:50" json:"maxSpeed" form:"maxSpeed" validate:"min=1,max=5000"`                  // 单接口最大提交速度
	Burst       int     `gorm:"column:burst;type:int(11);default:100" json:"burst" form:"burst" validate:"min=1,max=5000"`                           // 突发数(建议100)
	IpWhitelist StrList `gorm:"column:ip_whitelist;type:varchar(256)" json:"ipWhitelist,omitempty" validate:"omitempty,dive,ipv4"`                   // IP 白名单
}

type SpInfo struct {
	BaseSpInfo
	SpInfoConfig
}

func (SpInfo) TableName() string {
	return "gw_sp_info"
}

type ReqPutSpInfo struct {
	ID uint `json:"id" form:"id"`

	SpInfoBase
	SpInfoConfig
}

type ReqListSpInfo struct {
	BasePaging

	UserId      uint   `json:"userId" form:"userId"`                         // 用户id
	Disabled    *bool  `gorm:"column:disabled;type:boolean" json:"disabled"` // 是否禁用 true=禁用 false=正常
	SpName      string `json:"spName" form:"spName"`
	ServiceCode string `json:"serviceCode" form:"serviceCode"`
}

type ReqAdjustSpInfo struct {
	SpAccountId uint `json:"spAccountId"`

	MaxSpeed int `gorm:"column:max_speed;type:int(11);default:50" json:"maxSpeed" form:"maxSpeed" validate:"min=1,max=5000"`
	Burst    int `gorm:"column:burst;type:int(11);default:100" json:"burst" form:"burst" validate:"min=1,max=5000"` // 突发数(建议100)
}

// SpInfoListData SP信息列表响应数据
type SpInfoListData struct {
	List  []*SpInfo `json:"list"`
	Total int64     `json:"total"`
}
