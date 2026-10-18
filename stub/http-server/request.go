package main

type BasicFields struct {
	AppID     string `json:"appId"`
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
	Version   string `json:"version"`
	SignType  string `json:"signType"`
	Content   string `json:"content"`
	ExtendNo  string `json:"extendNo"`
}

type ReqSingleSend struct {
	BasicFields
	Phone   string `json:"phone"`
	SpMsgId string `json:"spMsgId"`
}

type BatchSend struct {
	Phone   string `json:"phone"`
	SpMsgId string `json:"spMsgId"`
}

type ReqBatchSend struct {
	BasicFields
	Phones []BatchSend `json:"phones" validate:"required,min=1"`
}
