package consts

type ErrCode uint32

const (
	// 注意，cmpp20协议是 uint8, 最大值255
	ErrCodeNoRouteNext ErrCode = 100 + iota
	ErrCodeWaitAckTimeout

	ErrCodeReportNotFound

	ErrCodeHttpSendFailed
	ErrCodeHttpParseFailed
	ErrCodeHttpSubmitFailed
)

var ErrCodeMap = map[ErrCode]string{
	ErrCodeNoRouteNext:    "没有路由",
	ErrCodeWaitAckTimeout: "等待确认超时",

	ErrCodeReportNotFound: "状态报告MsgID不存在",

	ErrCodeHttpSendFailed:   "http发送失败",
	ErrCodeHttpParseFailed:  "http解析失败",
	ErrCodeHttpSubmitFailed: "http提交失败",
}

func (e ErrCode) String() string {
	return ErrCodeMap[e]
}
