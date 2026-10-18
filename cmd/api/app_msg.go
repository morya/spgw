package api

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/router/response"
	"github.com/morya/spgw/internal/seq"
	"github.com/morya/spgw/spgwpb"
)

const maxContentRunes = 70

func (a *App) sendToServer(m *spgwpb.Msg) error {
	data := gostring.JsonEncode(m)
	return a.mq.Publish("spgw.server", data)
}

// 系统指令消息
func (a *App) OnSystemMsg(m *nats.Msg) {
	msg := &spgwpb.Msg{}
	if err := json.Unmarshal(m.Data, msg); err != nil {
		glog.RunS().With("topic", m.Subject).With("data", string(m.Data)).Errorf("unmarshal msg error: %v", err)
		return
	}
	glog.RunS().With(zap.Any("msg", msg)).Debugf("recv system msg")

	switch msg.Head.CommandId {
	case spgwpb.CmdID_CmdReloadIAccount:
		err := a.reloadAccountReq(msg)
		if err != nil {
			glog.RunS().Errorf("reload account req error: %v", err)
		}
	}
}

func (a *App) reloadAccountReq(*spgwpb.Msg) (err error) {
	// TODO: 根据名字单独加载特定账号
	err = a.loadIAccounts()
	if err != nil {
		return fmt.Errorf("load iaccounts from db failed: %v", err)
	}

	err = a.loadOAccounts()
	if err != nil {
		return fmt.Errorf("load oaccounts from db failed: %v", err)
	}

	glog.RunS().Infof("reload iaccounts from db success")
	return nil
}

func (a *App) loadIAccounts() error {
	var arr []*models.SpInfo
	err := db.MainDB.Find(&arr).Error
	if err != nil {
		return fmt.Errorf("load iaccounts from db error: %v", err)
	}

	mapAppidToIAccount := &sync.Map{}
	for _, v := range arr {
		mapAppidToIAccount.Store(v.Appid, v)
	}

	a.mapAppidToIAccount = mapAppidToIAccount
	return nil
}

func (a *App) OnSingleSend(c *gin.Context, req *models.ReqSingleSend) {
	if n := utf8.RuneCountInString(req.Content); n > maxContentRunes {
		response.ErrorWithCode(c, consts.ErrRestCodeErrParam, fmt.Sprintf("content 超过最大长度限制，最多 %d 个字符，当前 %d 个字符", maxContentRunes, n))
		return
	}

	acc, ok := c.Get("account")
	if !ok {
		response.ErrorWithCode(c, consts.ErrRestCodeErrUser, "account not found")
		return
	}
	iAcc := acc.(*models.SpInfo)

	msgid := "msg-" + gonanoid.Must(16)

	m := a.buildSubmitMsg(iAcc, req.Content, req.ExtendNo, req.Phone)

	if err := a.sendToServer(m); err != nil {
		response.ErrorWithCode(c, consts.ErrRestCodeErrUser, fmt.Sprintf("send to server failed: %v", err))
		return
	}

	response.SuccessWithData(c, &models.AckSingleSend{MsgID: msgid})
}

func (a *App) OnBatchSend(c *gin.Context, req *models.ReqBatchSend) {
	if len(req.Phones) == 0 {
		response.ErrorWithCode(c, consts.ErrRestCodeErrParam, "phones 不能为空，至少需要 1 条记录")
		return
	}
	if len(req.Phones) > 99 {
		response.ErrorWithCode(c, consts.ErrRestCodeErrParam, fmt.Sprintf("phones 数量超过限制，最多 99 条，当前 %d 条", len(req.Phones)))
		return
	}
	if n := utf8.RuneCountInString(req.Content); n > maxContentRunes {
		response.ErrorWithCode(c, consts.ErrRestCodeErrParam, fmt.Sprintf("content 超过最大长度限制，最多 %d 个字符，当前 %d 个字符", maxContentRunes, n))
		return
	}

	acc, ok := c.Get("account")
	if !ok {
		response.ErrorWithCode(c, consts.ErrRestCodeErrUser, "account not found")
		return
	}
	iAcc := acc.(*models.SpInfo)

	arr := make([]*models.MsgIdArr, 0, len(req.Phones))
	for _, v := range req.Phones {
		msgid := gonanoid.MustGenerate("msg-", 16)

		m := a.buildSubmitMsg(iAcc, req.Content, req.ExtendNo, v.Phone)

		if err := a.sendToServer(m); err != nil {
			glog.RunS().With("err", err, "phone", v.Phone).Errorf("batch send to server failed")
			continue
		}

		arr = append(arr, &models.MsgIdArr{MsgID: msgid, Phone: v.Phone})
	}

	response.SuccessWithData(c, arr)
}

func (a *App) buildSubmitMsg(iAcc *models.SpInfo, content, extendNo, phone string) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId:     spgwpb.CmdID_CmdSubmitReq,
			SeqId:         seq.NextSeqId(),
			FromAccountId: uint32(iAcc.ID),
			FromSpid:      iAcc.Spid,
			Source:        spgwpb.MsgSource_MsgSrcHTTP,
			RecvTime:      time.Now().UnixMilli(),
		},
		SubmitReq: &spgwpb.SubmitReq{
			PkTotal:            1,
			PkNumber:           1,
			RegisteredDelivery: 1,
			MsgFmt:             15,
			ServiceId:          iAcc.ServiceCode,
			MsgSrc:             iAcc.Spid,
			SrcId:              extendNo,
			DestUsrTl:          1,
			DestTerminalId:     []string{phone},
			DestTerminalType:   1,
			MsgLength:          uint32(len(content)),
			MsgContent:         []byte(content),
		},
	}
}
