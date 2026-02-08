package worker_report

import (
	"context"
	"errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type WatchWorkerReport interface {
	GetWatchResponseChannel(context.Context) <-chan *clientv3.WatchResponse
	PushResponse(context.Context, *clientv3.WatchResponse) error
}

type EtcdWatchWorkerReport struct {
	clientv3.WatchChan
}

func (e EtcdWatchWorkerReport) PushResponse(context.Context, *clientv3.WatchResponse) error {
	return errors.New("not implemented")
}

func (e EtcdWatchWorkerReport) GetWatchResponseChannel(ctx context.Context) <-chan *clientv3.WatchResponse {
	readableChannel := make(chan *clientv3.WatchResponse, 5)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(readableChannel)
				break
			case wresp := <-e.WatchChan:
				readableChannel <- &wresp
				continue
			}
		}

	}()
	return readableChannel
}

func NewEtcdWatchWorkerReport(watch clientv3.WatchChan) WatchWorkerReport {
	return EtcdWatchWorkerReport{watch}
}
