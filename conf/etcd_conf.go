package conf

import (
	"context"
	"github.com/caarlos0/env/v11"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var EtcdStoreInstance *EtcdStore

type EtcdStore struct {
	Client *clientv3.Client
	ctx    context.Context
	cancel context.CancelFunc
}

type EtcdConf struct {
	Endpoints   []string      `env:"ETCD_ENDPOINTS" envSeparator:"," envDefault:"127.0.0.1:2379"`
	DialTimeout time.Duration `env:"ETCD_DIAL_TIMEOUT" envDefault:"5s"`
}

func NewEtcdConfFromEnv() (*EtcdConf, error) {
	cfg := &EtcdConf{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
