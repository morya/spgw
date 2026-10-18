package iagent

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/spgwpb"
)

func (a *App) OnMsg(msg *spgwpb.Msg) (err error) {
	logger := glog.RunS()
	defer func() {
		logger = logger.With(zap.Any("head", msg.Head.String()))
		logger.Debug("iagent -> OnMsg()")
	}()

	switch msg.Head.CommandId {
	case spgwpb.CmdID_CmdSubmitReq:
		msg.Head.FromNodeType = spgwpb.NodeType_NodeTypeIAgent
		msg.Head.FromNodeId = NodeID
		msg.Head.Source = spgwpb.MsgSource_MsgSrcTCP
		a.submitReqCount.Add(1)

	case spgwpb.CmdID_CmdDeliverAck, spgwpb.CmdID_CmdReportAck:
		msg.Head.FromNodeId = NodeID
		msg.Head.FromNodeType = spgwpb.NodeType_NodeTypeIAgent
		// 直接返回，不处理
		return nil

	case spgwpb.CmdID_CmdSubmitAck:
		msg.Head.FromNodeType = spgwpb.NodeType_NodeTypeIAgent
		msg.Head.FromNodeId = NodeID
		l := a.findLinkByAccountId(msg.Head.ToAccountId)
		if l == nil {
			logger = logger.With(zap.Any("head", msg.Head), zap.Any("msg", fmt.Sprintf("find link by account failed, account id:%v", msg.Head.ToAccountId)))
			return fmt.Errorf("find link by account failed, account id:%v", msg.Head.ToAccountId)
		}
		logger = logger.With(zap.Any("head", msg.Head), zap.Any("msg", fmt.Sprintf("send msg to link, link id:%v", l.Info.ID)))
		return l.SendMsg(msg)

	case spgwpb.CmdID_CmdDeliverReq, spgwpb.CmdID_CmdReportReq:
		msg.Head.ToNodeType = spgwpb.NodeType_NodeTypeIAgent
		msg.Head.ToNodeId = NodeID
		l := a.findLinkByAccountId(msg.Head.ToAccountId)
		if l == nil {
			logger = logger.With(zap.Any("head", msg.Head), zap.Any("msg", fmt.Sprintf("find link by account failed, account id:%v", msg.Head.ToAccountId)))
			return fmt.Errorf("find link by account failed, account id:%v", msg.Head.ToAccountId)
		}
		return l.SendMsg(msg)

	case spgwpb.CmdID_CmdDeleteILink:
		return a.onDeleteILinkReq(msg, logger)
	}

	err = a.sendToServer(msg)
	if err != nil {
		ack := &spgwpb.Msg{
			Head: &spgwpb.Head{
				CommandId: spgwpb.CmdID_CmdSubmitAck,
				SeqId:     msg.Head.SeqId,
				Status:    1, // 直接用cmpp的错误码，暂不拓展内部错误码表
				RecvTime:  time.Now().Unix(),
			},
		}
		glog.RunS().With(
			zap.Any("ack", ack),
		).Errorf("publish msg to bus failed, err:%v", err)

		// TODO: find link by msg.Head.ToNodeId
		// l.SendMsg(ack)
		return fmt.Errorf("publish msg to bus failed: %w", err)
	}

	return nil
}
