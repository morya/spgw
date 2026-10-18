package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	app := cli.NewApp()
	app.Name = "recv"
	app.Usage = "south-bound HTTP stub: single/batch send + async report callback"
	app.Flags = flags()
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		zap.S().Fatalf("recv exit: %v", err)
	}
}

func run(c *cli.Context) error {
	initLogger(c.String("log-level"))

	cfg, err := parseConfig(c)
	if err != nil {
		return err
	}
	zap.S().Infow("config loaded",
		"listen", cfg.Listen,
		"accounts", len(cfg.Accounts),
		"callback_url", cfg.CallbackURL,
		"callback_delay", cfg.CallbackDelay,
		"callback_fail_rate", cfg.CallbackFailRate,
		"batch_max", cfg.BatchMax,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cb := NewCallback(cfg)
	cb.Start(ctx)

	app := NewApp(cfg, cb)
	srv := NewServer(cfg, app)

	httpErr := make(chan error, 1)
	go func() {
		zap.S().Infof("listening on %s", cfg.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErr <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sigCh:
		zap.S().Warnf("signal %v, shutting down", s)
	case err := <-httpErr:
		zap.S().Errorf("http server error: %v", err)
		cancel()
		cb.Drain(cfg.ShutdownTimeout)
		return err
	}

	shutCtx, shutCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		zap.S().Errorf("http shutdown: %v", err)
	}
	cancel()
	cb.Drain(cfg.ShutdownTimeout)
	zap.S().Info("recv exited cleanly")
	return nil
}

func initLogger(level string) {
	cfg := zap.NewDevelopmentConfig()
	cfg.DisableStacktrace = true
	cfg.DisableCaller = true
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, _ := cfg.Build()
	zap.ReplaceGlobals(logger)
}
