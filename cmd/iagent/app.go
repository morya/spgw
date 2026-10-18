package iagent

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"go-micro.dev/v5"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/link"
	"github.com/morya/spgw/internal/linkid"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/natsx"
	"github.com/morya/spgw/internal/rconn"
	"github.com/morya/spgw/internal/seq"
	"github.com/morya/spgw/internal/util/ipx"
	"github.com/morya/spgw/spgwpb"
)

type App struct {
	link.DummyMng

	ctx    context.Context
	cancel context.CancelFunc

	closeOnce  sync.Once
	etcdClient *clientv3.Client

	svcCmcenter spgwpb.CmcenterService
	mq          *nats.Conn

	rl    *rate.Limiter
	queue chan *spgwpb.Msg

	lock          sync.Mutex
	mapId2Account map[uint32]*Account
	mapId2Link    sync.Map

	SeqId    <-chan uint32
	closeSeq chan<- struct{}

	AppQueueDropCount  uint32
	LinkQueueDropCount uint32

	submitReqCount atomic.Uint32
}

var _ link.AppMng = (*App)(nil)

func newApp() *App {
	ctx, cancel := context.WithCancel(context.Background())

	// TODO: 也可以从db/etcd等其他配置区域读取
	start := rconn.GetLastSeqID("iagent", NodeID)
	SeqId, closeSeq := seq.NewSeqIdGenerator(start)

	app := &App{
		ctx:    ctx,
		cancel: cancel,

		rl:    rate.NewLimiter(rate.Limit(gconfig.Cfg.Agent.MaxConcurrency), gconfig.Cfg.Agent.Burst),
		queue: make(chan *spgwpb.Msg, gconfig.AppQueueCap(gconfig.Cfg.Agent.AppQueueCap)),

		mapId2Account: make(map[uint32]*Account),

		SeqId:    SeqId,
		closeSeq: closeSeq,
	}

	c := micro.NewService(micro.Name("spgw.iagent"))
	app.svcCmcenter = spgwpb.NewCmcenterService("spgw.cmcenter", c.Client())

	return app
}

func (a *App) loadIAccounts() error {
	var arr []*models.SpInfo
	err := db.MainDB.Find(&arr).Error
	if err != nil {
		return fmt.Errorf("load iaccounts from db error: %v", err)
	}

	mapId2Account := make(map[uint32]*Account)

	for _, v := range arr {
		if v.Disabled != nil && *v.Disabled {
			glog.RunS().With(zap.Any("account", v)).Infof("skip disabled iaccount")
			continue
		}
		acc := NewAccount(uint32(v.ID), v.Spid)
		mapId2Account[uint32(v.ID)] = acc
	}

	a.lock.Lock()
	defer a.lock.Unlock()
	a.mapId2Account = mapId2Account

	return nil
}

func (a *App) initMQ() (err error) {
	a.mq, err = natsx.Connect(gconfig.Cfg.Nats)
	if err != nil {
		return err
	}
	lo.Must1(a.mq.QueueSubscribe("spgw.iagent-sys", "queue", a.OnSystemMsg))
	lo.Must1(a.mq.Subscribe("spgw.iagent", a.onMsgFromBus))
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

	err = a.loadIAccounts()
	if err != nil {
		return fmt.Errorf("load iaccounts from db failed: %v", err)
	}

	return nil
}

func (a *App) Stop() {
	a.cancel()

	a.closeOnce.Do(func() {
		a.etcdClient.Close()
		close(a.closeSeq)
		close(a.queue)
		a.mq.Close()

		a.mapId2Link.Range(func(key, value any) bool {
			l := value.(*link.Link)
			l.Close("app stop")
			return true
		})
	})
}

func (a *App) OnLinkClosed(ctx context.Context, l *link.Link) error {
	acc := a.FindOrCreateAccount(l.Info.IAccountID, l.Info.Spid)
	if acc != nil {
		acc.RemoveLinkByID(l.Info.ID)
	}

	a.mapId2Link.Delete(l.Info.ID)

	linkid.Release(int(l.Info.ID))
	glog.RunS().Debugw("link closed", "link", l.Info)
	return nil
}

