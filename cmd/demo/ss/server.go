package ss

import (
	"github.com/urfave/cli/v2"
	"go-micro.dev/v5"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/util/microx"
	"github.com/morya/spgw/spgwpb"
)

var Command = &cli.Command{
	Name:  "demo-server",
	Usage: "demo server",
	Action: func(c *cli.Context) error {
		s := micro.NewService(micro.Name("spgw.demo-server"))
		s.Init(micro.WrapHandler(microx.LogHandle), micro.WrapSubscriber(microx.LogSubscribe))

		app := NewApp()
		spgwpb.RegisterSpgwHandler(s.Server(), app)

		if err := s.Run(); err != nil {
			zap.S().Errorw("s.Run error", "err", err)
			return err
		}

		return nil
	},
}
