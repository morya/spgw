package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	app := cli.NewApp()
	app.Name = "spclient"
	app.Usage = "CMPP SP-side stub: submit_req sender + ack/report collector"

	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "log-level", Value: "info", Usage: "debug / info / warn / error"},
	}
	app.Before = func(c *cli.Context) error {
		initLogger(c.String("log-level"))
		return nil
	}

	app.Commands = []*cli.Command{
		{
			Name:   "bench",
			Usage:  "run performance benchmark against spgw iagent",
			Flags:  benchFlags(),
			Action: runBench,
		},
		{
			Name:    "unittest",
			Aliases: []string{"test"},
			Usage:   "run protocol test cases against spgw iagent",
			Flags:   unittestFlags(),
			Action:  runTests,
		},
	}

	zap.S().Info("spclient start")
	if err := app.Run(os.Args); err != nil {
		zap.S().Fatalf("spclient exit: %v", err)
	}
}

func runBench(c *cli.Context) error {
	cfg, err := parseConfig(c)
	if err != nil {
		return err
	}
	zap.S().Infof("config: %+v", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigCh
		zap.S().Warnf("signal received: %v, cancelling", s)
		cancel()
	}()

	cli, err := DialAndBind(ctx, cfg)
	if err != nil {
		return err
	}
	defer cli.Close()

	time.Sleep(time.Millisecond * 100)

	tracker := NewTracker()
	cli.Tracker = tracker

	go cli.ReadLoop(ctx)

	startedAt := time.Now()
	sendErr := cli.RunSender(ctx, tracker)
	if sendErr != nil && !errors.Is(sendErr, context.Canceled) {
		zap.S().Errorf("sender error: %v", sendErr)
	}
	sendEndAt := time.Now()
	zap.S().Infof("send phase finished, entering drain (%s)", cfg.DrainTimeout)

	drainCtx, drainCancel := context.WithTimeout(ctx, cfg.DrainTimeout)
	<-drainCtx.Done()
	drainCancel()

	endAt := time.Now()
	if err := cli.Terminate(); err != nil {
		zap.S().Warnf("terminate: %v", err)
	}

	if err := WriteResult(cfg, tracker, startedAt, sendEndAt, endAt); err != nil {
		zap.S().Errorf("write result: %v", err)
		return err
	}
	zap.S().Infof("result written to %s", cfg.ResultFile)

	s := tracker.Snapshot()
	zap.S().Infof("=== bench 汇总 === submitReq 发送=%d  submitAck 接收=%d  deliverReq 接收=%d  deliverAck 回送=%d ===",
		s.Sent, s.AckTotal, s.DeliverReqTotal, s.DeliverAckSent)

	if len(s.AckResultDist) > 0 {
		codes := make([]uint32, 0, len(s.AckResultDist))
		for k := range s.AckResultDist {
			codes = append(codes, k)
		}
		slices.Sort(codes)
		parts := make([]string, 0, len(codes))
		for _, code := range codes {
			parts = append(parts, fmt.Sprintf("%d:%d", code, s.AckResultDist[code]))
		}
		zap.S().Infof("=== submitAck Result 分布 === %s ===", strings.Join(parts, "  "))
	}
	return nil
}

func runTests(c *cli.Context) error {
	cfg := &TestConfig{
		Host20:      c.String("cmpp20-host"),
		Host30:      c.String("cmpp30-host"),
		Spid:        c.String("spid"),
		Password:    c.String("password"),
		WrongSpid:   c.String("wrong-spid"),
		WrongPwd:    c.String("wrong-password"),
		ServiceID:   c.String("service-id"),
		FeeType:     c.String("fee-type"),
		SrcID:       c.String("src-id"),
		Sign:        c.String("sign"),
		Content:     c.String("content"),
		DialTimeout: c.Duration("dial-timeout"),
		ReadTimeout: 10 * time.Second,
	}
	zap.S().Infof("test config: host20=%s host30=%s spid=%s", cfg.Host20, cfg.Host30, cfg.Spid)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		s := <-sigCh
		zap.S().Warnf("signal received: %v, cancelling", s)
		cancel()
	}()

	for {
		runner := NewTestRunner(cfg)
		results := runner.Run(ctx)

		var pass, fail, skip int
		for _, r := range results {
			switch r.Status {
			case StatusPass:
				pass++
			case StatusFail:
				fail++
			case StatusSkip:
				skip++
			}
		}

		if fail > 0 {
			zap.S().Infof("=== 测试统计: 总计=%d %s %s %s ===",
				len(results),
				colorize(fmt.Sprintf("PASS=%d", pass), colorGreen),
				colorize(fmt.Sprintf("FAIL=%d", fail), colorRed),
				colorize(fmt.Sprintf("SKIP=%d", skip), colorYellow),
			)

			return fmt.Errorf("%d test cases failed", fail)
		}

		time.Sleep(time.Millisecond * 300)
	}

	// runner := NewTestRunner(cfg)
	// results := runner.Run(ctx)

	// var pass, fail, skip int
	// for _, r := range results {
	// 	switch r.Status {
	// 	case StatusPass:
	// 		pass++
	// 	case StatusFail:
	// 		fail++
	// 	case StatusSkip:
	// 		skip++
	// 	}
	// }
	// zap.S().Infof("=== 测试统计: 总计=%d %s %s %s ===",
	// 	len(results),
	// 	colorize(fmt.Sprintf("PASS=%d", pass), colorGreen),
	// 	colorize(fmt.Sprintf("FAIL=%d", fail), colorRed),
	// 	colorize(fmt.Sprintf("SKIP=%d", skip), colorYellow),
	// )

	// resultFile := c.String("test-result-file")
	// if err := WriteTestResult(results, resultFile); err != nil {
	// 	zap.S().Errorf("write test result: %v", err)
	// 	return err
	// }
	// zap.S().Infof("test result written to %s", resultFile)

	// if fail > 0 {
	// 	return fmt.Errorf("%d test cases failed", fail)
	// }
	// return nil
}

func initLogger(level string) {
	cfg := zap.NewDevelopmentConfig()
	cfg.DisableStacktrace = true
	// cfg.DisableCaller = true
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
