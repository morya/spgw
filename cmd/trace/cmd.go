package trace

import (
	"encoding/json"
	"os"
	"os/signal"

	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/spgwpb"
)

var Command = &cli.Command{
	Name:  "trace",
	Usage: "消息跟踪模块",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "topic",
			Usage: "trace topic",
			Value: "spgw.server",
		},
	},

	Action: func(c *cli.Context) error {
		topic := c.String("topic")

		opts := []nats.Option{nats.MaxReconnects(-1)}
		n, err := nats.Connect(gconfig.Cfg.Nats.Address, opts...)
		if err != nil {
			zap.S().Errorf("connect nats error: %v", err)
			return err
		}

		_, err = n.Subscribe(topic, func(m *nats.Msg) {
			msg := &spgwpb.Msg{}
			if err := json.Unmarshal(m.Data, msg); err != nil {
				zap.S().Errorf("unmarshal msg error: %v", err)
				return
			}
			zap.S().With(
				zap.Any("msg", msg.String()),
				zap.String("topic", m.Subject),
			).
				Info("trace")
		})
		if err != nil {
			zap.S().Errorf("subscribe topic error: %v", err)
			return err
		}

		zap.S().Infof("subscribe topic %s", topic)

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		defer n.Close()

		return nil
	},
}
