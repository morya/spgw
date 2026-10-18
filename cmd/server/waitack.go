package server

import (
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/spgwpb"
)

type WaitAck struct {
	Seq       uint32
	IsHandled uint32
	Data      []byte // 序列化原始消息的原因，主要是为了保持原消息的 msgID, seqId 等信息，再次取出，不需要恢复现场
}

type WaitRpt struct {
	MsgID     uint64
	IsHandled uint32
	Data      []byte
}

func (a *App) AddWaitAck(m *spgwpb.Msg) *WaitAck {
	if a.waitAck.Count() > gconfig.Cfg.Server.MaxWaitAck {
		glog.RunS().Errorf("waitAck max %d", gconfig.Cfg.Server.MaxWaitAck)
		return nil
	}

	data, err := proto.Marshal(m)
	if err != nil {
		glog.RunS().Errorf("marshal msg failed: %v", err)
		return nil
	}

	newSeq := <-a.SeqId
	wack := &WaitAck{
		Seq:       newSeq,
		IsHandled: 0,
		Data:      data,
	}

	// 先加换成，再设置seqId
	a.waitAck.Add(newSeq, gconfig.Cfg.Server.WaitAckTimeout.Duration, wack)
	m.Head.SeqId = newSeq

	size := a.waitAck.Count()
	glog.RunS().Debugf("waitAck curr size: %d", size)
	return wack
}

// 主动删除
func (a *App) DeleteWaitAck(wack *WaitAck) *spgwpb.Msg {
	atomic.StoreUint32(&wack.IsHandled, 1)
	glog.RunS().With("seqId", wack.Seq).Debugf("deleting waitAck")

	item, err := a.waitAck.Delete(wack.Seq)
	if err != nil {
		glog.RunS().With(zap.Any("err", err)).Errorf("delete waitAck failed")
		return nil
	}
	// 删除消息，需要恢复 seqId
	msg := &spgwpb.Msg{}
	v := item.Data().(*WaitAck)
	err = proto.Unmarshal(v.Data, msg)
	if err != nil {
		glog.RunS().With(zap.Any("err", err)).Errorf("unmarshal msg failed")
		return nil
	}

	size := a.waitAck.Count()
	glog.RunS().Debugf("waitAck curr size: %d", size)
	return msg
}

func (a *App) GetWaitAckAndDel(seqid uint32) (*spgwpb.Msg, error) {
	item, err := a.waitAck.Value(seqid)
	if err != nil {
		return nil, err
	}

	wack := item.Data().(*WaitAck)
	atomic.StoreUint32(&wack.IsHandled, 1)
	a.waitAck.Delete(seqid)

	msg := &spgwpb.Msg{}
	err = proto.Unmarshal(wack.Data, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// 写入等状态报告实体，可能会因内存满写入失败
func (a *App) AddWaitRpt(m *spgwpb.Msg, logger *zap.SugaredLogger) *zap.SugaredLogger {
	if a.waitRpt.Count() > gconfig.Cfg.Server.MaxWaitRpt {
		glog.RunS().Errorf("waitRpt max %d", gconfig.Cfg.Server.MaxWaitRpt)
		logger = logger.With("msg", "add waitRpt failed, full")
		return logger
	}
	logger = logger.With("msgid", m.SubmitReq.MsgId)
	logger.Debugf("AddWaitRpt %v", m.SubmitReq.MsgId)

	data, err := proto.Marshal(m)
	if err != nil {
		logger = logger.With("msg", "add waitRpt failed, marshal err")
		return logger
	}

	wr := &WaitRpt{
		MsgID:     m.SubmitReq.MsgId,
		IsHandled: 0,
		Data:      data,
	}
	a.waitRpt.Add(m.SubmitReq.MsgId, gconfig.Cfg.Server.WaitRptTimeout.Duration, wr)
	return logger
}

func (a *App) GetWaitRptAndDel(msgId uint64) (*spgwpb.Msg, error) {
	v, err := a.waitRpt.Value(msgId)
	if err != nil {
		return nil, err
	}
	wr := v.Data().(*WaitRpt)
	atomic.StoreUint32(&wr.IsHandled, 1)
	a.waitRpt.Delete(msgId)

	msg := &spgwpb.Msg{}
	err = proto.Unmarshal(wr.Data, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
