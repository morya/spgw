package oagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"go-micro.dev/v5"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/link"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/natsx"
	"github.com/morya/spgw/internal/rconn"
	"github.com/morya/spgw/internal/seq"
	"github.com/morya/spgw/internal/validatex"
	"github.com/morya/spgw/spgwpb"
)

type App struct {
	link.DummyMng

	closeOnce sync.Once

	ctx    context.Context
	cancel context.CancelFunc

	svcCmcenter spgwpb.CmcenterService

	etcdClient *clientv3.Client
	mq         *nats.Conn
	queue      chan *spgwpb.Msg

	lock            sync.RWMutex
	mapIdToAccount  map[uint32]*Account // map id to account
	linkConnecting  sync.Map            // todo: 这个map下一层的 link 暂未维护生命周期，可能在程序退出时出现时序错误
	linkReady       sync.Map
	linkNoReconnect sync.Map // 被syncLinksFromDB主动关闭的link标记，OnLinkBroken据此跳过回写linkConnecting

	// seq
	SeqId    <-chan uint32
	closeSeq chan<- struct{} // 关闭seq
}

func newApp() *App {
	ctx, cancel := context.WithCancel(context.Background())

	// 在 redis 做一个简单的持久化
	start := rconn.GetLastSeqID("oagent", NodeID)
	SeqId, closeSeq := seq.NewSeqIdGenerator(start)

	return &App{
		ctx:    ctx,
		cancel: cancel,

		mapIdToAccount: make(map[uint32]*Account),

		SeqId:    SeqId,
		closeSeq: closeSeq,
		queue:    make(chan *spgwpb.Msg, gconfig.AppQueueCap(gconfig.Cfg.Agent.AppQueueCap)),
	}
}

func (a *App) initMQ() (err error) {
	a.mq, err = natsx.Connect(gconfig.Cfg.Nats)
	if err != nil {
		return err
	}
	lo.Must1(a.mq.Subscribe(consts.TopicOAgentSys, a.OnSystemMsg))
	lo.Must1(a.mq.QueueSubscribe(consts.TopicOAgent, "queue", a.onMsgFromBus))
	return nil
}

func (a *App) initCmcenter() {
	c := micro.NewService(micro.Name("spgw.oagent"))
	a.svcCmcenter = spgwpb.NewCmcenterService("spgw.cmcenter", c.Client())
}

func (a *App) loadAccountsAndLinks() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	var arr []models.OAccount
	err := db.MainDB.Model(&models.OAccount{}).Find(&arr).Error
	if err != nil {
		glog.RunS().With(zap.Any("err", err)).Errorf("load oaccount failed")
		return err
	}

	for _, oa := range arr {
		if oa.Disabled != nil && *oa.Disabled {
			glog.RunS().With(zap.Any("account", oa)).Infof("skip disabled oaccount")
			continue
		}
		acc := newAccount(oa)
		a.mapIdToAccount[acc.ID] = acc
		glog.RunS().With(zap.Any("account", acc)).Infof("loaded oaccount")
	}

	var arrLinks []*models.LinkInfo
	err = db.MainDB.Model(&models.LinkInfo{}).Find(&arrLinks).Error
	if err != nil {
		glog.RunS().With(zap.Any("err", err)).Errorf("load linkinfo failed")
		return err
	}

	count := 0
	for _, item := range arrLinks {
		if item.Disabled != nil && *item.Disabled {
			glog.RunS().With(zap.Any("link", item)).Infof("skip disabled linkinfo")
			continue
		}
		if err := validatex.Validate(item); err != nil {
			glog.RunS().With(zap.Any("err", err)).Errorf("validate linkinfo failed")
			return err
		}

		glog.RunS().With(zap.Any("olink", item)).Infof("loaded linkinfo")

		count++
		a.linkConnecting.Store(item.ID, item)
	}

	glog.RunS().Infof("load count=%d linkinfo", count)

	return nil
}

func (a *App) Init() error {
	err := a.initMQ()
	if err != nil {
		return err
	}

	a.etcdClient = lo.Must(clientv3.New(clientv3.Config{
		Endpoints:        gconfig.Cfg.Etcd.Endpoints,
		DialTimeout:      5 * time.Second,
		AutoSyncInterval: gconfig.Cfg.Etcd.AutoSyncInterval.Duration,
	}))

	a.initCmcenter()
	lo.Must0(a.loadAccountsAndLinks())
	return nil
}

