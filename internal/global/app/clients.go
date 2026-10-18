package app

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/natsx"
)

func InitEtcdClient() error {
	var err error
	EtcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:        gconfig.Cfg.Etcd.Endpoints,
		DialTimeout:      5 * time.Second,
		AutoSyncInterval: gconfig.Cfg.Etcd.AutoSyncInterval.Duration,
	})
	if err != nil {
		return err
	}
	return nil
}

func InitNatsClient() error {
	var err error
	NatsClient, err = natsx.Connect(gconfig.Cfg.Nats)
	if err != nil {
		return err
	}
	return nil
}
