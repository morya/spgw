package server

import (
	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/spgwpb"
)

func (a *App) publishBill(rec *spgwpb.BillRecord) {
	if rec == nil {
		return
	}

	msg := &spgwpb.Msg{
		BillRecord: rec,
	}
	data := gostring.JsonEncode(msg)
	if err := a.mq.Publish(consts.TopicBill, data); err != nil {
		glog.RunS().Errorf("publish bill failed, type=%d, err=%v", rec.BillType, err)
	}
}
