package migrate

import (
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/migrate"
)

var Command = &cli.Command{
	Name:    "migrate",
	Usage:   "migrate database",
	Aliases: []string{"m"},

	Action: func(c *cli.Context) error {
		if err := db.ConnectMainDB(gconfig.Cfg.MainDB); err != nil {
			return err
		}
		zap.S().Infof("start migrate db")
		err := migrate.MigrateDB()
		if err != nil {
			zap.S().Errorf("migrate db failed, err: %v", err)
			return err
		}
		zap.S().Infof("migrate db success")
		return nil
	},
}
