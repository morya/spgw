package natsx

import (
	"github.com/nats-io/nats.go"

	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
)

func Connect(cfg gconfig.NatsSetting) (*nats.Conn, error) {
	return nats.Connect(
		cfg.Address,
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait.Duration),
		nats.Timeout(cfg.Timeout.Duration),
		nats.RetryOnFailedConnect(true),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			glog.RunS().Warnf("nats disconnected: %v, reconnecting...", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			glog.RunS().Infof("nats reconnected to: %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			if err := nc.LastError(); err != nil {
				glog.RunS().Errorf("nats connection closed: %v", err)
			} else {
				glog.RunS().Errorf("nats connection closed")
			}
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			glog.RunS().Errorf("nats async error: %v", err)
		}),
	)
}

func MustConnect(cfg gconfig.NatsSetting) *nats.Conn {
	nc, err := Connect(cfg)
	if err != nil {
		glog.RunS().Fatalf("nats connect failed: %v", err)
	}
	return nc
}
