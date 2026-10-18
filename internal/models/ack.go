package models

type AckSingleSend struct {
	MsgID string `json:"msgId"`
}

type MsgIdArr struct {
	MsgID string `json:"msgId"`
	Phone string `json:"phone"`
}
