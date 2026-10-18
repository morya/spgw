package models

type Route struct {
	ID          uint32 `gorm:"primaryKey"`
	PartialAddr string `gorm:"column:partial_addr"`          // 省份号段
	StartIndex  int    `gorm:"column:start_index;default:4"` // 起始索引 默认值 4
}
