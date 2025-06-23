package tests

import (
	"context"
	"fmt"
	actservice "github.com/D1-3105/ActService/api/gen/ActService"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"io"
)

type mockComment struct {
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

type issueCommentPayload struct {
	Action       string      `json:"action"`
	IssueComment mockComment `json:"comment"`
}

type mockClientStream struct {
	grpc.ClientStream
	recvCount int
	logs      []*actservice.JobLogMessage
}

func (m *mockClientStream) RecvMsg(msg interface{}) error {
	if m.recvCount >= len(m.logs) {
		return io.EOF
	}
	orig := m.logs[m.recvCount]
	m.recvCount++

	msgProto, ok := msg.(proto.Message)
	if !ok {
		return fmt.Errorf("unexpected message type %T: not proto.Message", msg)
	}

	proto.Reset(msgProto)
	proto.Merge(msgProto, orig)
	return nil
}

func (m *mockClientStream) SendMsg(_ interface{}) error {
	return nil
}

func (m *mockClientStream) CloseSend() error {
	return nil
}

type mockClientConn struct {
	InvokeFunc func(
		ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption,
	) error
	//
	NewStreamFunc func(
		ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption,
	) (grpc.ClientStream, error)
}

func (m *mockClientConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, method, args, reply, opts...)
	}
	return nil
}

func (m *mockClientConn) NewStream(
	ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	if m.NewStreamFunc != nil {
		return m.NewStreamFunc(ctx, desc, method, opts...)
	}
	return nil, nil
}
