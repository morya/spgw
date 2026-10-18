package models

// RspBase 成功响应基础字段
type RspBase struct {
	Code      int    `example:"0" json:"code"`
	Message   string `example:"success" json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// RspSuccess 成功响应，携带 data
type RspSuccess struct {
	RspBase
	Data any `json:"data"`
}

// RspFailure 失败响应
type RspFailure struct {
	RspBase
	Data any `json:"data,omitempty"`
}
