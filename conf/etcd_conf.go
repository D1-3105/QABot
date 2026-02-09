package conf

import (
	"ActQABot/internal/etcd_utils"
	"github.com/caarlos0/env/v11"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type EtcdConf struct {
	Endpoints   []string      `env:"ETCD_ENDPOINTS" envSeparator:"," envDefault:"127.0.0.1:2379"`
	DialTimeout time.Duration `env:"ETCD_DIAL_TIMEOUT" envDefault:"5s"`
}

func NewEtcdConfFromEnv() (*EtcdConf, error) {
	cfg := &EtcdConf{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if etcd_utils.EtcdStoreInstance == nil {
		cli, err := clientv3.New(
			clientv3.Config{
				Endpoints:   cfg.Endpoints,
				DialTimeout: cfg.DialTimeout,
			},
		)
		if err != nil {
			return nil, err
		}
		etcd_utils.EtcdStoreInstance = &etcd_utils.EtcdStore{Client: cli}
	}
	return cfg, nil
}
