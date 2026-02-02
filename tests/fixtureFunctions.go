package tests

import (
	"ActQABot/conf"
	"ActQABot/internal/grpc_utils"
	"ActQABot/pkg/github/gh_api"
	"ActQABot/pkg/hosts"
	"ActQABot/pkg/worker_report"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/google/go-github/v60/github"
	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"math/big"
	"os"
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
	hosts.HostAvbl = hosts.NewAvailability(conf.Hosts)
}

type etcdGithubMetaMock struct {
	jobs          map[string][]byte
	lastLease     *clientv3.LeaseID
	revokedLeases *map[clientv3.LeaseID]bool
}

type mocksForGithubMetaEtcd struct {
	mockLeaseCreate        *func(ctx context.Context) (*clientv3.LeaseID, error)
	mockLeaseRevoke        *func(ctx context.Context, leaseID clientv3.LeaseID)
	mockJobReportWatchFunc *func(ctx context.Context) <-chan clientv3.WatchResponse
	mockJobCreateFunc      *func(
		ctx context.Context, jobId string, data []byte, leaseID clientv3.LeaseID,
	) (interface{}, error)
}

func mockGithubMetaEtcd(mocks mocksForGithubMetaEtcd) etcdGithubMetaMock {
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
		worker_report.JobReportInitWatchFunc = func(ctx context.Context) <-chan clientv3.WatchResponse {
			return make(chan clientv3.WatchResponse)
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
	return etcdGithubMetaMock{lastLease: &lastLease, revokedLeases: &revokedLeases, jobs: jobs}
}

func mockGithub() {
	gh_api.Authorize = func(ghEnv conf.GithubAPIEnvironment, owner, repo string) (*github.InstallationToken, error) {
		return &github.InstallationToken{Token: &testToken}, nil
	}
}

func postIssueCommentFixture(t *testing.T) chan *gh_api.BotResponse {
	mockGithub()
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
	t.Cleanup(
		func() {
			gh_api.PostIssueCommentFunc = original
		},
	)
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
	t.Cleanup(
		func() {
			grpc_utils.NewGRPCConn = original
		},
	)
}

func generateRSAPrivateKeyPEM(t *testing.T, filePath string, bits int) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	pemBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)

	err = os.WriteFile(filePath, pemBytes, 0600)
	if err != nil {
		t.Fatalf("failed to write private key file: %v", err)
	}
}
