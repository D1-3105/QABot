package worker_report

import (
	"ActQABot/conf"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

// github meta

var GithubJobMetaLeaseCreateFunc = leaseCreate
var GithubJobMetaLeaseRevokeFunc = leaseRevoke
var StoreGithubJobMetaFunc = storeGithubJobMeta
var RetrieveGithubJobMetaFunc = etcdRetrieveJobMeta
var DeleteGithubJobMetaFunc = etcdDeleteJobMeta

// Job report

var JobReportInitWatchFunc = initWatch
var JobReportSendEventFunc = putEtcdReport
var JobMakeAckFunc = makeAckFn
var JobMakeNackFunc = makeNackFunc

// Realization

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

func initWatch(ctx context.Context) WatchWorkerReport {
	rawWatch := conf.EtcdStoreInstance.Client.Watch(ctx, string(JobReportChannel), clientv3.WithPrefix())
	return NewEtcdWatchWorkerReport(rawWatch)
}

func putEtcdReport(ctx context.Context, key string, value string) error {
	_, err := conf.EtcdStoreInstance.Client.Put(
		ctx,
		key,
		value,
	)
	return err
}

func makeAckFn(
	key string,
	modRev int64,
) func(ctx context.Context) error {
	cli := conf.EtcdStoreInstance.Client
	return func(ctx context.Context) error {
		txn := cli.Txn(ctx).
			If(clientv3.Compare(clientv3.ModRevision(key), "=", modRev)).
			Then(clientv3.OpDelete(key))

		resp, err := txn.Commit()
		if err != nil {
			return err
		}
		if !resp.Succeeded {
			return fmt.Errorf("ack failed: key was modified or already acked: %s", key)
		}
		return nil
	}
}

func makeNackFunc(
	key string,
	report *JobReport,
	delay time.Duration,
	maxRetries int,
) func(ctx context.Context) error {
	retryCount := 0
	cli := conf.EtcdStoreInstance.Client
	return func(ctx context.Context) error {
		if retryCount >= maxRetries {
			glog.Errorf(
				"job %s exceeded max retries (%d), dropping",
				report.JobId,
				maxRetries,
			)
			return nil
		}

		retryCount++

		data, err := json.Marshal(report)
		if err != nil {
			return err
		}

		// rewrite with updated retry count → new ModRevision
		_, err = cli.Put(ctx, key, string(data))
		// backoff
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return err
	}
}
