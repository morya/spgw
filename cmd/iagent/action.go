package iagent

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"github.com/morya/gommon/wait_group_wrapper"

	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/httpserver"
)

var (
	portCmpp30 int
	portCmpp20 int
)

func action(c *cli.Context) error {
	glog.RunS().Info("iagent starting...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := newApp()
	if err := app.Init(); err != nil {
		return fmt.Errorf("init app failed: %v", err)
	}

	addrCmpp30 := fmt.Sprintf("0.0.0.0:%d", portCmpp30)
	l30, err := net.Listen("tcp", addrCmpp30)
	if err != nil {
		glog.RunS().Fatalf("create listener error: %v", err)
	}
	glog.RunS().Infof("cmpp30 listening on %s", addrCmpp30)

	addrCmpp20 := fmt.Sprintf("0.0.0.0:%d", portCmpp20)
	l20, err := net.Listen("tcp", addrCmpp20)
	if err != nil {
		glog.RunS().Fatalf("create cmpp20 listener error: %v", err)
	}
	glog.RunS().Infof("cmpp20 listening on %s", addrCmpp20)

	wg := &wait_group_wrapper.WaitGroupWrapper{}

	wg.Wrap(func() {
		defer cancel()

		port := c.Int("http-port")
		if err := httpserver.Run(ctx, port, "/iagent"); err != nil {
			glog.RunS().Fatalf("http.httpServer failed: %v", err)
		}
	})

	wg.Wrap(func() {
		defer cancel()
		defer l30.Close()
		defer l20.Close()
		defer app.Stop()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return
		case sig := <-sig:
			glog.RunS().Infof("signal %v received", sig)
			return
		}
	})

	wg.Wrap(func() {
		app.statCounter(ctx)
	})

	wg.Wrap(func() {
		defer cancel()
		app.Run()
	})

	wg.Wrap(func() {
		defer cancel()
		acceptLoop(ctx, l30, app, "cmpp30")
	})

	wg.Wrap(func() {
		defer cancel()
		acceptLoop(ctx, l20, app, "cmpp20")
	})

	wg.Wait()
	return nil
}

func acceptLoop(ctx context.Context, ln net.Listener, app *App, proto string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				glog.RunS().Errorf("accept error: %v", err)
			}
			return
		}
		go app.handleConn(ctx, conn, proto)
	}
}
