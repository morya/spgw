package server

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/seq"
	"github.com/morya/spgw/spgwpb"
)

func (a *App) genMsgID() uint64 {
	n := time.Now()
	seq := seq.NextSeqId()
	// Go 中 uint32 向下转换到 uint16 时，超出 uint16 范围的高 16 位被 直接截断丢弃
	return cmpp.CombineMsgID(uint64(n.Month()), uint64(n.Day()), uint64(n.Hour()), uint64(n.Minute()), uint64(n.Second()), uint64(gconfig.Cfg.Server.SPGwId), uint64(seq))
}

func (a *App) onMsg(m *spgwpb.Msg) {
	logger := glog.RunS()
	defer func() {
		logger = logger.With(
			zap.Any("head", m.Head.String()),
		)
		logger.Debugf("server -> OnMsg()")
	}()

	switch m.Head.CommandId {
	case spgwpb.CmdID_CmdSubmitReq:
		logger = a.onSubmitReq(m, logger)
	case spgwpb.CmdID_CmdSubmitAck:
		logger = a.onSubmitAck(m, logger)

	case spgwpb.CmdID_CmdDeliverReq:
		a.onDeliverReq(m)
	case spgwpb.CmdID_CmdDeliverAck:
		a.onDeliverAck(m)

	case spgwpb.CmdID_CmdCallbackReportReq, spgwpb.CmdID_CmdReportReq:
		logger = a.onReportReq(m, logger)

	case spgwpb.CmdID_CmdReloadALLAccount:
		a.router.Reload(a.etcdClient)

	default:
		logger = logger.With(zap.Any("msg", "unknown cmdID, drop msg"))
		return
	}
}

func (a *App) onSubmitReq(m *spgwpb.Msg, logger *zap.SugaredLogger) *zap.SugaredLogger {
	req := m.SubmitReq

	billRec := &spgwpb.BillRecord{
		BillType:   spgwpb.BillType_Bill60,
		BillTime:   time.Now().UnixMilli(),
		FromNodeId: NodeID,
	}
	defer func() {
		billRec.OriginalMsg = proto.Clone(m).(*spgwpb.Msg)
		a.publishBill(billRec)
	}()

	// 检查并发限制
	if !a.rl.Allow() {
		billRec.ErrCode = consts.ErrCmppCodeMaxFlowErr
		a.router.IncrISubmitFailCnt(m.Head.FromAccountId)
		logger = logger.With(zap.Any("msg", "concurrency limit"))
		a.buildFailAckAndSendToIAgent(m, nil, consts.ErrCmppCodeMaxFlowErr)
		return logger
	}

	// 生成新的 msgid
	m.SubmitReq.MsgId = a.genMsgID()
	m.SubmitReq.LocalMsgId = m.SubmitReq.MsgId
	m.SubmitReq.OrgRegisteredDelivery = req.RegisteredDelivery
	m.SubmitReq.RegisteredDelivery = 1 // 强制南向给report
	logger = logger.With(
		zap.Any("msgID", m.SubmitReq.MsgId),
		zap.Any("localMsgID", m.SubmitReq.LocalMsgId),
	)

	oa, err := a.router.Route(m, req)
	if err != nil {
		billRec.ErrCode = uint32(consts.ErrCodeNoRouteNext)
		a.router.IncrISubmitFailCnt(m.Head.FromAccountId)
		logger = logger.With(zap.Any("msg", fmt.Sprintf("route msg failed, err:%v", err)))
		a.buildFailAckAndSendToIAgent(m, err, consts.ErrCodeNoRouteNext)
		return logger
	}

	if oa.IsHttp != nil && *oa.IsHttp {
		// HTTP 南向路径
		wack := a.AddWaitAck(m)
		go a.sendViaHttp(oa, wack, m, logger)
		return logger
	}

	l := oa.NextLink()
	// CMPP 南向路径
	if l == nil {
		billRec.ErrCode = uint32(consts.ErrCodeNoRouteNext)
		a.router.IncrISubmitFailCnt(m.Head.FromAccountId)
		logger = logger.With(zap.Any("msg", "find route empty link"))
		a.buildFailAckAndSendToIAgent(m, fmt.Errorf("find route empty link"), consts.ErrCodeNoRouteNext)
		return logger
	}
	wack := a.AddWaitAck(m)
	if wack == nil {
		billRec.ErrCode = uint32(consts.ErrCodeNoRouteNext)
		a.router.IncrISubmitFailCnt(m.Head.FromAccountId)
		logger = logger.With(zap.Any("msg", "find route empty link"))
		a.buildFailAckAndSendToIAgent(m, fmt.Errorf("find route empty link"), consts.ErrCodeNoRouteNext)
		return logger
	}
	logger = logger.With(zap.Any("newSeq", wack.Seq))
	go a.sendToOAgentByLink(l, wack, m)

	return logger
}

func (a *App) buildSubmitAck(mm *spgwpb.Msg) *spgwpb.Msg {
	ack := &spgwpb.Msg{
		Head: &spgwpb.Head{},
		SubmitAck: &spgwpb.SubmitAck{
			MsgId: mm.SubmitReq.MsgId,
		},
	}
	copier.Copy(ack.Head, mm.Head)
	ack.Head.CommandId = spgwpb.CmdID_CmdSubmitAck
	return ack
}

