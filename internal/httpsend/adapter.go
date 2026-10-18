package httpsend

import (
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/spgwpb"
)

// SubmitRequest 构建的下游 HTTP 请求
type SubmitRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// SubmitResponse 下游 HTTP 响应解析结果
type SubmitResponse struct {
	Success   bool
	DownMsgID string // 下游平台消息ID
	Code      string // 下游返回码
	Message   string // 下游返回消息
}

// ReportData 回调中的状态报告（示例字段名，实际以对接厂商文档为准）
type ReportData struct {
	Status       string `json:"status"` // DELIVERED/FAILED
	Phone        string `json:"phone"`
	FeeCount     int32  `json:"feeCount"`
	DeliveryMsg  string `json:"deliveryMsg"`
	DeliveryCode string `json:"deliveryCode"`
	DeliveryTime string `json:"deliveryTime"`
	ExtendCode   string `json:"extendCode"`
	MsgID        string `json:"msgId"`
	SpMsgID      string `json:"spMsgId"`
	AppID        string `json:"appId"`
}

// Adapter 下游 HTTP API 适配器接口
type Adapter interface {
	// Vendor 返回供应商标识
	Vendor() string

	// BuildSubmitRequest 将 CMPP SubmitReq 转换为下游 HTTP 请求
	// acc: OAccount 模型，包含 ApiAddress, Appid, ApiKey 等字段
	// req: CMPP SubmitReq
	// spgwMsgID: spgw 生成的 MsgID（字符串形式，用于下游的 spMsgId 字段）
	BuildSubmitRequest(acc *models.OAccount, req *spgwpb.SubmitReq, spgwMsgID uint64) (*SubmitRequest, error)

	// ParseSubmitResponse 解析下游 HTTP 响应
	ParseSubmitResponse(body []byte) (*SubmitResponse, error)

	ParseReports(body []byte) ([]ReportData, error)
}
