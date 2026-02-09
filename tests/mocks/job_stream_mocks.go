package mocks

import (
	"fmt"
	"github.com/D1-3105/ActService/api/gen/ActService"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"io"
)

type MockClientStream struct {
	grpc.ClientStream
	recvCount int
	logs      []*actservice.JobLogMessage
}

func (m *MockClientStream) RecvMsg(msg interface{}) error {
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

func (m *MockClientStream) SendMsg(_ interface{}) error {
	return nil
}

func (m *MockClientStream) CloseSend() error {
	return nil
}
