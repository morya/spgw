package portal

import (
	"github.com/urfave/cli/v2"

	"github.com/morya/spgw/bootstrap"
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/routes"
	"github.com/morya/spgw/internal/utils/ginhelper"
	"github.com/morya/spgw/plugins"
)

var Command = &cli.Command{
	Name:    "portal",
	Usage:   "portal management API server",
	Aliases: []string{"p"},

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "bill-dir",
			Aliases: []string{"b"},
			Value:   "/data/bills/",
			Usage:   "SP 话单文件根目录（下载话单时从此目录读取实际文件）",
		},
	},

	Before: before,
	After:  after,
	Action: action,
}

func before(c *cli.Context) error {
	bootstrap.Init()
	plugins.Init()
	app.BillDir = c.String("bill-dir")
	return nil
}

func after(c *cli.Context) error {
	return nil
}

func action(c *cli.Context) error {
	engine := ginhelper.GetEngine()
	routes.InitRoutes(engine)
	ginhelper.InitPluginRoutes(engine)
	return ginhelper.StartServer(engine)
}
