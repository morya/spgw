package billdumper

import (
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
)

var Command = &cli.Command{
	Name:    "billdumper",
	Aliases: []string{"bd"},
	Usage:   "billdumper - 统计话单落盘",
	Before:  before,
	Action:  action,
}

func before(ctx *cli.Context) error {
	glog.InitLoggers(ctx.String("loglevel"), ctx.String("logformat"), ctx.String("logdir"), ctx.Command.Name)

	if err := initNodeID(); err != nil {
		return err
	}

	return db.ConnectMainDB(gconfig.Cfg.MainDB)
}

func action(*cli.Context) error {
	zap.S().Infof("billdumper starting, nodeID=%d", NodeID)

	app := newApp()
	if err := app.Init(); err != nil {
		return err
	}
	return app.Run()
}
