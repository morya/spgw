package oagent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/gommon/wait_group_wrapper"

	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/httpserver"
)

func action(c *cli.Context) error {
	zap.S().Infof("oagent starting ...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := newApp()
	if err := app.Init(); err != nil {
		return fmt.Errorf("init app failed: %v", err)
	}

	wg := &wait_group_wrapper.WaitGroupWrapper{}

	wg.Wrap(func() {
		defer cancel()

		port := c.Int("http-port")
		if err := httpserver.Run(ctx, port, "/oagent"); err != nil {
			glog.RunS().Fatalf("http.httpServer failed: %v", err)
		}
	})

	wg.Wrap(func() {
		defer cancel()
		defer app.Stop()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return

		case <-sig:
			return
		}
	})

	wg.Wrap(func() {
		if err := app.Start(); err != nil {
			zap.S().Errorf("oagent start failed, err: %v", err)
			return
		}
	})

	wg.Wait()
	return nil
}
