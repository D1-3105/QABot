package worker_report

import (
	"ActQABot/internal/etcd_utils"
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

// var DeleteGithubJobMetaFunc = etcdDeleteJobMeta

// Job report

var JobReportInitWatchFunc = initWatch
var JobReportFetchFunc = etcdFetchJobReports
var JobReportSendEventFunc = putEtcdReport
var JobMakeAckFunc = makeAckFn
var JobMakeNackFunc = makeNackFunc

const maxNacks = 10

// Realization

func leaseRevoke(ctx context.Context, leaseID clientv3.LeaseID) {
	if etcd_utils.EtcdStoreInstance == nil || etcd_utils.EtcdStoreInstance.Client == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	_, _ = etcd_utils.EtcdStoreInstance.Client.Lease.Revoke(ctx, leaseID)
}

func leaseCreate(ctx context.Context) (*clientv3.LeaseID, error) {
	if etcd_utils.EtcdStoreInstance == nil || etcd_utils.EtcdStoreInstance.Client == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	resp, err := etcd_utils.EtcdStoreInstance.Client.Lease.Grant(ctx, int64((10 * time.Hour).Seconds()))
	if err != nil {
		return nil, err
	}
	return &resp.ID, nil
}

func storeGithubJobMeta(ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID) (interface{}, error) {
	if etcd_utils.EtcdStoreInstance == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	return etcd_utils.EtcdStoreInstance.Client.Put(
		ctx, GithubIssueMetaPrefix+jobId, string(data), clientv3.WithLease(leaseID),
	)
}

func initWatch(ctx context.Context, revision int64) WatchWorkerReport {
	if etcd_utils.EtcdStoreInstance == nil || etcd_utils.EtcdStoreInstance.Client == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	rawWatch := etcd_utils.EtcdStoreInstance.Client.Watch(
		ctx,
		string(JobReportChannel),
		clientv3.WithPrefix(),
		clientv3.WithRev(revision),
	)
	return NewEtcdWatchWorkerReport(rawWatch)
}

func putEtcdReport(ctx context.Context, key string, value string) error {
	if etcd_utils.EtcdStoreInstance == nil || etcd_utils.EtcdStoreInstance.Client == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	_, err := etcd_utils.EtcdStoreInstance.Client.Put(
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
	if etcd_utils.EtcdStoreInstance == nil || etcd_utils.EtcdStoreInstance.Client == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	cli := etcd_utils.EtcdStoreInstance.Client
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
) func(ctx context.Context) error {
	if etcd_utils.EtcdStoreInstance == nil || etcd_utils.EtcdStoreInstance.Client == nil {
		glog.Fatal("etcd_store instance is nil")
	}
	cli := etcd_utils.EtcdStoreInstance.Client
	return func(ctx context.Context) error {
		if report.Retried != nil && *report.Retried > maxNacks {
			glog.Errorf("job %s was rejected too many times, dropping", report.JobId)
			_, err := etcd_utils.EtcdStoreInstance.Client.Delete(ctx, key, clientv3.WithPrevKV())
			return err
		} else if report.Retried == nil {
			report.Retried = new(int32)
			*report.Retried = 0
		}
		// backoff
		if delay > 0 {
			select {
			case <-time.After(time.Second * delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		*report.Retried++
		data, err := json.Marshal(report)
		if err != nil {
			return err
		}
		// rewrite with updated retry count → new ModRevision
		_, err = cli.Put(ctx, key, string(data))
		return err
	}
}
