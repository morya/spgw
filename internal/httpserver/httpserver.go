package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http/pprof"
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/glog"
)

// Run 启动一个内置的 HTTP 控制端 server，包含日志级别动态调整和 pprof。
// 阻塞直到 ctx.Done()，返回 nil。
func Run(ctx context.Context, port int, prefix string, opts ...func(g *gin.Engine)) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	glog.RunS().With(zap.Int("port", port)).Infof("http server start")

	defer l.Close()
	gin.SetMode(gin.ReleaseMode)
	g := gin.Default()
	g.Use(gin.Recovery())

	if prefix == "" {
		prefix = "/"
	}

	grp := g.Group(prefix)
	grp.GET("/loglevel", func(c *gin.Context) {
		l := glog.Logger.Level()
		c.JSON(200, gin.H{
			"level": l.String(),
		})
	})

	type Req struct {
		Level string `json:"level" form:"level"`
	}
	grp.POST("/loglevel", func(c *gin.Context) {
		req := &Req{}
		if err := c.ShouldBindJSON(req); err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}
		l, err := zap.ParseAtomicLevel(req.Level)
		if err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}
		glog.SetLevel(l.Level())
		c.JSON(200, gin.H{
			"level": l.String(),
		})
	})

	grp.Any("/debug/pprof/",
		// 需要去除掉 prefix 前缀
		gin.WrapF(pprof.Index),
	)
	grp.Any("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	grp.Any("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	grp.Any("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	grp.Any("/debug/pprof/trace", gin.WrapF(pprof.Trace))

	for _, opt := range opts {
		opt(g)
	}

	go g.RunListener(l)

	<-ctx.Done()
	l.Close()
	return nil
}
