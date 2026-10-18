package models

type BasicFields struct {
	AppID     string `json:"appId" validate:"required"`
	Timestamp int64  `json:"timestamp" validate:"required"`
	Sign      string `json:"sign" validate:"required"`
	Version   string `json:"version"`
	SignType  string `json:"signType" validate:"required,oneof=md5"`
	Content   string `json:"content" validate:"required"`
	ExtendNo  string `json:"extendNo"`
}

/*
	{
	    "appId":"",
	    "timestamp": 1694502400000,
	    "sign":"",
	    "version":"",
	    "signType":"",
	    "phone":"",
	    "content":"",
	    "spMsgId":"",
	    "extendNo":""
	}
*/
type ReqSingleSend struct {
	BasicFields

	SpMsgId string `json:"spMsgId"`
	Phone   string `json:"phone" validate:"required"`
}

type BatchSend struct {
	Phone   string `json:"phone" validate:"required"`
	SpMsgId string `json:"spMsgId"`
}

/*
	 {
	    "appId":"",
	    "timestamp": 1694502400000,
	    "sign":"",
	    "version":"",
	    "signType":"",
	    "phones":[{
	        "phone":"",
	        "spMsgId":""
	    },{
	        "phone":"",
	        "spMsgId":""
	    }],
	    "content":"",
	    "extendNo":""
	}
*/
type ReqBatchSend struct {
	BasicFields
	Phones []BatchSend `json:"phones" validate:"max=99"`
}
