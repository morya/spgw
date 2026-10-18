package billdumper

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/morya/gommon/wait_group_wrapper"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/natsx"
	"github.com/morya/spgw/spgwpb"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	queueCap       = 10000
	queueGroupName = "billdumper"
	rotateCheckSec = 60
)

type App struct {
	mq     *nats.Conn
	ctx    context.Context
	cancel context.CancelFunc

	writer *BillWriter
	queue  chan *spgwpb.BillRecord

	NatsQueueDropCount uint32
}

func newApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:    ctx,
		cancel: cancel,
		queue:  make(chan *spgwpb.BillRecord, queueCap),
	}
}

func (a *App) Init() error {
	a.mq = natsx.MustConnect(gconfig.Cfg.Nats)
	a.writer = NewBillWriter(gconfig.Cfg.BillDumper, NodeID)
	return nil
}

func (a *App) Run() error {
	defer a.cancel()

	wg := &wait_group_wrapper.WaitGroupWrapper{}

	wg.Wrap(func() {
		defer a.writer.Close()
		a.run()
	})

	wg.Wrap(func() {
		lo.Must(a.mq.QueueSubscribe(consts.TopicBill, queueGroupName, a.onMsgFromMQ))
		<-a.ctx.Done()
	})

	wg.Wrap(func() {
		ticker := time.NewTicker(rotateCheckSec * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				a.writer.CheckRotate()
			}
		}
	})

	wg.Wrap(func() {
		defer a.cancel()
		defer a.writer.Close()
		defer a.mq.Close()
		defer close(a.queue)

		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		<-s
		zap.S().Info("billdumper received exit signal")
	})

	zap.S().Info("billdumper running...")
	wg.Wait()
	zap.S().Info("billdumper exit")
	return nil
}

func (a *App) onMsgFromMQ(msg *nats.Msg) {
	m := &spgwpb.Msg{}
	if err := json.Unmarshal(msg.Data, m); err != nil {
		glog.RunS().Errorf("unmarshal bill msg error: %v", err)
		return
	}

	rec := m.GetBillRecord()
	if rec == nil {
		glog.RunS().Warnf("bill msg without bill_record, head: %s", m.GetHead().String())
		return
	}

	select {
	case a.queue <- rec:
	default:
		atomic.AddUint32(&a.NatsQueueDropCount, 1)
		glog.RunS().Warnf("billdumper queue full, drop bill record, type=%d", rec.BillType)
	}
}

func (a *App) run() {
	for rec := range a.queue {
		if err := a.writer.Write(rec); err != nil {
			glog.RunS().Errorf("write bill failed, type=%d, err=%v", rec.BillType, err)
		}
	}
}
