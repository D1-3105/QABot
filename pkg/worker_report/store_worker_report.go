package worker_report

import (
	"ActQABot/internal/etcd_utils"
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
	Retried       *int32 `json:"retried"`
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

func etcdFetchJobReports(ctx context.Context) ([]*JobReport, int64, error) {
	resp, err := etcd_utils.EtcdStoreInstance.Client.Get(ctx, string(JobReportChannel), clientv3.WithPrefix())
	if err != nil {
		return nil, 0, err
	}
	jobReports := make([]*JobReport, 0, len(resp.Kvs))
	for _, v := range resp.Kvs {
		var jr JobReport
		err := json.Unmarshal(v.Value, &jr)
		if err != nil {
			glog.Errorf("failed to unmarshal job report: %v", err)
			continue
		}
		jobReports = append(jobReports, &jr)
	}
	return jobReports, resp.Header.Revision, nil
}

func SubscribeJobReports(ctx context.Context) (<-chan *JobReportEvent, error) {

	wrapJobReportIntoEvent := func(k string, v *JobReport, rev int64) *JobReportEvent {
		ack := JobMakeAckFunc(k, rev)
		nack := JobMakeNackFunc(k, v, 10)

		event := &JobReportEvent{
			Report: v,
			Ack:    ack,
			Nack:   nack,
		}
		return event
	}

	out := make(chan *JobReportEvent)

	go func() {
		glog.V(1).Info("SubscribeJobReports started...")
		defer glog.V(1).Info("SubscribeJobReports finished...")
		defer close(out)

		oldJobReports, activeRev, err := JobReportFetchFunc(ctx)

		if err != nil {
			glog.Errorf("failed to fetch job reports: %v", err)
		} else {
			for _, oldJobReport := range oldJobReports {
				ev := wrapJobReportIntoEvent(oldJobReport.JobId, oldJobReport, activeRev)
				out <- ev
			}
		}
		for {
			rch := JobReportInitWatchFunc(ctx, activeRev+1).GetWatchResponseChannel(ctx)
			for wresp := range rch {
				if wresp.Canceled {
					glog.Error("watch canceled, reconnecting:", wresp)
					break
				}

				for _, ev := range wresp.Events {
					if ev.Type != clientv3.EventTypePut {
						glog.V(2).Infof("SubscribeJobReports - event watch response received for event %v", ev.Type)
						continue
					}

					var jr JobReport
					if err := json.Unmarshal(ev.Kv.Value, &jr); err != nil {
						glog.Error("failed to unmarshal JobReport:", err)
						continue
					}

					event := wrapJobReportIntoEvent(string(ev.Kv.Key), &jr, ev.Kv.ModRevision)
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
