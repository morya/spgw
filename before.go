package main

import (
	"os"

	"github.com/samber/lo"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/rconn"
)

func before(c *cli.Context) error {
	os.Args = os.Args[:1] // hack go-micro 命令行解析的问题

	p := lo.Must(process.NewProcess(int32(os.Getpid())))
	processName := lo.Must(p.Name())

	glog.InitLoggers(c.String("loglevel"), c.String("logformat"), c.String("logdir"), processName+"-"+c.Command.Name)
	zap.ReplaceGlobals(glog.Logger)

	if err := gconfig.Load(c.String("config")); err != nil {
		return err
	}

	if err := rconn.Connect(gconfig.Cfg.RedisAddress); err != nil {
		return err
	}
	zap.S().Infof("redis connect success, addr: %s", gconfig.Cfg.RedisAddress)
	return nil
}
