package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	"github.com/morya/gommon/strmd5"
)

const (
	ErrCodeInvalidParam = "PK:1001"
	ErrCodeAuthFailed   = "PK:1003"
	ErrCodeOther        = "PK:1005"
	ErrCodeTooMany      = "PK:1008"
)

type App struct {
	cfg *Config
	cb  *Callback
}

func NewApp(cfg *Config, cb *Callback) *App {
	return &App{cfg: cfg, cb: cb}
}

func mapKey(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	return cast.ToString(v)
}

func (a *App) signAuthMiddle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			fail(c, ErrCodeInvalidParam, "方法错误")
			c.Abort()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			fail(c, ErrCodeInvalidParam, "读取参数失败")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		m := make(map[string]any)
		if err := json.Unmarshal(body, &m); err != nil {
			fail(c, ErrCodeInvalidParam, "读取参数失败")
			c.Abort()
			return
		}

		clientSign := mapKey(m, "sign")
		if clientSign == "" {
			fail(c, ErrCodeInvalidParam, "sign不能为空")
			c.Abort()
			return
		}

		// 签名计算：去掉空值和 sign/key 字段，按 key ASCII 升序排列，拼接 key=apiKey，MD5 后转大写
		sortedKeys := make([]string, 0)
		for k := range m {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		ss := strings.Builder{}
		for _, k := range sortedKeys {
			if k == "sign" || k == "key" {
				continue
			}
			if k == "phones" {
				// phones 字段不参与签名计算
				continue
			}
			v := cast.ToString(m[k])
			if len(v) == 0 {
				continue
			}
			ss.WriteString(k)
			ss.WriteString("=")
			ss.WriteString(v)
			ss.WriteString("&")
		}

		// 兼容 appid/appId 两种大小写
		appId := mapKey(m, "appId")
		if appId == "" {
			appId = mapKey(m, "appid")
		}
		if appId == "" {
			fail(c, ErrCodeInvalidParam, "appid不能为空")
			c.Abort()
			return
		}

		apiKey, found := a.cfg.Accounts[appId]
		if !found {
			fail(c, ErrCodeInvalidParam, "appid不存在")
			c.Abort()
			return
		}

		ts := cast.ToInt64(mapKey(m, "timestamp"))
		tt := time.Unix(ts, 0)
		now := time.Now()
		if now.Sub(tt).Seconds() > 300 {
			fail(c, ErrCodeInvalidParam, "timestamp过期")
			c.Abort()
			return
		}

		ss.WriteString("key=")
		ss.WriteString(apiKey)

		raw := ss.String()
		h := strmd5.StrMd5(raw)
		calculatedSign := strings.ToUpper(h)

		if calculatedSign != clientSign {
			zap.S().With(
				zap.String("appId", appId),
				zap.String("clientSign", clientSign),
				zap.String("calculatedSign", calculatedSign),
				zap.String("raw", raw),
			).Warnf("签名校验失败")
			fail(c, ErrCodeAuthFailed, "签名校验失败")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (a *App) handleSendOne(c *gin.Context, req *ReqSingleSend) {
	appID := req.AppID
	spMsgID := req.SpMsgId
	phone := req.Phone
	content := req.Content
	if phone == "" || content == "" {
		fail(c, ErrCodeInvalidParam, "phone/content required")
		return
	}

	msgID := newMsgID()
	msg := SentMessage{
		MsgID:    msgID,
		Phone:    phone,
		SpMsgID:  spMsgID,
		AppID:    appID,
		ExtendNo: req.ExtendNo,
		Content:  content,
	}
	a.cb.Enqueue(msg)

	zap.S().Infow("send", "appId", appID, "phone", phone, "msgId", msgID, "spMsgId", spMsgID, "contentLen", len(content))
	ok(c, map[string]any{"msgId": msgID})
}

type batchRespItem struct {
	MsgID string `json:"msgId"`
	Phone string `json:"phone"`
}

func (a *App) handleBatchSend(c *gin.Context, req *ReqBatchSend) {
	content := req.Content
	if content == "" {
		fail(c, ErrCodeInvalidParam, "content required")
		return
	}
	extendNo := req.ExtendNo

	rawPhones := req.Phones
	if len(rawPhones) == 0 {
		fail(c, ErrCodeInvalidParam, "phones required")
		return
	}
	if len(rawPhones) > a.cfg.BatchMax {
		fail(c, ErrCodeTooMany, "phones exceeds batch-max")
		return
	}

	items := make([]batchRespItem, 0, len(rawPhones))
	for _, p := range rawPhones {
		phone := p.Phone
		spMsgID := p.SpMsgId
		if phone == "" {
			fail(c, ErrCodeInvalidParam, "phones[].phone required")
			return
		}

		msgID := newMsgID()
		a.cb.Enqueue(SentMessage{
			MsgID:    msgID,
			Phone:    phone,
			SpMsgID:  spMsgID,
			AppID:    req.AppID,
			ExtendNo: extendNo,
			Content:  content,
		})
		items = append(items, batchRespItem{MsgID: msgID, Phone: phone})
	}

	zap.S().Infow("batchSend", "appId", req.AppID, "count", len(items), "contentLen", len(content))

	ok(c, items)
}