func (a *App) onSubmitAck(submitAck *spgwpb.Msg, logger *zap.SugaredLogger) *zap.SugaredLogger {
	downSeq := submitAck.Head.SeqId
	orgSubmit, err := a.GetWaitAckAndDel(submitAck.Head.SeqId)
	if err != nil || orgSubmit == nil {
		logger = logger.With(zap.Any("msg", fmt.Sprintf("get wait ack failed, err:%v", err)))
		return logger
	}
	logger = logger.With(zap.Any("msgid", submitAck.SubmitAck.MsgId))

	// 构造 62 话单：携带完整原始 submitReq（在 orgSubmit 被改写前 Clone）
	a.publishBill(&spgwpb.BillRecord{
		BillType:     spgwpb.BillType_Bill62,
		ErrCode:      submitAck.Head.Status,
		BillTime:     time.Now().UnixMilli(),
		FromNodeId:   NodeID,
		OriginalMsg:  proto.Clone(submitAck).(*spgwpb.Msg),
		OrgSubmitMsg: proto.Clone(orgSubmit).(*spgwpb.Msg),
	})

	iAccID := orgSubmit.Head.FromAccountId
	oAccID := submitAck.Head.FromAccountId

	orgSubmit.SubmitReq.MsgId = submitAck.SubmitAck.MsgId
	orgSubmit.SubmitReq.LocalMsgId = a.genMsgID() // 使用本地msgid给北向回复

	if submitAck.Head.Status != consts.ErrCmppCodeSuccess {
		a.router.IncrISubmitFailCnt(iAccID)
		if oAccID != 0 {
			a.router.IncrOSubmitFailCnt(oAccID)
		}
	} else {
		a.router.IncrISubmitSuccCnt(iAccID)
		if oAccID != 0 {
			a.router.IncrOSubmitSuccCnt(oAccID)
		}
		logger = a.AddWaitRpt(orgSubmit, logger)
	}

	submitAck.Head.SeqId = orgSubmit.Head.SeqId
	submitAck.SubmitAck.MsgId = orgSubmit.SubmitReq.LocalMsgId
	submitAck.Head.ToAccountId = orgSubmit.Head.FromAccountId

	logger = logger.With("downSeq", downSeq, "orgSeq", orgSubmit.Head.SeqId).With("msg", "process submit ack")

	a.sendToIAgent(submitAck)
	return logger
}

func (a *App) onDeliverReq(mm *spgwpb.Msg) {
	// ccc
}

func (a *App) onDeliverAck(mm *spgwpb.Msg) {
	// ccc
}

func (a *App) onReportReq(m *spgwpb.Msg, logger *zap.SugaredLogger) *zap.SugaredLogger {
	logger = logger.With(zap.Any("reportReq", m.ReportReq))

	if m.ReportReq == nil {
		return logger
	}
	reportReq := m.ReportReq
	oAccID := m.Head.FromAccountId

	ack := &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId:   spgwpb.CmdID_CmdReportAck,
			SeqId:       m.Head.SeqId,
			ToAccountId: m.Head.FromAccountId,
		},
		DeliverAck: &spgwpb.DeliverAck{
			MsgId: reportReq.DeliverBase.MsgId,
		},
		ReportAck: &spgwpb.ReportAck{
			MsgId:  reportReq.DeliverBase.MsgId,
			Result: consts.ErrCmppCodeSuccess,
		},
	}

	orgSubmitReq, err := a.GetWaitRptAndDel(reportReq.MsgId)
	if err != nil || orgSubmitReq == nil {
		logger = logger.With(zap.Any("msg", fmt.Sprintf("get wait rpt failed, err:%v", err)))
		ack.ReportAck.Result = uint32(consts.ErrCodeReportNotFound)
		a.router.IncrOSReportFailCnt(oAccID)
		a.sendToOAgent(ack)
		return logger
	}

	// 构造 63 话单：携带完整原始 submitReq（在 m 被改写前 Clone）
	a.publishBill(&spgwpb.BillRecord{
		BillType:     spgwpb.BillType_Bill63,
		ErrCode:      0,
		BillTime:     time.Now().UnixMilli(),
		FromNodeId:   NodeID,
		OriginalMsg:  proto.Clone(m).(*spgwpb.Msg),
		OrgSubmitMsg: proto.Clone(orgSubmitReq).(*spgwpb.Msg),
	})

	a.sendToOAgent(ack)

	if oAccID != 0 {
		switch reportReq.Stat {
		case consts.ErrStrDelivrd:
			a.router.IncrOSReportSuccCnt(oAccID)
			a.router.IncrIReportSuccCnt(orgSubmitReq.Head.GetFromAccountId())
		default:
			a.router.IncrIReportFailCnt(orgSubmitReq.Head.GetFromAccountId())
			a.router.IncrOSReportFailCnt(oAccID)
		}
	}

	logger = logger.With("MsgID", reportReq.MsgId).
		With("orgmsg-seq", orgSubmitReq.Head.SeqId).
		With("stat", reportReq.Stat).
		With("destTerminalId", reportReq.DestTerminalId).
		With("msg", "found origin msg")

	m.Head.SeqId = seq.NextSeqId()
	m.Head.ToAccountId = orgSubmitReq.Head.FromAccountId

	m.ReportReq.DeliverBase.MsgId = a.genMsgID()
	m.ReportReq.MsgId = orgSubmitReq.SubmitReq.LocalMsgId

	sendToIAgent := true
	if orgSubmitReq.SubmitReq.OrgRegisteredDelivery == 0 {
		sendToIAgent = false
	} else if orgSubmitReq.SubmitReq.OrgRegisteredDelivery == 2 && reportReq.Stat == consts.ErrStrDelivrd {
		sendToIAgent = false
	}

	if !sendToIAgent {
		return logger
	}
	switch orgSubmitReq.Head.Source {
	case spgwpb.MsgSource_MsgSrcHTTP:
		a.sendReportViaHTTP(m, orgSubmitReq, logger)
	default:
		a.sendToIAgent(m)
	}

	return logger
}
