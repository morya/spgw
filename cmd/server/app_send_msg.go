package server

import (
	"sync/atomic"
	"time"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/spgwpb"
)

func (a *App) sendToIAgent(mm *spgwpb.Msg) {
	err := a.mq.Publish(consts.TopicIAgent, gostring.JsonEncode(mm))
	if err != nil {
		atomic.AddUint32(&a.NatsQueueIAgentDropCount, 1)
	}
}

func (a *App) sendToOAgent(mm *spgwpb.Msg) {
	err := a.mq.Publish(consts.TopicOAgent, gostring.JsonEncode(mm))
	if err != nil {
		atomic.AddUint32(&a.NatsQueueOAgentDropCount, 1)
	}
}

func (a *App) sendToOAgentByLink(l *OLink, _ *WaitAck, mm *spgwpb.Msg) {
	topic := consts.TopicOAgent
	mm.Head.ToAccountId = l.OAccountID
	err := a.mq.Publish(topic, gostring.JsonEncode(mm))
	if err != nil {
		atomic.AddUint32(&a.NatsQueueOAgentDropCount, 1)
	}
}

func (a *App) buildFailAckAndSendToIAgent(mm *spgwpb.Msg, _ error, code consts.ErrCode) {
	ack := &spgwpb.Msg{
		Head: &spgwpb.Head{
			SeqId:         mm.Head.SeqId,
			FromAccountId: mm.Head.FromAccountId,
			RecvTime:      time.Now().UnixMilli(),
			ToAccountId:   mm.Head.FromAccountId,
		},
		SubmitAck: &spgwpb.SubmitAck{
			MsgId:  mm.SubmitReq.MsgId,
			Result: uint32(code),
		},
	}

	ack.Head.CommandId = spgwpb.CmdID_CmdSubmitAck
	a.sendToIAgent(ack)
}

func (a *App) buildFailRptAndSendToIAgent(mm *spgwpb.Msg, _ error) {
	if mm.SubmitReq == nil {
		return
	}
	submitReq := mm.SubmitReq
	for idx, destID := range submitReq.DestTerminalId {
		ack := &spgwpb.Msg{
			Head: &spgwpb.Head{
				SeqId:         mm.Head.SeqId,
				FromAccountId: mm.Head.FromAccountId,
				RecvTime:      time.Now().UnixMilli(),
				ToAccountId:   mm.Head.FromAccountId,
				Source:        mm.Head.Source,
			},
			ReportReq: &spgwpb.ReportReq{
				MsgId: mm.SubmitReq.LocalMsgId, // 本地消息id
				Stat:  consts.ErrStrExpired,
				DeliverBase: &spgwpb.DeliverBase{
					MsgId:              a.genMsgID(),
					DestId:             destID, // TODO: 这个字段这么填? 待定
					ServiceId:          submitReq.ServiceId,
					TpPid:              submitReq.TpPid,
					TpUdhi:             submitReq.TpUdhi,
					MsgFmt:             submitReq.MsgFmt,
					SrcTerminalId:      submitReq.DestTerminalId[idx],
					SrcTerminalType:    submitReq.DestTerminalType,
					RegisteredDelivery: 1,
				},
			},
		}
		ack.Head.CommandId = spgwpb.CmdID_CmdReportReq

		if ack.Head.Source == spgwpb.MsgSource_MsgSrcTCP {
			a.sendToIAgent(ack)
		} else {
			go a.sendReportToVendor(ack)
		}
	}
}
