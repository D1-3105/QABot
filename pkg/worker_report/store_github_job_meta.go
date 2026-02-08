package worker_report

import (
	"ActQABot/conf"
	"context"
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

const GithubIssueMetaPrefix = "/github-issue-meta/"

type GithubIssueMeta struct {
	Sender            string            `json:"sender"`
	Body              string            `json:"body"`
	Owner             string            `json:"owner"`
	Repository        string            `json:"repository"`
	AnswerCommentBody *string           `json:"answer_comment_body"`
	IssueId           int               `json:"issue_id"`
	Host              string            `json:"host"`
	MyLeaseID         *clientv3.LeaseID `json:"lease"`
	JobId             *string           `json:"job_id"`
}

func (g *GithubIssueMeta) Store(ctx context.Context, jobId string, retries int64) error {
	metaCreated := false
	for retries++; retries > 0; retries-- {
		if g.MyLeaseID == nil {
			leaseID, err := GithubJobMetaLeaseCreateFunc(ctx)
			if err != nil {
				glog.Errorf("failed to store job meta on lease creation: %v, retries left: %d", err, retries-1)
				time.Sleep(10)
				continue
			}
			g.MyLeaseID = new(clientv3.LeaseID)
			*g.MyLeaseID = *leaseID
		}
		data, err := json.Marshal(g)
		if err != nil {
			GithubJobMetaLeaseRevokeFunc(ctx, *g.MyLeaseID)
			g.MyLeaseID = nil
			return err
		}
		_, err = StoreGithubJobMetaFunc(ctx, jobId, data, *g.MyLeaseID)
		if err != nil {
			glog.Errorf("failed to store job meta: %v, retries left: %d", err, retries-1)
			time.Sleep(10)
			continue
		}
		metaCreated = true
		break
	}
	if !metaCreated {
		if g.MyLeaseID != nil {
			GithubJobMetaLeaseRevokeFunc(ctx, *g.MyLeaseID)
		}
		return errors.New("failed to store job meta")
	}
	glog.V(1).Infof("successfully stored job meta: %v", g)
	return nil
}

func etcdRetrieveJobMeta(ctx context.Context, jobId string) (*GithubIssueMeta, error) {
	resp, err := conf.EtcdStoreInstance.Client.Get(ctx, GithubIssueMetaPrefix+jobId, nil)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return nil, nil
	}
	githubIssueMeta := &GithubIssueMeta{}
	err = json.Unmarshal(resp.Kvs[0].Value, githubIssueMeta)
	if err != nil {
		return nil, err
	}
	return githubIssueMeta, nil
}

func etcdDeleteJobMeta(ctx context.Context, jobId string, leaseID *clientv3.LeaseID) error {
	_, err := conf.EtcdStoreInstance.Client.Delete(ctx, GithubIssueMetaPrefix+jobId, nil)
	if err != nil {
		return err
	}
	if leaseID != nil {
		GithubJobMetaLeaseRevokeFunc(ctx, *leaseID)
	}
	return nil
}
