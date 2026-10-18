package server

import (
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/httpsend"
	"github.com/morya/spgw/spgwpb"
)

// http 流程
func (a *App) sendReportToVendor(rpt *spgwpb.Msg) error {
	return nil
}

// sendViaHttp 通过 HTTP 协议发送短信到下游
// 跑在 go routine 中，需要确保访问的函数都是 thread-safe 的
func (a *App) sendViaHttp(oa *OAccount, wack *WaitAck, m *spgwpb.Msg, logger *zap.SugaredLogger) *zap.SugaredLogger {
	logger = logger.With(zap.Uint64("msgID", m.SubmitReq.MsgId))

	iAccID := m.Head.FromAccountId
	oAccID := oa.ID

	failCnt := func() {
		a.router.IncrISubmitFailCnt(iAccID)
		a.router.IncrOSubmitFailCnt(oAccID)
	}

	adapter, err := a.httpMgr.Get(oa.HttpVendor)
	if err != nil {
		logger.Warnf("find http adapter failed: %v", err)
		msg := a.DeleteWaitAck(wack)
		failCnt()
		a.buildFailAckAndSendToIAgent(msg, err, consts.ErrCodeNoRouteNext)
		return logger
	}

	httpReq, err := adapter.BuildSubmitRequest(&oa.OAccount, m.SubmitReq, m.SubmitReq.MsgId)
	if err != nil {
		logger.Warnf("build http request failed: %v", err)
		msg := a.DeleteWaitAck(wack)
		failCnt()
		a.buildFailAckAndSendToIAgent(msg, err, consts.ErrCodeHttpSendFailed)
		return logger
	}

	respBody, err := httpsend.Send(httpReq)
	if err != nil {
		logger.Warnf("http send failed: %v", err)
		msg := a.DeleteWaitAck(wack)
		failCnt()
		a.buildFailAckAndSendToIAgent(msg, err, consts.ErrCodeHttpSendFailed)
		return logger
	}

	submitResp, err := adapter.ParseSubmitResponse(respBody)
	if err != nil {
		logger.Warnf("parse http response failed: %v", err)
		msg := a.DeleteWaitAck(wack)
		failCnt()
		a.buildFailAckAndSendToIAgent(msg, err, consts.ErrCodeHttpParseFailed)
		return logger
	}

	if !submitResp.Success {
		logger.With(zap.Any("resp", submitResp)).Warnf("downstream submit failed")
		msg := a.DeleteWaitAck(wack)
		failCnt()
		a.buildFailAckAndSendToIAgent(msg, err, consts.ErrCodeHttpSubmitFailed)
		return logger
	}

	a.router.IncrISubmitSuccCnt(iAccID)
	a.router.IncrOSubmitSuccCnt(oAccID)

	logger = a.AddWaitRpt(m, logger)

	ack := a.buildSubmitAck(m)
	ack.Head.ToAccountId = m.Head.FromAccountId
	ack.SubmitAck.Result = 0
	a.sendToIAgent(ack)

	a.DeleteWaitAck(wack)

	logger.With(
		zap.String("downMsgID", submitResp.DownMsgID),
		zap.Any("msg", "http submit success"))
	return logger
}
