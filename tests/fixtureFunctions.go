package tests

import (
	"ActQABot/conf"
	"ActQABot/internal/grpc_utils"
	"ActQABot/pkg/github/gh_api"
	"ActQABot/pkg/hosts"
	"context"
	"fmt"
	"github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"testing"
	"time"
)

var testToken = "test-token"

func setupTestEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HOST_CONF", "hosts.example.yaml")
	t.Setenv("GITHUB_TOKEN", "test-token")
	conf.NewEnviron(&conf.GeneralEnvironments)

	var err error
	conf.Hosts, err = conf.NewHostsEnvironment(conf.GeneralEnvironments.HostConf)
	if err != nil {
		t.Fatalf("failed to init conf.Hosts: %v", err)
	}
	conf.NewEnviron(&conf.GithubEnvironment)
	hosts.HostAvbl = hosts.NewAvailability(conf.Hosts)
}

func postIssueCommentFixture(t *testing.T) chan *gh_api.BotResponse {
	original := gh_api.PostIssueCommentFunc
	call := make(chan *gh_api.BotResponse, 1)
	gh_api.PostIssueCommentFunc = func(botComment *gh_api.BotResponse, token string) error {
		call <- botComment
		if botComment.Text == "" {
			t.Errorf("expected botComment.Text to be non-empty")
		}
		if token != testToken {
			t.Errorf("expected token to be %s, got %s", testToken, token)
		}
		return nil
	}
	t.Cleanup(func() {
		gh_api.PostIssueCommentFunc = original
	})
	return call
}

func grpcConnFixture(t *testing.T) {
	original := grpc_utils.NewGRPCConn
	mockConn := &mockClientConn{
		InvokeFunc: func(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
			if method == actservice.ActService_ScheduleActJob_FullMethodName {
				resp, ok := reply.(*actservice.JobResponse)
				if !ok {
					return fmt.Errorf("unexpected reply type")
				}
				resp.JobId = uuid.New().String()
				return nil
			}
			return nil
		},
		NewStreamFunc: func(
			ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			if method == actservice.ActService_JobLogStream_FullMethodName {
				return &mockClientStream{
					logs: []*actservice.JobLogMessage{
						{Timestamp: time.Now().Unix(), Line: "line1", Type: actservice.JobLogMessage_STDERR},
						{Timestamp: time.Now().Unix(), Line: "line2", Type: actservice.JobLogMessage_STDOUT},
					},
					recvCount: 0,
				}, nil
			}
			return nil, nil
		},
	}

	grpc_utils.NewGRPCConn = func(host conf.Host) (grpc.ClientConnInterface, error) {
		return mockConn, nil
	}
	t.Cleanup(func() {
		grpc_utils.NewGRPCConn = original
	})
}
