package tests

import (
	"ActQABot/pkg/worker_report"
	"ActQABot/tests/mocks"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"strings"
	"testing"
	"time"
)

func Test_JobReportsConsumer(t *testing.T) {
	bg, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupTestEnv(t)
	reportMocks := mocks.MockWorkerReportEtcd(nil, nil)
	reportMockChannelGet := func(ctx context.Context, rev int64) worker_report.WatchWorkerReport {
		return reportMocks.WorkerReportEventChannel
	}
	_ = mocks.MockGithubMetaEtcd(
		mocks.MockForGithubMetaEtcd{
			MockJobReportWatchFunc: &reportMockChannelGet,
		},
	)
	commentPosted := mocks.PostIssueCommentFixture(t)
	subscribed, err := worker_report.SubscribeJobReports(bg)
	if err != nil {
		t.Errorf("Error subscribing job reports: %v", err)
	}
	go worker_report.JobReportsConsumer(bg, subscribed)
	// correct job
	ans := "Answer"
	jobId := uuid.New().String()
	newJobMeta := worker_report.GithubIssueMeta{
		Sender:            "user",
		Body:              "body",
		Owner:             "user",
		Repository:        "repo",
		IssueId:           1,
		AnswerCommentBody: &ans,
		Host:              "vm",
		JobId:             &jobId,
	}
	report := worker_report.JobReport{JobId: jobId, JobReportText: "Some text"}
	serializedReport, err := json.Marshal(report)
	if err != nil {
		t.Errorf("Failed to serilize report!")
	}
	if err = newJobMeta.Store(t.Context(), *newJobMeta.JobId, 1); err != nil {
		t.Errorf("Error storing job report: %v", err)
	}

	if err = reportMocks.WorkerReportEventChannel.PushResponse(
		bg,
		&clientv3.WatchResponse{
			Events: []*clientv3.Event{
				{
					Type: clientv3.EventTypePut,
					Kv: &mvccpb.KeyValue{
						Key:   []byte(jobId),
						Value: serializedReport,
					},
				},
			},
		},
	); err != nil {
		t.Errorf("Error pushing event: %v", err)
	}
	t.Log("Pushed a report")

	select {
	case <-t.Context().Done():
		return
	case <-time.After(time.Second * 5):
		t.Errorf("Timed out waiting for comment")
	case comment := <-commentPosted:
		require.Condition(
			t, func() (success bool) {
				answerCommentIsEmbed := strings.Contains(comment.Text, *newJobMeta.AnswerCommentBody)
				initialCommentIsEmbed := strings.Contains(comment.Text, newJobMeta.Body)
				return comment.Repo == newJobMeta.Repository && comment.Owner == newJobMeta.
					Owner && answerCommentIsEmbed && initialCommentIsEmbed
			},
		)
	}
	// no job created
	jobId = uuid.New().String()
	report = worker_report.JobReport{JobId: jobId, JobReportText: "Some text"}
	serializedReport, err = json.Marshal(report)
	if err != nil {
		t.Errorf("Failed to serilize report!")
	}
	if err = reportMocks.WorkerReportEventChannel.PushResponse(
		bg,
		&clientv3.WatchResponse{
			Events: []*clientv3.Event{
				{
					Type: clientv3.EventTypePut,
					Kv: &mvccpb.KeyValue{
						Key:   []byte(jobId),
						Value: serializedReport,
					},
				},
			},
		},
	); err != nil {
		t.Errorf("Error pushing event: %v", err)
	}
	select {
	case <-t.Context().Done():
		return
	case <-time.After(time.Second * 5):
		t.Logf("Timed out waiting for comment, valid!")
		break
	case comment := <-commentPosted:
		t.Errorf("Comment posted when not expected: %v", comment)
	}
}
