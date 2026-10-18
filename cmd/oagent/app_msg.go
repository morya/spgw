package oagent

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/spgwpb"
)

func (a *App) sendToServer(m *spgwpb.Msg) error {
	data := gostring.JsonEncode(m)
	return a.mq.Publish("spgw.server", data)
}

func (a *App) onSubmitReq(msg *spgwpb.Msg, logger *zap.SugaredLogger) (*zap.SugaredLogger, error) {
	toAccID := msg.Head.ToAccountId
	a.lock.Lock()
	defer a.lock.Unlock()

	logger = logger.With(zap.Any("cmd", msg.Head.CommandId),
		zap.Any("msgID", msg.SubmitReq.MsgId),
		zap.Any("toAccID", toAccID),
	)

	oa, ok := a.mapIdToAccount[toAccID]
	if !ok {
		return logger, fmt.Errorf("unknown ToAccountId: %v", toAccID)
	}
	err := oa.SendMsg(msg)
	if err != nil {
		return logger, err
	}
	return logger, nil
}

func (a *App) onReportAck(msg *spgwpb.Msg, logger *zap.SugaredLogger) (*zap.SugaredLogger, error) {
	toAccID := msg.Head.ToAccountId
	a.lock.Lock()
	defer a.lock.Unlock()

	logger = logger.With(zap.Any("cmd", msg.Head.CommandId),
		zap.Any("toAccID", toAccID),
	)

	oa, ok := a.mapIdToAccount[toAccID]
	if !ok {
		return logger, fmt.Errorf("unknown ToAccountId: %v", toAccID)
	}
	err := oa.SendMsg(msg)
	if err != nil {
		return logger, err
	}
	return logger, nil
}

func (a *App) OnMsg(msg *spgwpb.Msg) error {
	logger := glog.RunS()
	defer func() {
		logger = logger.With(zap.Any("head", msg.Head.String()))
		logger.Debug("oagent -> OnMsg()")
	}()

	switch msg.Head.CommandId {
	case spgwpb.CmdID_CmdSubmitReq:
		var err error
		logger, err = a.onSubmitReq(msg, logger)
		if err != nil {
			return err
		}
	case spgwpb.CmdID_CmdSubmitAck:
		logger = logger.With(zap.Any("submitAck", msg.SubmitAck))
		return a.sendToServer(msg)

	case spgwpb.CmdID_CmdDeliverReq:
		logger = logger.With(zap.Any("deliverReq", msg.DeliverReq))
		return a.sendToServer(msg)

	case spgwpb.CmdID_CmdReportReq:
		logger = logger.With(zap.Any("reportReq", msg.ReportReq))
		return a.sendToServer(msg)

	case spgwpb.CmdID_CmdReportAck, spgwpb.CmdID_CmdDeliverAck:
		logger, _ = a.onReportAck(msg, logger)

	case spgwpb.CmdID_CmdReloadOAccount:
		a.loadAccountsAndLinks()

	case spgwpb.CmdID_CmdReloadOLink:
		a.syncLinksFromDB()

	case spgwpb.CmdID_CmdDeleteOLink:
		a.onDelOLinkByID(msg)

	default:
		logger = logger.With(zap.Any("msg", fmt.Sprintf("unknown msg type: %v", msg.Head.CommandId)))
	}

	return nil
}
