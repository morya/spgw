package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	protocol "github.com/hujm2023/go-sms-protocol"

	"github.com/morya/spgw/internal/httpsend"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/spgwpb"
)

// ExampleAdapter 下游通用 HTTP API 的示例适配器。
// 演示典型短信平台对接方式：参数 ASCII 排序拼接 + key 后缀 + MD5 大写签名，
// 提交接口 POST {apiAddress}/api/sms/send，回执由 /api/v1/callback/{vendor} 接收。
// 接入真实厂商时，复制本文件按需调整字段与接口路径即可。
type ExampleAdapter struct{}

func (h *ExampleAdapter) Vendor() string {
	return "example"
}

func (h *ExampleAdapter) buildSignParams(req *spgwpb.SubmitReq, appid string, spgwMsgID uint64) map[string]string {
	params := make(map[string]string)
	params["appid"] = appid
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	spMsgID := fmt.Sprintf("%d", spgwMsgID)
	params["timestamp"] = timestamp
	params["spMsgId"] = spMsgID
	params["signType"] = "md5"
	params["version"] = "1"

	content, err := protocol.DecodeCMPPContentSimple(context.TODO(), uint8(req.MsgFmt), []byte(req.MsgContent))
	if err != nil {
		return nil
	}
	params["content"] = string(content)
	if len(req.DestTerminalId) > 1 {
		// 多个手机号，不编码 phones 字段
	} else {
		params["phone"] = req.DestTerminalId[0]
	}

	return params
}

// BuildSubmitRequest(acc *models.OAccount, req *spgwpb.SubmitReq, spgwMsgID uint64) (*SubmitRequest, error)
func (h *ExampleAdapter) BuildSubmitRequest(acc *models.OAccount, req *spgwpb.SubmitReq, spgwMsgID uint64) (*httpsend.SubmitRequest, error) {
	if len(req.DestTerminalId) == 0 {
		return nil, fmt.Errorf("DestTerminalID is empty")
	}

	params := h.buildSignParams(req, acc.Appid, spgwMsgID)
	if params == nil {
		return nil, fmt.Errorf("build sign params failed")
	}
	// 计算签名
	sign := httpsend.SignMD5(params, acc.ApiKey)

	// 构建请求 body（包含 sign）
	bodyMap := make(map[string]any, len(params)+1)
	for k, v := range params {
		bodyMap[k] = v
	}
	bodyMap["sign"] = sign

	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("json marshal body failed: %w", err)
	}

	return &httpsend.SubmitRequest{
		Method: "POST",
		URL:    acc.ApiAddress + "/api/sms/send",
		Headers: map[string]string{
			"Content-Type": "application/json;charset=utf-8",
		},
		Body: body,
	}, nil
}

// submitResponse 下游响应 JSON 结构
type submitResponse struct {
	Code    string      `json:"code"`
	Success bool        `json:"success"`
	Data    *submitData `json:"data"`
}

type submitData struct {
	MsgID string `json:"msgId"`
}

func (h *ExampleAdapter) ParseSubmitResponse(body []byte) (*httpsend.SubmitResponse, error) {
	var resp submitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse submit response failed: %w", err)
	}

	result := &httpsend.SubmitResponse{
		Success: resp.Success,
		Code:    resp.Code,
	}

	if resp.Data != nil {
		result.DownMsgID = resp.Data.MsgID
	}

	return result, nil
}

func (h *ExampleAdapter) ParseReports(body []byte) ([]httpsend.ReportData, error) {
	var result []httpsend.ReportData
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse reports failed: %w", err)
	}
	return result, nil
}
