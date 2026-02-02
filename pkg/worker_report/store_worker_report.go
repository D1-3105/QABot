package worker_report

import (
	"ActQABot/conf"
	"context"
	"encoding/json"
	"github.com/golang/glog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type JobKeys string

const JobReportChannel JobKeys = "/job-report-chan/"

var JobReportInitWatchFunc = initWatch

func initWatch(ctx context.Context) <-chan clientv3.WatchResponse {
	return conf.EtcdStoreInstance.Client.Watch(ctx, string(JobReportChannel), clientv3.WithPrefix())
}

type JobReport struct {
	JobId         string
	JobReportText string
}

func (j *JobReport) SendEvent(ctx context.Context) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	_, err = conf.EtcdStoreInstance.Client.Put(
		ctx,
		string(JobReportChannel)+j.JobId,
		string(data),
	)
	return err
}

func SubscribeJobReports(ctx context.Context) (<-chan *JobReport, error) {
	newRch := make(chan *JobReport)

	go func() {
		defer close(newRch)
		for {
			rch := JobReportInitWatchFunc(ctx)
			for wresp := range rch {
				if wresp.Canceled {
					glog.Error("watch canceled, reconnecting:", wresp)
					break
				}
				for _, ev := range wresp.Events {
					var jr JobReport
					if err := json.Unmarshal(ev.Kv.Value, &jr); err != nil {
						glog.Error("failed to unmarshal JobReport:", err)
						continue
					}
					select {
					case newRch <- &jr:
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
	return newRch, nil
}
