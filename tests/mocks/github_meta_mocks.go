package mocks

import (
	"ActQABot/pkg/worker_report"
	"context"
	"crypto/rand"
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
	mockLeaseCreate        *func(ctx context.Context) (*clientv3.LeaseID, error)
	mockLeaseRevoke        *func(ctx context.Context, leaseID clientv3.LeaseID)
	mockJobReportWatchFunc *func(ctx context.Context) worker_report.WatchWorkerReport
	mockJobCreateFunc      *func(
		ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID,
	) (interface{}, error)
}

func MockGithubMetaEtcd(mocks MockForGithubMetaEtcd) EtcdGithubMetaMock {
	jobs := make(map[string][]byte)
	lastLease := clientv3.LeaseID(0)
	revokedLeases := make(map[clientv3.LeaseID]bool)

	if mocks.mockLeaseCreate != nil {
		worker_report.GithubJobMetaLeaseCreateFunc = *mocks.mockLeaseCreate
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

	if mocks.mockLeaseRevoke != nil {
		worker_report.GithubJobMetaLeaseRevokeFunc = *mocks.mockLeaseRevoke
	} else {
		worker_report.GithubJobMetaLeaseRevokeFunc = func(ctx context.Context, leaseID clientv3.LeaseID) {
			revokedLeases[leaseID] = true
		}
	}

	if mocks.mockJobReportWatchFunc != nil {
		worker_report.JobReportInitWatchFunc = *mocks.mockJobReportWatchFunc
	} else {
		worker_report.JobReportInitWatchFunc = func(ctx context.Context) worker_report.WatchWorkerReport {
			return newMockedWatchWorkerReport()
		}
	}

	if mocks.mockJobCreateFunc != nil {
		worker_report.StoreGithubJobMetaFunc = *mocks.mockJobCreateFunc
	} else {
		worker_report.StoreGithubJobMetaFunc = func(
			ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID,
		) (interface{}, error) {
			jobs[jobId] = data
			return nil, nil
		}
	}
	return EtcdGithubMetaMock{LastLease: &lastLease, RevokedLeases: &revokedLeases, Jobs: jobs}
}

type EtcdWorkerReportMock struct {
	WorkerReportEventChannel worker_report.WatchWorkerReport
}

type mockedWatchWorkerReport struct {
	responseChannel chan *clientv3.WatchResponse
}

func (m mockedWatchWorkerReport) PushResponse(ctx context.Context, response *clientv3.WatchResponse) error {
	select {
	case m.responseChannel <- response:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m mockedWatchWorkerReport) GetWatchResponseChannel(ctx context.Context) <-chan *clientv3.WatchResponse {
	return m.responseChannel
}

func newMockedWatchWorkerReport() worker_report.WatchWorkerReport {
	return mockedWatchWorkerReport{responseChannel: make(chan *clientv3.WatchResponse, 5)}
}

func MockWorkerReportEtcd(
	mockWorkerReportSendFun *func(ctx context.Context, key string, value string) error,
	mockWorkerReportSubscribe *func(ctx context.Context) worker_report.WatchWorkerReport,
) EtcdWorkerReportMock {
	workerReportMock := EtcdWorkerReportMock{
		WorkerReportEventChannel: newMockedWatchWorkerReport(),
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
		worker_report.JobReportInitWatchFunc = func(ctx context.Context) worker_report.WatchWorkerReport {
			return workerReportMock.WorkerReportEventChannel
		}
	}
	worker_report.JobMakeAckFunc = func(key string, modRev int64) func(ctx context.Context) error {
		return nil
	}
	worker_report.JobMakeNackFunc = func(
		key string, report *worker_report.JobReport, delay time.Duration, maxRetries int,
	) func(ctx context.Context) error {
		return nil
	}
	return workerReportMock
}
