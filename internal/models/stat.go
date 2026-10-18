package models

// IStat 北向账号统计（按 IAccountID + NodeID 聚合，按 StatFlushInterval 周期累加）
type IStat struct {
	ID            uint   `json:"id" gorm:"primarykey"`
	IAccountID    uint32 `json:"iaccountID" gorm:"column:iaccount_id;index"`
	SpInfo        SpInfo `json:"spInfo" gorm:"foreignKey:IAccountID;references:ID"`
	NodeID        uint32 `json:"nodeID" gorm:"column:node_id"`
	SubmitSuccCnt uint32 `json:"submitSuccCnt" gorm:"column:submit_succ_cnt;default:0"`
	SubmitFailCnt uint32 `json:"submitFailCnt" gorm:"column:submit_fail_cnt;default:0"`
	ReportSuccCnt uint32 `json:"reportSuccCnt" gorm:"column:report_succ_cnt;default:0"`
	ReportFailCnt uint32 `json:"reportFailCnt" gorm:"column:report_fail_cnt;default:0"`
	CreatedAt     int64  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     int64  `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (IStat) TableName() string {
	return "gw_istat"
}

// OStat 南向账号统计（按 OAccountID + NodeID 聚合，按 StatFlushInterval 周期累加）
type OStat struct {
	ID            uint     `json:"id" gorm:"primarykey"`
	OAccountID    uint32   `json:"oaccountID" gorm:"column:oaccount_id;index"`
	OAccount      OAccount `json:"oAccount" gorm:"foreignKey:OAccountID;references:ID"`
	NodeID        uint32   `json:"nodeID" gorm:"column:node_id"`
	SubmitSuccCnt uint32   `json:"submitSuccCnt" gorm:"column:submit_succ_cnt;default:0"`
	SubmitFailCnt uint32   `json:"submitFailCnt" gorm:"column:submit_fail_cnt;default:0"`
	ReportSuccCnt uint32   `json:"reportSuccCnt" gorm:"column:report_succ_cnt;default:0"`
	ReportFailCnt uint32   `json:"reportFailCnt" gorm:"column:report_fail_cnt;default:0"`
	CreatedAt     int64    `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     int64    `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (OStat) TableName() string {
	return "gw_ostat"
}

// IStatListData 北向账号统计列表响应数据
type IStatListData struct {
	List  []*IStat `json:"list"`
	Total int64    `json:"total"`
}

// OStatListData 南向账号统计列表响应数据
type OStatListData struct {
	List  []*OStat `json:"list"`
	Total int64    `json:"total"`
}
