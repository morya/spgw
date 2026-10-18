package models

type LinkInfo struct {
	ID       uint32 `json:"id" gorm:"primaryKey"`
	Disabled *bool  `json:"disabled" gorm:"column:disabled"`

	IAccountID uint32 `json:"iaccountID" gorm:"column:iaccount_id"`
	INodeID    uint32 `json:"inodeID" gorm:"column:inode_id"` // iagent节点ID

	OAccountID uint32 `json:"oaccountID" gorm:"column:oaccount_id" validate:"required"`
	ONodeID    uint32 `json:"onodeID" gorm:"column:onode_id"` // oagent节点ID 连接成功后，才会记录

	Active        *bool  `json:"active" gorm:"column:active;default:false"` // 是否活跃
	Spid          string `json:"spid" gorm:"column:spid" validate:"max=6,alphanum"`
	Pswd          string `json:"pswd" gorm:"column:pswd" validate:"max=16,alphanum"`
	Count         int    `json:"count" gorm:"column:count"`
	LinkTime      string `json:"linkTime" gorm:"column:link_time"` // 连接成功时间
	RemoteAddress string `json:"remoteAddress" gorm:"column:remote_address" validate:"hostname_port"`
	LocalAddress  string `json:"localAddress" gorm:"column:local_address" validate:"max=128"`     // 接收tcp时记录
	Proto         string `json:"proto" gorm:"column:proto" validate:"max=32,oneof=cmpp30 cmpp20"` // range: [cmpp30, cmpp20, -sgip, -smpp, -smgp] // "-" means, will be supported later
}

func (LinkInfo) TableName() string {
	return "gw_linkinfo"
}

// LinkListData 链路列表响应数据
type LinkListData struct {
	List  []*LinkInfo `json:"list"`
	Total int         `json:"total"`
}

// LinkDetailData 链路详情响应数据
type LinkDetailData struct {
	Link *LinkInfo `json:"link"`
}
