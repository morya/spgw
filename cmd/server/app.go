package server

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/muesli/cache2go"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/counter"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/httpsend"
	"github.com/morya/spgw/internal/httpsend/adapters"
	"github.com/morya/spgw/internal/natsx"
	"github.com/morya/spgw/internal/rconn"
	"github.com/morya/spgw/internal/seq"
	"github.com/morya/spgw/spgwpb"
)

type App struct {
	mq     *nats.Conn
	ctx    context.Context
	cancel context.CancelFunc

	etcdClient *clientv3.Client
	router     *Router

	rl    *rate.Limiter
	queue chan *spgwpb.Msg

	waitAck *cache2go.CacheTable
	waitRpt *cache2go.CacheTable

	httpMgr *httpsend.Manager

	SeqId    <-chan uint32
	closeSeq chan<- struct{}

	AppQueueDropCount        uint32
	NatsQueueIAgentDropCount uint32
	NatsQueueOAgentDropCount uint32
}

func newApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	start := rconn.GetLastSeqID("server", NodeID)
	SeqId, closeSeq := seq.NewSeqIdGenerator(start)

	a := &App{
		ctx:    ctx,
		cancel: cancel,

		waitAck: cache2go.Cache("waitAck"),
		waitRpt: cache2go.Cache("waitRpt"),

		rl:    rate.NewLimiter(rate.Limit(gconfig.Cfg.Server.MaxConcurrency), gconfig.Cfg.Server.Burst),
		queue: make(chan *spgwpb.Msg, gconfig.ServerAppQueueCap(gconfig.Cfg.Server.AppQueueCap)),

		httpMgr: httpsend.NewManager(),

		router: newRouter(),

		SeqId:    SeqId,
		closeSeq: closeSeq,
	}

	// waitAck 超时回调：30min 未收到下游应答则标记失败
	a.waitAck.SetAboutToDeleteItemCallback(func(e *cache2go.CacheItem) {
		if wack, ok := e.Data().(*WaitAck); ok {
			if atomic.LoadUint32(&wack.IsHandled) == 1 {
				return
			}

			data := wack.Data
			var mm spgwpb.Msg
			err := proto.Unmarshal(data, &mm)
			if err != nil {
				glog.RunS().With(zap.Any("err", err)).Errorf("unmarshal msg failed")
				return
			}

			a.buildFailAckAndSendToIAgent(&mm, nil, consts.ErrCodeWaitAckTimeout)
		}
	})

	// waitRpt 超时回调：30min 未收到 report 则记录日志
	a.waitRpt.SetAboutToDeleteItemCallback(func(e *cache2go.CacheItem) {
		if w, ok := e.Data().(*WaitRpt); ok {
			if atomic.LoadUint32(&w.IsHandled) == 1 {
				return
			}

			mm := &spgwpb.Msg{}
			err := proto.Unmarshal(w.Data, mm)
			if err != nil {
				glog.RunS().With(zap.Any("err", err)).Errorf("unmarshal msg failed")
				return
			}

			a.buildFailRptAndSendToIAgent(mm, nil)
			return
		}
	})

	return a
}

func (a *App) Init() error {
	counter.SetNodeID(NodeID)

	a.mq = natsx.MustConnect(gconfig.Cfg.Nats)

	a.etcdClient = lo.Must(clientv3.New(clientv3.Config{
		Endpoints:        gconfig.Cfg.Etcd.Endpoints,
		DialTimeout:      5 * time.Second,
		AutoSyncInterval: gconfig.Cfg.Etcd.AutoSyncInterval.Duration,
	}))

	if err := a.router.Reload(a.etcdClient); err != nil {
		return err
	}

	// 注册 HTTP 适配器
	a.httpMgr.Register(&adapters.ExampleAdapter{})

	return nil
}

func (a *App) Watch(ctx context.Context, c *clientv3.Client) error {
	w := c.Watch(ctx, "/spgw", clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case resp := <-w:
			for _, ev := range resp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					glog.RunS().Infof("watch put event %s", ev.Kv.Key)
					m := spgwpb.NewMsg(spgwpb.CmdID_CmdReloadALLAccount, 0)
					a.sendMsg(m)

				case clientv3.EventTypeDelete:
					glog.RunS().Infof("watch delete event %s", ev.Kv.Key)
					m := spgwpb.NewMsg(spgwpb.CmdID_CmdReloadALLAccount, 0)
					a.sendMsg(m)

				default:
					continue
				}
			}
		}
	}
}

func (a *App) Run() error {
	defer a.cancel()
	go a.Watch(a.ctx, a.etcdClient)
	go a.run()
	go a.flushStatLoop()

	lo.Must(a.mq.Subscribe("spgw.server", a.onMsgFromMQ))
	defer a.mq.Close()

	zap.S().Infof("app running...")
	<-a.ctx.Done()

	close(a.queue)
	counter.FlushToDB()
	zap.S().Infof("app exit")
	return nil
}

func (a *App) printDropCount() {
	var appDrop uint32
	var iagentDrop uint32
	var oagentDrop uint32
	appDrop = atomic.SwapUint32(&a.AppQueueDropCount, 0)
	iagentDrop = atomic.SwapUint32(&a.NatsQueueIAgentDropCount, 0)
	oagentDrop = atomic.SwapUint32(&a.NatsQueueOAgentDropCount, 0)

	glog.StatS().With(
		zap.Uint32("appDrop", appDrop),
		zap.Uint32("iagentDrop", iagentDrop),
		zap.Uint32("oagentDrop", oagentDrop),
	).Infof("drop-counter")
}

func (a *App) flushStatLoop() {
	ticker := time.NewTicker(gconfig.Cfg.Server.StatFlushInterval.Duration)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			lo.Try0(a.printDropCount)

			if err := counter.FlushToDB(); err != nil {
				glog.RunS().With(zap.Any("err", err)).Errorf("flush stat to db failed")
			}
		}
	}
}

func (a *App) Stop() {
	a.cancel()

	seq := <-a.SeqId
	rconn.SaveLastSeqID("server", NodeID, seq)
}

func (a *App) sendMsg(m *spgwpb.Msg) {
	select {
	case a.queue <- m:
	default:
		atomic.AddUint32(&a.AppQueueDropCount, 1)
		glog.RunS().With(zap.Any("head", m.Head)).Warnf("server app queue full, drop msg")
	}
}

func (a *App) run() {
	for msg := range a.queue {
		a.onMsg(msg)
	}
}

func (a *App) onMsgFromMQ(msg *nats.Msg) {
	m := &spgwpb.Msg{}
	if err := json.Unmarshal(msg.Data, m); err != nil {
		zap.S().Errorf("unmarshal msg error: %v", err)
		return
	}

	m.Head.LastRetryTsMilli = uint64(time.Now().UnixMilli())
	m.Head.RetryCount++
	if m.Head.RetryCount > 3 {
		zap.S().With(zap.Any("head", m.Head.String())).Errorf("[onMsgFromMQ] retry count > 3")
	}

	a.sendMsg(m)
}
