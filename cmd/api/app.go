package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	"github.com/morya/gommon/strmd5"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/httpsend"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/natsx"
	"github.com/morya/spgw/internal/router/middle"
	"github.com/morya/spgw/internal/router/response"
	routerhelper "github.com/morya/spgw/internal/router/router_helper"
)

type App struct {
	e  *gin.Engine
	mq *nats.Conn

	mapAppidToIAccount *sync.Map
	mapAppidToOAccount *sync.Map

	callbackMgr *httpsend.Manager
}

func newApp() *App {
	gin.SetMode(gin.ReleaseMode)
	app := &App{
		e: gin.New(),

		mapAppidToIAccount: &sync.Map{},
		mapAppidToOAccount: &sync.Map{},

		callbackMgr: httpsend.NewManager(),
	}

	app.initMQ()

	app.loadIAccounts()
	app.loadOAccounts()
	app.initCallbackAdapters()

	return app
}

func (a *App) initMQ() {
	a.mq = natsx.MustConnect(gconfig.Cfg.Nats)
	lo.Must1(a.mq.QueueSubscribe("spgw.api", "queue", a.OnSystemMsg))
}

// accountLoader 根据 appId 返回对应的 ApiKey，用于签名验证中间件
// 返回 (apikey, accountPtr, found)
type accountLoader func(appId string) (apiKey string, account any, found bool)

// signMiddle 通用签名验证中间件
// 通过 accountLoader 参数区分 IAccount / OAccount 来源
func (a *App) signMiddle(loader accountLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			response.ErrorWithHttpCode(c, http.StatusBadRequest, consts.ErrRestCodeErrSign, "方法错误")
			c.Abort()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.ErrorWithHttpCode(c, http.StatusBadRequest, consts.ErrRestCodeErrSign, "读取参数失败")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		m := make(map[string]any)
		if err := json.Unmarshal(body, &m); err != nil {
			glog.RunS().With("body", string(body)).Errorf("参数解析失败: %v", err)
			response.ErrorWithHttpCode(c, http.StatusBadRequest, consts.ErrRestCodeErrSign, "参数解析失败")
			c.Abort()
			return
		}

		// 签名计算：去掉空值和 sign/key 字段，按 key ASCII 升序排列，拼接 key=apiKey，MD5 后转大写
		vv := url.Values{}
		for k, v := range m {
			if k == "sign" || k == "key" {
				continue
			}
			if k == "phones" {
				// phones 字段不参与签名计算
				continue
			}
			v := cast.ToString(v)
			if len(v) == 0 {
				continue
			}
			vv.Add(k, v)
		}

		// 兼容 appid/appId 两种大小写
		appId := vv.Get("appId")
		if appId == "" {
			appId = vv.Get("appid")
		}
		if appId == "" {
			response.ErrorWithHttpCode(c, http.StatusBadRequest, consts.ErrRestCodeErrSign, "appid不能为空")
			c.Abort()
			return
		}

		apiKey, acc, found := loader(appId)
		if !found {
			response.ErrorWithHttpCode(c, http.StatusBadRequest, consts.ErrRestCodeErrSign, "appid不存在")
			c.Abort()
			return
		}

		ts := cast.ToInt64(vv.Get("timestamp"))
		tt := time.Unix(ts, 0)
		now := time.Now()
		if now.Sub(tt).Seconds() > 300 {
			response.ErrorWithHttpCode(c, http.StatusBadRequest, consts.ErrRestCodeErrTS, "timestamp过期")
			c.Abort()
			return
		}

		raw := vv.Encode() + "&key=" + apiKey
		h := strmd5.StrMd5(raw)
		calculatedSign := strings.ToUpper(h)

		clientSign := vv.Get("sign")

		if calculatedSign != clientSign {
			glog.RunS().With(
				zap.String("appId", appId),
				zap.String("clientSign", clientSign),
				zap.String("calculatedSign", calculatedSign),
				zap.String("raw", raw),
			).Warnf("签名校验失败")
			response.ErrorWithHttpCode(c, http.StatusUnauthorized, consts.ErrRestCodeErrSign, "签名校验失败")
			c.Abort()
			return
		}

		// 校验通过，将 appId 和 account 存入 context 供后续 handler 使用
		c.Set("appid", appId)
		c.Set("account", acc)

		c.Next()
	}
}

func (a *App) Run() error {
	a.e.Use(middle.BodyDump(), gin.Recovery(), cors.Default())

	addr := fmt.Sprintf(":%d", httpPort)

	// SP HTTP API 组（IAccount 验签）
	apiGrp := a.e.Group("/api/sms", a.signMiddle(func(appId string) (string, any, bool) {
		if acc, ok := a.mapAppidToIAccount.Load(appId); ok {
			return acc.(*models.SpInfo).ApiKey, acc, true
		}
		return "", nil, false
	}))
	a.initApi(apiGrp)

	// 下游回调组（OAccount 验签）
	cbGrp := a.e.Group("/api/v1/callback")
	cbGrp.POST("/:vendor", a.OnCallback)

	srv := &http.Server{Addr: addr, Handler: a.e}
	return srv.ListenAndServe()
}

func (a *App) loadOAccounts() (err error) {
	var arr []*models.OAccount
	if err := db.MainDB.Model(&models.OAccount{}).Where("allow_http = ?", true).Find(&arr).Error; err != nil {
		zap.S().Errorf("load oaccount failed: %v", err)
		return err
	}
	for _, oa := range arr {
		a.mapAppidToOAccount.Store(oa.Appid, oa)
	}
	zap.S().Infof("loaded %d http oaccounts", len(arr))
	return nil
}

func (a *App) initApi(g *gin.RouterGroup) {
	g.POST("/send", routerhelper.HandleWithModel(a.OnSingleSend))
	g.POST("/batchSend", routerhelper.HandleWithModel(a.OnBatchSend))
}
