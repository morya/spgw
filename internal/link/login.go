package link

import (
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/glog"
)

func (l *Link) buildAndSendBindMsg() error {
	m := l.myCodec.BuildBindReq(l.Info.Spid, l.Info.Pswd, <-l.seqId)
	return l.SendMsg(m)
}

func (l *Link) waitLogin() error {
	defer l.wg.Done()

	defer func() {
		glog.BinS().Debugf("wait login failed, closing link")
		l.cancel()
	}()

	if l.Mode == ModeClient {
		err := l.buildAndSendBindMsg()
		if err != nil {
			return err
		}
	}

	select {
	case <-l.ctx.Done():
		zap.S().With(zap.Any("link", l.Info)).Debugf("wait login, outer ctx done")
		return nil

	case <-time.After(time.Second * 5):
		zap.S().With(zap.Any("link", l.Info)).Debugf("wait login timeout")
		return errors.New("wait login timeout")

	case ok := <-l.loginOK:
		if !ok || l.State != consts.StateAuthed {
			return errors.New("wait login failed")
		}
		break
	}

	l.wg.Add(1)
	return l.heartbeat(l.ctx)
}
