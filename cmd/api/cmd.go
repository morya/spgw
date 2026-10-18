package api

import (
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
)

var httpPort int

var Command = &cli.Command{
	Name:  "api",
	Usage: "api to recv msg from http",

	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "port",
			Usage:       "http listen port",
			Value:       80,
			Destination: &httpPort,
		},
	},

	Before: before,
	Action: action,
}

func before(ctx *cli.Context) error {
	name := ctx.Command.Name
	glog.InitLoggers(ctx.String("loglevel"), ctx.String("logformat"), ctx.String("logdir"), name)

	return db.ConnectMainDB(gconfig.Cfg.MainDB)
}

func action(*cli.Context) error {
	zap.S().Info("api starting...")

	app := newApp()
	return app.Run()
}
