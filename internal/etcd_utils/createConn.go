package etcd_utils

import (
	"go.etcd.io/etcd/client/v3"
)

var EtcdStoreInstance *EtcdStore

type EtcdStore struct {
	Client *clientv3.Client
}
