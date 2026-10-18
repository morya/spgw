package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/spf13/cast"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/httpsend"
	"github.com/morya/spgw/spgwpb"
)

var reportHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

func (a *App) sendReportViaHTTP(m *spgwpb.Msg, orgSubmit *spgwpb.Msg, logger *zap.SugaredLogger) {
	ia := a.router.FindIAccountByID(orgSubmit.Head.FromAccountId)
	if ia == nil {
		logger.With(zap.Uint32("accountId", orgSubmit.Head.FromAccountId)).Errorf("find iaccount failed")
		a.router.IncrIReportFailCnt(orgSubmit.Head.FromAccountId)
		return
	}

	if ia.CallbackUrl == "" {
		logger.With(zap.Uint32("accountId", uint32(ia.ID))).Warnf("callback url is empty, skip report")
		a.router.IncrIReportFailCnt(uint32(ia.ID))
		return
	}

	reportReq := m.ReportReq
	rd := httpsend.ReportData{
		Status:       reportReq.Stat,
		Phone:        reportReq.DestTerminalId,
		DeliveryTime: reportReq.DoneTime,
		MsgID:        cast.ToString(reportReq.MsgId),
		AppID:        ia.Appid,
		ExtendCode:   orgSubmit.SubmitReq.SrcId,
	}

	body, err := json.Marshal(rd)
	if err != nil {
		logger.With(zap.Any("err", err)).Errorf("marshal report data failed")
		a.router.IncrIReportFailCnt(uint32(ia.ID))
		return
	}

	req, err := http.NewRequest("POST", ia.CallbackUrl, bytes.NewReader(body))
	if err != nil {
		logger.With(zap.Any("err", err)).Errorf("build callback request failed")
		a.router.IncrIReportFailCnt(uint32(ia.ID))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := reportHTTPClient.Do(req)
	if err != nil {
		logger.With(zap.Any("err", err), zap.String("callbackUrl", ia.CallbackUrl)).Warnf("send callback failed")
		a.router.IncrIReportFailCnt(uint32(ia.ID))
		return
	}
	defer resp.Body.Close()

	a.router.IncrIReportSuccCnt(uint32(ia.ID))
	logger.With(zap.Int("statusCode", resp.StatusCode), zap.String("callbackUrl", ia.CallbackUrl)).Debugf("report callback sent")
}
