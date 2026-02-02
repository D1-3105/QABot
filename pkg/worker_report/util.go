package worker_report

import (
	"ActQABot/conf"
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var GithubJobMetaLeaseCreateFunc = leaseCreate
var GithubJobMetaLeaseRevokeFunc = leaseRevoke
var StoreGithubJobMetaFunc = storeGithubJobMeta

func leaseRevoke(ctx context.Context, leaseID clientv3.LeaseID) {
	_, _ = conf.EtcdStoreInstance.Client.Lease.Revoke(ctx, leaseID)
}

func leaseCreate(ctx context.Context) (*clientv3.LeaseID, error) {
	resp, err := conf.EtcdStoreInstance.Client.Lease.Grant(ctx, int64((10 * time.Hour).Seconds()))
	if err != nil {
		return nil, err
	}
	return &resp.ID, nil
}

func storeGithubJobMeta(ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID) (interface{}, error) {
	return conf.EtcdStoreInstance.Client.Put(
		ctx, GithubIssueMetaPrefix+jobId, string(data), clientv3.WithLease(leaseID),
	)
}
