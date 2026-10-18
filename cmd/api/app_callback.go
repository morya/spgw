package api

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	adapters "github.com/morya/spgw/internal/httpsend/adapters"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/router/response"
	"github.com/morya/spgw/spgwpb"
)

// callbackRsp 下游回调的响应格式（简洁，不同于 SP API 的 RspBase 格式）
type callbackRsp struct {
	Status string `json:"status"`
}

// OnCallback 接收下游 HTTP callback 的 handler
// POST /api/v1/callback/:vendor
// 签名验证由 signMiddle 中间件统一完成，此 handler 从 context 取已验签的 appId
func (a *App) OnCallback(c *gin.Context) {
	// 从消息内容解析，字段名与下游厂商文档一致
	adapter, err := a.callbackMgr.Get(c.Param("vendor"))
	if err != nil {
		response.Error(c, errors.New("uknown-vendor"))
		return
	}

	logger := zap.S()

	body, err := io.ReadAll(c.Request.Body)
	items, err := adapter.ParseReports(body)
	if err != nil {
		response.Error(c, err)
		return
	}

	for _, item := range items {
		v, ok := a.mapAppidToOAccount.Load(item.AppID)
		if !ok {
			logger.With(zap.String("appId", item.AppID)).Warnf("oAccount not found")
			continue
		}
		oAccount := v.(*models.OAccount)
		m := &spgwpb.Msg{
			Head: &spgwpb.Head{
				CommandId:     spgwpb.CmdID_CmdCallbackReportReq,
				FromAccountId: oAccount.ID,
				RecvTime:      time.Now().UnixMilli(),
			},

			ReportReq: &spgwpb.ReportReq{
				MsgId:          cast.ToUint64(item.SpMsgID),
				Stat:           item.Status,
				DestTerminalId: item.Phone,
				DoneTime:       item.DeliveryTime,
			},
		}

		a.sendToServer(m)
	}

	c.JSON(http.StatusOK, callbackRsp{Status: "0"})
}

// initCallbackAdapters 注册所有回调适配器
func (a *App) initCallbackAdapters() {
	a.callbackMgr.Register(&adapters.ExampleAdapter{})
	// 后续新增的适配器在这里注册
}
