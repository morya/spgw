package iagent

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/link"
	"github.com/morya/spgw/spgwpb"
)

func (a *App) GetNodeID() uint32 {
	return NodeID
}

func (a *App) GetNodeType() spgwpb.NodeType {
	return spgwpb.NodeType_NodeTypeIAgent
}

func (a *App) sendToServer(m *spgwpb.Msg) error {
	data := gostring.JsonEncode(m)
	return a.mq.Publish("spgw.server", data)
}

// OnSystemMsg 系统指令消息
func (a *App) OnSystemMsg(m *nats.Msg) {
	msg := &spgwpb.Msg{}
	if err := json.Unmarshal(m.Data, msg); err != nil {
		glog.RunS().Errorf("unmarshal msg error: %v", err)
		return
	}
	glog.RunS().With(zap.Any("msg", msg)).Debugf("recv system msg")
	a.SendMsg(msg)
}

// 业务广播消息，特定到当前节点的
func (a *App) onMsgFromBus(m *nats.Msg) {
	msg := &spgwpb.Msg{}
	if err := json.Unmarshal(m.Data, msg); err != nil {
		glog.RunS().Errorf("unmarshal msg error: %v", err)
		return
	}
	a.SendMsg(msg)
}

func (a *App) SendMsg(msg *spgwpb.Msg) {
	select {
	case a.queue <- msg:
	default:
		atomic.AddUint32(&a.AppQueueDropCount, 1)
		glog.RunS().With(zap.Any("head", msg.Head)).Warnf("iagent app queue full, drop msg")
	}
}

func (a *App) run() {
	for m := range a.queue {
		err := a.OnMsg(m)
		if err != nil {
			if errors.Is(err, link.ErrQueueFull) || errors.Is(err, link.ErrLinkClosed) {
				atomic.AddUint32(&a.LinkQueueDropCount, 1)
			}
			glog.RunS().Errorf("on msg error: %v", err)
		}

	}
	glog.RunS().Infof("iagent app run exit")
}

func (a *App) Run() {
	a.run()
}

func (a *App) findLinkByAccountId(accountId uint32) *link.Link {
	a.lock.Lock()
	defer a.lock.Unlock()
	acc, ok := a.mapId2Account[accountId]
	if !ok {
		return nil
	}
	return acc.NextLink()
}

func (a *App) statCounter(ctx context.Context) {
	const statIntervalSec = 10
	ticker := time.NewTicker(time.Second * statIntervalSec)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			var (
				appQueueDrop  uint32
				linkQueueDrop uint32
			)
			appQueueDrop = atomic.SwapUint32(&a.AppQueueDropCount, 0)
			linkQueueDrop = atomic.SwapUint32(&a.LinkQueueDropCount, 0)
			submitReqTotal := a.submitReqCount.Swap(0)
			submitReqSpeed := float64(submitReqTotal) / float64(statIntervalSec)

			glog.RunS().With(
				zap.Any("appQueueDrop", appQueueDrop),
				zap.Any("linkQueueDrop", linkQueueDrop),
				zap.Uint32("submitReqTotal", submitReqTotal),
				zap.Float64("submitReqSpeed", submitReqSpeed),
			).Infof("iagent stat counter")
		}
	}
}
