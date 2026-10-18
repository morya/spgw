package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/spgw/cmd/api"
	"github.com/morya/spgw/cmd/billdumper"
	"github.com/morya/spgw/cmd/iagent"
	"github.com/morya/spgw/cmd/migrate"
	"github.com/morya/spgw/cmd/oagent"
	"github.com/morya/spgw/cmd/portal"
	"github.com/morya/spgw/cmd/server"
	"github.com/morya/spgw/cmd/trace"
)

// @title SPGW API
// @version 1.0
// @description SPGW API文档
// @termsOfService https://github.com/morya/spgw

// @contact.name API Support
// @contact.url https://github.com/morya/spgw/issues
// @contact.email your-email@example.com

// @license.name MIT
// @license.url https://github.com/morya/spgw/blob/master/LICENSE

// @host https://spgw-api.example.com
// @BasePath /api
func main() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))

	app := cli.NewApp()

	app.Flags = flags()
	app.Before = before

	app.Commands = []*cli.Command{
		portal.Command,
		migrate.Command,

		api.Command,
		iagent.Command,
		oagent.Command,
		billdumper.Command,
		server.Command,
		trace.Command,
	}

	zap.S().Info("spgw start")
	if err := app.Run(os.Args); err != nil {
		zap.S().Fatalf("app.Run error: %v", err)
	}
}
