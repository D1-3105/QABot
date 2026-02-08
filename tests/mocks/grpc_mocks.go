package mocks

import (
	"ActQABot/conf"
	"ActQABot/internal/grpc_utils"
	"context"
	"fmt"
	"github.com/D1-3105/ActService/api/gen/ActService"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func GrpcConnFixture(t *testing.T) {
	original := grpc_utils.NewGRPCConn
	mockConn := &MockClientConn{
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
				return &MockClientStream{
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

type MockClientConn struct {
	InvokeFunc func(
		ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption,
	) error
	//
	NewStreamFunc func(
		ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption,
	) (grpc.ClientStream, error)
}

func (m *MockClientConn) Invoke(
	ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption,
) error {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, method, args, reply, opts...)
	}
	return nil
}

func (m *MockClientConn) NewStream(
	ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	if m.NewStreamFunc != nil {
		return m.NewStreamFunc(ctx, desc, method, opts...)
	}
	return nil, nil
}
