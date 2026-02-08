package worker_report

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

type JobKeys string

const JobReportChannel JobKeys = "/job-report-chan/"

type JobReport struct {
	JobId         string `json:"job_id"`
	JobReportText string `json:"report_text"`
}

type JobReportEvent struct {
	Report *JobReport
	Ack    func(context.Context) error
	Nack   func(context.Context) error

	Finish sync.Once
}

func (j *JobReport) SendEvent(ctx context.Context) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	err = JobReportSendEventFunc(ctx, string(JobReportChannel)+j.JobId, string(data))
	return err
}

func SubscribeJobReports(ctx context.Context) (<-chan *JobReportEvent, error) {

	out := make(chan *JobReportEvent)

	go func() {
		defer close(out)

		for {
			rch := JobReportInitWatchFunc(ctx).GetWatchResponseChannel(ctx)
			for wresp := range rch {
				if wresp.Canceled {
					glog.Error("watch canceled, reconnecting:", wresp)
					break
				}

				for _, ev := range wresp.Events {
					if ev.Type != clientv3.EventTypePut {
						continue
					}

					var jr JobReport
					if err := json.Unmarshal(ev.Kv.Value, &jr); err != nil {
						glog.Error("failed to unmarshal JobReport:", err)
						continue
					}

					ack := JobMakeAckFunc(string(ev.Kv.Key), ev.Kv.ModRevision)
					nack := JobMakeNackFunc(
						string(ev.Kv.Key),
						&jr,
						10*time.Second,
						5,
					)

					event := &JobReportEvent{
						Report: &jr,
						Ack:    ack,
						Nack:   nack,
					}

					select {
					case out <- event:
					case <-ctx.Done():
						return
					}
				}
			}

			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}
