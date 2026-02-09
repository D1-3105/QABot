package mocks

import (
	"ActQABot/pkg/worker_report"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"math/big"
	"time"
)

type EtcdGithubMetaMock struct {
	Jobs          map[string][]byte
	LastLease     *clientv3.LeaseID
	RevokedLeases *map[clientv3.LeaseID]bool
}

type MockForGithubMetaEtcd struct {
	MockLeaseCreate        *func(ctx context.Context) (*clientv3.LeaseID, error)
	MockLeaseRevoke        *func(ctx context.Context, leaseID clientv3.LeaseID)
	MockJobReportWatchFunc *func(ctx context.Context, revision int64) worker_report.WatchWorkerReport
	MockJobCreateFunc      *func(
		ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID,
	) (interface{}, error)
	MockJobRetrieveFunc *func(ctx context.Context, jobId string) (*worker_report.GithubIssueMeta, error)
}

func MockGithubMetaEtcd(mocks MockForGithubMetaEtcd) EtcdGithubMetaMock {
	jobs := make(map[string][]byte)
	lastLease := clientv3.LeaseID(0)
	revokedLeases := make(map[clientv3.LeaseID]bool)

	if mocks.MockLeaseCreate != nil {
		worker_report.GithubJobMetaLeaseCreateFunc = *mocks.MockLeaseCreate
	} else {
		worker_report.GithubJobMetaLeaseCreateFunc = func(ctx context.Context) (*clientv3.LeaseID, error) {
			maxLease := big.NewInt(50)
			randlease, _ := rand.Int(rand.Reader, maxLease)
			if randlease != nil {
				lastLease = clientv3.LeaseID(randlease.Int64())
			}
			if revokedLeases[lastLease] && randlease != nil {
				revokedLeases[lastLease] = false
			}
			return &lastLease, nil
		}
	}

	if mocks.MockLeaseRevoke != nil {
		worker_report.GithubJobMetaLeaseRevokeFunc = *mocks.MockLeaseRevoke
	} else {
		worker_report.GithubJobMetaLeaseRevokeFunc = func(ctx context.Context, leaseID clientv3.LeaseID) {
			revokedLeases[leaseID] = true
		}
	}

	if mocks.MockJobReportWatchFunc != nil {
		worker_report.JobReportInitWatchFunc = *mocks.MockJobReportWatchFunc
	} else {
		worker_report.JobReportInitWatchFunc = func(ctx context.Context, rev int64) worker_report.WatchWorkerReport {
			return newMockedWatchWorkerReport()
		}
	}

	if mocks.MockJobCreateFunc != nil {
		worker_report.StoreGithubJobMetaFunc = *mocks.MockJobCreateFunc
	} else {
		worker_report.StoreGithubJobMetaFunc = func(
			ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID,
		) (interface{}, error) {
			jobs[jobId] = data
			return nil, nil
		}
	}

	if mocks.MockJobRetrieveFunc != nil {
		worker_report.RetrieveGithubJobMetaFunc = *mocks.MockJobRetrieveFunc
	} else {
		worker_report.RetrieveGithubJobMetaFunc = func(
			ctx context.Context, jobId string,
		) (*worker_report.GithubIssueMeta, error) {
			jobSerialized, found := jobs[jobId]
			if !found {
				return nil, nil
			}
			job := &worker_report.GithubIssueMeta{}
			err := json.Unmarshal(jobSerialized, job)
			return job, err
		}
	}
	return EtcdGithubMetaMock{LastLease: &lastLease, RevokedLeases: &revokedLeases, Jobs: jobs}
}

type EtcdWorkerReportMock struct {
	WorkerReportEventChannel worker_report.WatchWorkerReport
	JobReportStatuses        map[string]bool
}

type mockedWatchWorkerReport struct {
	responseChannel chan *clientv3.WatchResponse
}

func (m mockedWatchWorkerReport) PushResponse(ctx context.Context, response *clientv3.WatchResponse) error {
	select {
	case m.responseChannel <- response:
		glog.V(2).Info("pushed a new WatchResponse")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m mockedWatchWorkerReport) GetWatchResponseChannel(_ context.Context) <-chan *clientv3.WatchResponse {
	return m.responseChannel
}

func newMockedWatchWorkerReport() worker_report.WatchWorkerReport {
	return mockedWatchWorkerReport{responseChannel: make(chan *clientv3.WatchResponse, 5)}
}

func MockWorkerReportEtcd(
	mockWorkerReportSendFun *func(ctx context.Context, key string, value string) error,
	mockWorkerReportSubscribe *func(ctx context.Context, rev int64) worker_report.WatchWorkerReport,
) EtcdWorkerReportMock {
	workerReportMock := EtcdWorkerReportMock{
		WorkerReportEventChannel: newMockedWatchWorkerReport(),
	}
	worker_report.JobReportFetchFunc = func(ctx context.Context) ([]*worker_report.JobReport, int64, error) {
		return make([]*worker_report.JobReport, 0), 1, nil
	}
	if mockWorkerReportSendFun != nil {
		worker_report.JobReportSendEventFunc = *mockWorkerReportSendFun
	} else {
		worker_report.JobReportSendEventFunc = func(ctx context.Context, key string, value string) error {
			ev := &clientv3.Event{
				Kv: &mvccpb.KeyValue{Value: []byte(value), Key: []byte(key)},
			}
			err := workerReportMock.WorkerReportEventChannel.PushResponse(
				ctx,
				&clientv3.WatchResponse{
					Events: []*clientv3.Event{ev},
				},
			)
			return err
		}
	}
	if mockWorkerReportSubscribe != nil {
		worker_report.JobReportInitWatchFunc = *mockWorkerReportSubscribe
	} else {
		worker_report.JobReportInitWatchFunc = func(ctx context.Context, rev int64) worker_report.WatchWorkerReport {
			return workerReportMock.WorkerReportEventChannel
		}
	}

	jobReportStatuses := make(map[string]bool)
	worker_report.JobMakeAckFunc = func(key string, modRev int64) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			reportStatus, found := jobReportStatuses[key]
			if found {
				return fmt.Errorf("%s was already marked as processed, status: %v", key, reportStatus)
			}
			jobReportStatuses[key] = true
			return nil
		}
	}
	worker_report.JobMakeNackFunc = func(
		key string, report *worker_report.JobReport, delay time.Duration,
	) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			reportStatus, found := jobReportStatuses[key]
			if found {
				return fmt.Errorf("%s was already marked as processed, status: %v", key, reportStatus)
			}
			jobReportStatuses[key] = false
			return nil
		}
	}
	workerReportMock.JobReportStatuses = jobReportStatuses
	return workerReportMock
}