func (a *App) syncLink2Cluster(ctx context.Context, l *link.Link, info *models.LinkInfo) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	l.Info.Active = new(true)
	l.Info.LinkTime = time.Now().Format(time.DateTime)
	db.MainDB.Model(&models.LinkInfo{}).Where("id = ?", l.Info.ID).Updates(l.Info.Active)

	k := fmt.Sprintf("/spgw/ilink-%v-%v", NodeID, l.Info.ID)
	v := gostring.JsonEncodeString(info)
	return l.KeepAlive(ctx, a.etcdClient, k, v)
}

// FindOrCreateAccount 在内存中查找或创建账号, 不改db
func (a *App) FindOrCreateAccount(accountId uint32, spid string) *Account {
	a.lock.Lock()
	defer a.lock.Unlock()

	if accountId == 0 {
		return nil
	}

	value, ok := a.mapId2Account[accountId]
	if ok {
		return value
	}
	acc := NewAccount(accountId, spid)
	a.mapId2Account[accountId] = acc
	return acc
}

func (a *App) OnBindReq(ctx context.Context, l *link.Link, msg *spgwpb.Msg) *spgwpb.Msg {
	ack := &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdBindAck,
			SeqId:     msg.GetHead().SeqId,
			RecvTime:  time.Now().UnixMilli(),
		},
		BindAccountRsp: &spgwpb.BindAccountRsp{
			AuthenticatorSource: msg.BindAccountReq.AuthenticatorSource,
		},
	}
	logger := glog.BinS().With("remote", l.Conn.RemoteAddr().String())
	defer func() {
		logger.With(zap.Any("msg", ack)).Infof("bind ack")
	}()

	req := msg.GetBindAccountReq()
	if req == nil {
		logger = logger.With(zap.Any("msg", "bind req is nil"))
		ack.Head.Status = consts.ErrCmppCodeMsgStruct
		return ack
	}

	if req.Spid == "" || len(req.AuthenticatorSource) == 0 || req.Timestamp == 0 {
		logger = logger.With(zap.Any("msg", "bind req missing spid, authSource or timestamp"))
		ack.Head.Status = consts.ErrCmppCodeMsgStruct
		return ack
	}

	dbAcc := &models.SpInfo{}
	err := db.MainDB.Where("spid = ?", req.Spid).First(dbAcc).Error
	if err != nil {
		logger = logger.With(zap.Any("msg", fmt.Errorf("find iaccount by spid failed: %v", err)))
		ack.Head.Status = consts.ErrCmppCodeAuthErr
		return ack
	}
	if dbAcc.Disabled != nil && *dbAcc.Disabled {
		logger = logger.With(zap.Any("msg", fmt.Errorf("iaccount %s is disabled", dbAcc.Spid)))
		ack.Head.Status = consts.ErrCmppCodeInvalidSrc
		return ack
	}

	ts := cmpp.TimeStamp2Str(req.Timestamp)
	auth := cmpp.GenConnectAuth(dbAcc.Spid, dbAcc.Pswd, ts)
	orgAuth := []byte(req.AuthenticatorSource)
	if !bytes.Equal(auth, orgAuth) {
		logger = logger.With(
			"link", l.Info,
			"spid", req.Spid,
			"remote_ip", l.Conn.RemoteAddr().String(),
			"msg", fmt.Sprintf("bind req auth not match, recv=%s, should=%s", hex.EncodeToString(orgAuth), hex.EncodeToString(auth)))
		ack.Head.Status = consts.ErrCmppCodeAuthErr
		return ack
	}

	remoteIp := ipx.ParseToIpString(l.Conn.RemoteAddr())
	if remoteIp == "" {
		logger = logger.With(zap.Any("msg", fmt.Errorf("parse remote ip failed")))
		ack.Head.Status = consts.ErrCmppCodeSpIpErr
		return ack
	}
	if len(dbAcc.IpWhitelist) > 0 && !dbAcc.IpWhitelist.Contains(remoteIp) {
		logger = logger.With(
			"link", l.Info,
			"spid", req.Spid,
			"remote_ip", l.Conn.RemoteAddr().String(),
			"msg", fmt.Sprintf("bind req ip not in whitelist, ip=%s", l.Conn.RemoteAddr().String()))
		ack.Head.Status = consts.ErrCmppCodeSpIpErr
		return ack
	}

	info := &models.LinkInfo{
		ID:            l.Info.ID,
		IAccountID:    uint32(dbAcc.ID),
		INodeID:       NodeID,
		Spid:          dbAcc.Spid,
		Pswd:          dbAcc.Pswd,
		Proto:         l.Info.Proto,
		RemoteAddress: l.Conn.RemoteAddr().String(),
		LocalAddress:  l.Conn.LocalAddr().String(),
	}
	l.Info = *info

	if true {
		acc := a.FindOrCreateAccount(uint32(dbAcc.ID), dbAcc.Spid)
		if acc.LinkCount() >= int(dbAcc.MaxConn) {
			logger = logger.With(
				zap.Any("iaccount", dbAcc),
				zap.Any("linkinfo", info),
				zap.Any("msg", "max conn limit"),
			)
			ack.Head.Status = consts.ErrCmppCodeMaxConnErr
			return ack
		}
		acc.AddLink(l)
		a.mapId2Link.Store(l.Info.ID, l)

		// 同步信息到集群内部
		err = a.syncLink2Cluster(ctx, l, info)
		if err != nil {
			acc.RemoveLinkByID(l.Info.ID)
			a.mapId2Link.Delete(l.Info.ID)

			glog.RunS().Errorf("sync link to cluster failed: %v", err)
			ack.Head.Status = consts.ErrCmppCodeGatewayErr
			return ack
		}

		logger = logger.With(
			zap.Any("iaccount", dbAcc),
			zap.Any("linkinfo", info),
			zap.Any("msg", "bind req success"),
		)
	}

	if dbAcc.MaxSpeed > 0 {
		if dbAcc.Burst == 0 {
			dbAcc.Burst = 500
		}
		l.SetRateLimiter(rate.NewLimiter(rate.Limit(float64(dbAcc.MaxSpeed)), dbAcc.Burst))
	}
	ack.BindAccountRsp.Spid = dbAcc.Spid
	ack.BindAccountRsp.Password = dbAcc.Pswd
	return ack
}

