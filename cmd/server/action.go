package server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/gommon/wait_group_wrapper"

	"github.com/morya/spgw/internal/httpserver"
)

func action(c *cli.Context) error {
	zap.S().Infof("server starting...")
	app := newApp()
	if err := app.Init(); err != nil {
		zap.S().Errorf("init app error: %v", err)
		return err
	}

	wg := &wait_group_wrapper.WaitGroupWrapper{}

	wg.Wrap(func() {
		port := c.Int("http-port")
		httpserver.Run(app.ctx, port, "/server", func(g *gin.Engine) {
			g.GET("/queue", func(c *gin.Context) {
				waitAck := app.waitAck.Count()
				wakRpt := app.waitRpt.Count()

				c.JSON(200, gin.H{
					"msg": "queue",
					"data": gin.H{
						"waitAck": waitAck,
						"waitRpt": wakRpt,
					},
				})
			})
		})
	})

	wg.Wrap(func() {
		defer app.Stop()
		// handle signal and exit app
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		<-s
		zap.S().Infof("app exit")
	})
	wg.Wrap(func() {
		app.Run()
	})

	wg.Wait()
	zap.S().Infof("app stop")
	return nil
}
