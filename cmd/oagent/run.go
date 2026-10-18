package oagent

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/spgwpb"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (a *App) GetNodeID() uint32 {
	return NodeID
}

func (a *App) GetNodeType() spgwpb.NodeType {
	return spgwpb.NodeType_NodeTypeIAgent
}

// 只回特定发送到 oagent 的系统消息
func (a *App) OnSystemMsg(m *nats.Msg) {
	msg := &spgwpb.Msg{}
	if err := json.Unmarshal(m.Data, msg); err != nil {
		zap.S().Errorf("unmarshal msg error: %v", err)
		return
	}
	zap.S().With(zap.Any("msg", msg)).Infof("recv system msg --------------->")
	a.SendMsg(msg)
}

func (a *App) onMsgFromBus(m *nats.Msg) {
	msg := &spgwpb.Msg{}
	if err := json.Unmarshal(m.Data, msg); err != nil {
		zap.S().Errorf("unmarshal msg error: %v", err)
		return
	}
	a.SendMsg(msg)
}

func (a *App) SendMsg(msg *spgwpb.Msg) {
	select {
	case a.queue <- msg:
	default:
		zap.S().With(zap.Any("head", msg.Head)).Warnf("oagent app queue full, drop msg")
	}
}

func (a *App) run() {
	for m := range a.queue {
		err := a.OnMsg(m)
		if err != nil {
			glog.RunS().Errorf("on msg error: %v", err)
		}
	}
	glog.RunS().Warn("oagent app run exit")
}
