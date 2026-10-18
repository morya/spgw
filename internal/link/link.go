package link

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hujm2023/go-sms-protocol/codec"
	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/spgwpb"
)

// MaxHeartbeatLost 连续心跳丢失次数达到此值时强制断开连接
const MaxHeartbeatLost = 3

type Link struct {
	Info  models.LinkInfo
	Mode  Mode // 链接模式 // client or server
	State consts.State

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	loginOK chan bool
	myCodec MyCodec

	rl    *rate.Limiter
	queue chan *spgwpb.Msg
	Conn  net.Conn

	seqId              <-chan uint32
	mng                AppMng
	heartbeatLostCount int // 连续心跳丢失计数
}

type Option func(l *Link)

func newLink(info models.LinkInfo, mode Mode, p AppMng, seqId <-chan uint32) (*Link, error) {
	mc, err := NewMyCodec(info.Proto)
	if err != nil {
		return nil, err
	}
	l := &Link{
		Info: info,
		Mode: mode,

		loginOK: make(chan bool, 1),

		rl:      rate.NewLimiter(rate.Limit(gconfig.Cfg.Agent.MaxConcurrency), gconfig.Cfg.Agent.Burst),
		State:   consts.StateConnecting,
		queue:   make(chan *spgwpb.Msg, gconfig.LinkQueueCap(gconfig.Cfg.Agent.LinkQueueCap)),
		mng:     p,
		seqId:   seqId,
		myCodec: mc,
	}
	return l, nil
}