func (a *App) Stop() error {
	a.cancel()

	a.closeOnce.Do(func() {
		a.mq.Close()
		close(a.queue)
		a.etcdClient.Close()
		close(a.closeSeq)

		a.linkReady.Range(func(key, value any) bool {
			l := value.(*link.Link)
			l.Close("app stop")
			return true
		})

		seq := seq.NextSeqId()
		rconn.SaveLastSeqID("oagent", NodeID, seq)
	})
	return nil
}

// 连接永久删除
func (a *App) OnLinkClosed(ctx context.Context, l *link.Link) error {
	zap.S().With(zap.Any("link", l)).Infof("link closed")

	a.linkReady.Delete(l.Info.ID)
	a.linkConnecting.Delete(l.Info.ID)

	a.lock.Lock()
	defer a.lock.Unlock()

	acc, ok := a.mapIdToAccount[l.Info.OAccountID]
	if !ok {
		zap.S().With(zap.Any("link", l)).Errorf("0000 account not found")
		return fmt.Errorf("account not found")
	}
	acc.RemoveLinkByID(l.Info.ID)
	return nil
}

// 连接临时断开
func (a *App) OnLinkBroken(l *link.Link) {
	zap.S().With(zap.Any("link", l)).Infof("link broken")

	a.updateLink2DB(l.Info, false) // TODO: 数据库写入会否比较频繁，后续需要优化
	a.linkReady.Delete(l.Info.ID)

	// 被syncLinksFromDB主动关闭的link：跳过回写linkConnecting，避免用旧info覆盖syncLinksFromDB已设的最终态（删除/新配置）
	if _, skip := a.linkNoReconnect.LoadAndDelete(l.Info.ID); skip {
		a.lock.Lock()
		defer a.lock.Unlock()

		acc, ok := a.mapIdToAccount[l.Info.OAccountID]
		if !ok {
			zap.S().With(zap.Any("link", l)).Errorf("1111 account not found")
			return
		}
		acc.RemoveLinkByID(l.Info.ID)
		return
	}

	a.linkConnecting.Store(l.Info.ID, &l.Info)

	a.lock.Lock()
	defer a.lock.Unlock()

	acc, ok := a.mapIdToAccount[l.Info.OAccountID]
	if !ok {
		zap.S().With(zap.Any("link", l)).Errorf("1111 account not found")
		return
	}
	acc.RemoveLinkByID(l.Info.ID)
}

func (a *App) updateLink2DB(info models.LinkInfo, ready bool) {
	info.Active = &ready
	info.LinkTime = time.Now().Format(time.DateTime)
	db.MainDB.Model(&models.LinkInfo{}).Where("id = ?", info.ID).Update("active", *info.Active)
}

func (a *App) syncLink2Cluster(ctx context.Context, l *link.Link) error {
	k := fmt.Sprintf("/spgw/olink-%v-%v", NodeID, l.Info.ID)
	v := gostring.JsonEncodeString(l.Info)

	a.updateLink2DB(l.Info, true)
	return l.KeepAlive(ctx, a.etcdClient, k, v)
}

// 只有成功的ack才会调用到这里来
func (a *App) OnBindAck(ctx context.Context, l *link.Link, ack *spgwpb.Msg) error {
	// 因为是 link 异步call进来，存在 race condition ，需要加锁保护
	a.lock.Lock()
	defer a.lock.Unlock()

	a.linkConnecting.Delete(l.Info.ID)
	a.linkReady.Store(l.Info.ID, l)

	acc, found := a.mapIdToAccount[l.Info.OAccountID]
	if !found {
		zap.S().With(zap.Any("link", l)).Errorf("3333 account not found")
		return fmt.Errorf("account not found")
	}
	acc.AddLink(l)

	// 2. syncToCluster
	err := a.syncLink2Cluster(ctx, l)
	if err != nil {
		zap.S().With(zap.Any("err", err)).Errorf("sync link2cluster failed")
		return err
	}

	zap.S().With(zap.Any("info", l.Info)).Infof("create client link success")
	return nil
}

func (a *App) keepReconnect() {
	f := func() {
		a.linkConnecting.Range(func(key, value any) bool {
			info := value.(*models.LinkInfo)
			zap.S().With(zap.Any("info", info)).Infof("create client link")

			l, err := link.NewClientLink(*info, a, a.SeqId)
			if err != nil {
				zap.S().With(zap.Any("err", err)).Errorf("new link failed")
				return true
			}
			ctx, cancel := context.WithCancel(a.ctx)
			go l.Run(ctx, cancel)
			return true
		})
	}
	f()

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			zap.S().Warn("keepReconnect ctx done")
			return
		case <-ticker.C:
			f()
		}
	}
}

func (a *App) Start() error {
	defer a.cancel()
	zap.S().Error("oagent start")

	go a.keepReconnect()
	a.run()
	return nil
}