func (a *App) OnBindAck(ctx context.Context, l *link.Link, m *spgwpb.Msg) error {
	glog.RunS().With(
		zap.Any("link", l),
		zap.Any("bindAck", gostring.JsonEncodeString(m)),
	).Debugf("recv bind ack")

	ack := m.GetBindAccountRsp()
	if ack == nil {
		return fmt.Errorf("bind ack is nil")
	}

	if m.Head.Status != consts.ErrCmppCodeSuccess {
		return fmt.Errorf("bind ack status not success")
	}

	return nil
}

func (a *App) handleConn(ctx context.Context, conn net.Conn, proto string) {
	glog.RunS().Debugf("cmpp server onAccept [remote:%s] proto=%s", conn.RemoteAddr().String(), proto)

	lid := linkid.Get()
	if lid == 0 {
		glog.RunS().Errorf("no linkid left, closing conn %s", conn.RemoteAddr().String())
		return
	}

	l, err := link.NewServerLink(
		models.LinkInfo{
			ID:    uint32(lid),
			Proto: proto,
		},
		conn,
		a, a.SeqId)
	if err != nil {
		glog.RunS().Errorf("new server link error: %v", err)
		return
	}

	linkCtx, cancel := context.WithCancel(ctx)
	l.Run(linkCtx, cancel)

	a.OnLinkClosed(linkCtx, l)
	glog.RunS().With("remote", conn.RemoteAddr().String()).With(zap.Any("linkid", lid)).Infof("[onDisConnect] remote address")
}