func NewClientLink(info models.LinkInfo, p AppMng, seqId <-chan uint32) (*Link, error) {
	link, err := newLink(info, ModeClient, p, seqId)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func NewServerLink(info models.LinkInfo, conn net.Conn, p AppMng, seqId <-chan uint32) (*Link, error) {
	link, err := newLink(info, ModeServer, p, seqId)
	if err != nil {
		return nil, err
	}
	link.Conn = conn
	link.State = consts.StateConnected
	return link, nil
}

func (l *Link) SetRateLimiter(rl *rate.Limiter) {
	l.rl = rl
}

func (l *Link) SendMsg(msg *spgwpb.Msg) (err error) {
	ok := lo.Try(func() error {
		select {
		case l.queue <- msg:
			return nil
		default:
			return ErrQueueFull
		}
	})
	if !ok {
		return ErrLinkClosed
	}
	return nil
}

func (l *Link) KeepAlive(ctx context.Context, etcdClient *clientv3.Client, k, v string) error {
	if l.Conn == nil {
		return fmt.Errorf("conn not initialized")
	}
	ll, err := etcdClient.Grant(ctx, 5)
	if err != nil {
		glog.RunS().Errorf("grant lease error: %v", err)
		return err
	}

	lid := ll.ID
	go func() {
		defer l.cancel()

		defer func() {
			etcdClient.Revoke(context.TODO(), lid)
		}()

		tt := time.NewTicker(time.Second * 3)
		defer tt.Stop()
		for {
			select {
			case <-ctx.Done():
				glog.RunS().Debugf("link %d, leaseID %d etcd keep alive ctx done", l.Info.ID, lid)
				return

			case <-tt.C:
				_, err = etcdClient.KeepAliveOnce(ctx, lid)
				if err != nil {
					glog.RunS().Errorf("keep alive error: %v", err)
					return
				}
			}
		}
	}()

	_, err = etcdClient.Put(l.ctx, k, v, clientv3.WithLease(lid))
	if err != nil {
		glog.RunS().Errorf("put link info error: %v", err)
		return err
	}
	return nil
}

func (l *Link) heartbeat(ctx context.Context) error {
	defer l.wg.Done()
	defer l.cancel()

	t := time.NewTicker(time.Second * time.Duration(gconfig.Cfg.Agent.HeartbeatInterval))
	defer t.Stop()

OUT_FOR:
	for {
		select {
		case <-ctx.Done():
			break OUT_FOR

		case <-t.C:
			l.heartbeatLostCount++
			if l.heartbeatLostCount >= MaxHeartbeatLost {
				glog.RunS().Warnf("link %d heartbeat lost %d times, force closing link", l.Info.ID, l.heartbeatLostCount)
				return fmt.Errorf("heartbeat lost %d times, force closing link", l.heartbeatLostCount)
			}

			seqID := <-l.seqId
			msg := &spgwpb.Msg{
				Head: &spgwpb.Head{
					CommandId: spgwpb.CmdID_CmdActiveTestReq,
					SeqId:     seqID,
				},
			}
			raw, encErr := l.myCodec.Encode(msg)
			if encErr != nil {
				return encErr
			}
			msg.RawMsg = raw
			err := l.SendMsg(msg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *Link) sendLoop() error {
	defer l.wg.Done()

	defer l.cancel()
	for {
		select {
		case <-l.ctx.Done():
			glog.RunS().Debugf("send loop ctx done")
			return nil
		case msg := <-l.queue:
			if err := l.encodeMsgAndSend(msg); err != nil {
				glog.RunS().With(zap.Any("info", l.Info)).With(zap.Error(err)).Debugf("encodeMsg failed, send msg error")
				return err
			}
		}
	}
}

func (l *Link) readLoop() {
	defer l.wg.Done()
	defer l.cancel()

	for {
		select {
		case <-l.ctx.Done():
			// glog.RunS().Debugf("read loop ctx done")
			return
		default:
		}

		err := l.OnRead(l.ctx)
		if err != nil {
			glog.RunS().With(zap.Any("info", l.Info)).With(zap.Error(err)).Debug("read error")
			return
		}
	}
}

func (l *Link) connectRemote() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", l.Info.RemoteAddress, time.Second*4)
	if err != nil {
		glog.RunS().Errorf("dial remote error: %v", err)
		return nil, err
	}
	glog.RunS().With("linkid", l.Info.ID).With(zap.String("remote", l.Info.RemoteAddress)).Debugf("dial remote succ")
	return conn, nil
}

func (l *Link) runClient() error {
	defer l.wg.Done()

	l.State = consts.StateConnecting
	conn, err := l.connectRemote()
	if err != nil {
		glog.RunS().With(zap.Error(err)).Errorf("connect remote error")
		return err
	}
	l.Conn = conn

	l.wg.Add(1)
	go l.waitLogin()

	l.wg.Add(1)
	go l.sendLoop()

	l.wg.Add(1)
	l.readLoop()

	return nil
}

func (l *Link) onRecvBindReqAndAck(ctx context.Context, msg *spgwpb.Msg) error {
	if l.State == consts.StateAuthed {
		return nil
	}

	loginOK := false
	defer func() {
		l.loginOK <- loginOK
	}()

	switch msg.Head.CommandId {
	case spgwpb.CmdID_CmdBindReq:
		if l.Mode == ModeClient {
			return errors.New("client mode not support connect command")
		}

		remoteAddr, _, _ := net.SplitHostPort(l.Conn.RemoteAddr().String())

		msg.Head.RecvTime = time.Now().UnixMilli()
		msg.BindAccountReq.IpAddress = remoteAddr

		ack := l.mng.OnBindReq(ctx, l, msg)
		l.SendMsg(ack)
		if ack.Head.Status != 0 {
			glog.BinS().With("req", msg.String()).Warnf("bind req failed: %v", ack.Head.Status)
		} else {
			loginOK = true
		}

	case spgwpb.CmdID_CmdBindAck:
		err := l.mng.OnBindAck(ctx, l, msg)
		if err != nil {
			return err
		}
		loginOK = true
	}
	if loginOK {
		l.State = consts.StateAuthed
	}
	return nil
}

func (l *Link) onRecvMsg(ctx context.Context, msg *spgwpb.Msg) error {
	var err error

	switch msg.Head.CommandId {
	case spgwpb.CmdID_CmdBindAck, spgwpb.CmdID_CmdBindReq:
		err = l.onRecvBindReqAndAck(ctx, msg)
		if err != nil {
			return err
		}

	case spgwpb.CmdID_CmdActiveTestReq, spgwpb.CmdID_CmdActiveTestAck:
		l.heartbeatLostCount = 0

	case spgwpb.CmdID_CmdSubmitReq:
		// iagent 发送的消息
		if l.State == consts.StateAuthed {
			msg.Head.FromAccountId = l.Info.IAccountID
			msg.Head.FromSpid = l.Info.Spid
			msg.Head.FromLinkId = uint32(l.Info.ID)

			// 检查并发限制
			if !l.rl.Allow() {
				ack := &spgwpb.Msg{
					Head: &spgwpb.Head{
						CommandId: spgwpb.CmdID_CmdSubmitAck,
						SeqId:     msg.Head.SeqId,
					},
					SubmitAck: &spgwpb.SubmitAck{
						MsgId:  0x1212,
						Result: consts.ErrCmppCodeMaxFlowErr,
					},
				}
				l.SendMsg(ack)
				return nil
			}
			l.mng.SendMsg(msg)
		}
	case spgwpb.CmdID_CmdDeliverAck:
		// iagent 发送的消息
		if l.State == consts.StateAuthed {
			msg.Head.FromAccountId = l.Info.IAccountID
			msg.Head.FromSpid = l.Info.Spid
			l.mng.SendMsg(msg)
		}

	case spgwpb.CmdID_CmdSubmitAck, spgwpb.CmdID_CmdDeliverReq, spgwpb.CmdID_CmdReportReq:
		// oagent 发送的消息
		if l.State == consts.StateAuthed {
			msg.Head.FromAccountId = l.Info.OAccountID
			msg.Head.FromSpid = l.Info.Spid
			l.mng.SendMsg(msg)
		}

	case spgwpb.CmdID_CmdTerminateReq:
		time.Sleep(time.Millisecond * 100)
		return fmt.Errorf("terminate command received, closing link")

	default:
		// return fmt.Errorf("unhandled %s command %s", l.Info.Proto, msg.Head.CommandId)
		// 忽略未知命令字
		return nil
	}

	return nil
}

func (l *Link) OnRead(ctx context.Context) error {
	cr := newConnReader(l.Conn)
	for {
		l.Conn.SetReadDeadline(time.Now().Add(time.Second * 45))
		msg, ack, err := l.myCodec.Decode(cr)
		if err != nil {
			if ack != nil {
				l.encodeMsgAndSend(ack)
			}
			if errors.Is(err, codec.ErrPacketNotComplete) {
				return nil
			}
			return err
		}
		if ack != nil {
			l.SendMsg(ack)
		}
		if msg != nil {
			msg.Head.Source = spgwpb.MsgSource_MsgSrcTCP
			msg.Head.FromNodeId = l.mng.GetNodeID()
			msg.Head.FromNodeType = l.mng.GetNodeType()

			if l.Mode == ModeClient {
				msg.Head.FromAccountId = l.Info.OAccountID
			} else {
				msg.Head.FromAccountId = l.Info.IAccountID
			}
			if e := l.onRecvMsg(ctx, msg); e != nil {
				return e
			}
		}
	}
}

func (l *Link) buildRawMsg(msg *spgwpb.Msg) []byte {
	data, err := l.myCodec.Encode(msg)
	if err != nil {
		glog.RunS().With(zap.Error(err)).Debug("buildRawMsg encode failed")
		return nil
	}
	return data
}

func (l *Link) encodeMsgAndSend(msg *spgwpb.Msg) error {
	if msg == nil {
		return nil
	}

	if glog.RunS().Level() <= zapcore.DebugLevel {
		glog.BinS().With(
			zap.Any("head", msg.Head),
			zap.Any("rawData", hex.Dump(msg.RawMsg)),
		).Debugf("encodeMsgAndSend")
	}

	if l.Conn == nil {
		return fmt.Errorf("conn is not active")
	}

	msg.RawMsg = l.buildRawMsg(msg)
	if len(msg.RawMsg) == 0 {
		return fmt.Errorf("encode msg failed: msg.RawMsg is nil")
	}

	l.Conn.SetWriteDeadline(time.Now().Add(gconfig.WriteTimeout(gconfig.Cfg.Agent.WriteTimeout.Duration)))
	_, err := l.Conn.Write(msg.RawMsg)
	return err
}

func (l *Link) runServer() error {
	defer l.wg.Done()

	l.wg.Add(1)
	go l.waitLogin()

	l.wg.Add(1)
	go l.sendLoop()

	l.wg.Add(1)
	l.readLoop()
	return nil
}

func (l *Link) Close(reason string) {
	glog.RunS().Debugf("link closing, reason: %s", reason)
	if l.Conn != nil {
		l.Conn.Close()
	}
	l.cancel() // allow reentrancy
}

func (l *Link) Run(ctx context.Context, cancel context.CancelFunc) error {
	l.ctx = ctx
	l.cancel = cancel
	defer func() {
		if l.Conn != nil {
			l.Conn.Close()
		}
		glog.RunS().Debugf("link closed, mode=%v", l.Mode.String())
		l.cancel()
		close(l.queue)
	}()

	defer func() {
		if l.mng != nil && l.Mode == ModeClient {
			l.mng.OnLinkBroken(l)
		}
	}()

	var err error
	if l.Mode == ModeClient {
		l.wg.Add(1)
		err = l.runClient()
	} else {
		l.wg.Add(1)
		err = l.runServer()
	}

	l.wg.Wait()
	return err
}
