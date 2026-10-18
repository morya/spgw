package models

type SpBillFile struct {
	BaseModel

	Spid string `gorm:"column:spid;type:varchar(32);index" json:"spid"`

	FileName       string `gorm:"column:file_name;type:varchar(256)" json:"fileName"`
	OriginalSize   int64  `gorm:"column:original_size" json:"originalSize"`
	CompressedSize int64  `gorm:"column:compressed_size" json:"compressedSize"`

	BillCount int64 `gorm:"column:bill_count" json:"billCount"`

	StartTime int64 `gorm:"column:start_time" json:"startTime"`
	EndTime   int64 `gorm:"column:end_time" json:"endTime"`

	NodeID uint32 `gorm:"column:node_id" json:"nodeID"`
}

func (SpBillFile) TableName() string {
	return "gw_spbill_file"
}

type ReqListSpBill struct {
	BasePaging

	Spid      string `json:"spid" form:"spid"`
	StartTime int64  `json:"startTime" form:"startTime"`
	EndTime   int64  `json:"endTime" form:"endTime"`
}

// SpBillListData 话单文件列表响应数据
type SpBillListData struct {
	List  []*SpBillFile `json:"list"`
	Total int64         `json:"total"`
}
