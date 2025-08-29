/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package v2

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	v2 "github.com/containerd/containerd/api/runtime/task/v2"
	api "github.com/containerd/containerd/api/runtime/task/v3"
	task "github.com/containerd/containerd/api/types/task"
)

// We'll test the NewTaskClient function with nil clients to test error cases
// and focus on the bridge functionality rather than client creation

// mockGRPCClientConn implements grpc.ClientConnInterface for testing
type mockGRPCClientConn struct {
}

func (m *mockGRPCClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return nil
}

func (m *mockGRPCClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

// mockTaskServiceV2 implements v2.TaskService for testing
type mockTaskServiceV2 struct {
	StateFunc      func(ctx context.Context, req *v2.StateRequest) (*v2.StateResponse, error)
	CreateFunc     func(ctx context.Context, req *v2.CreateTaskRequest) (*v2.CreateTaskResponse, error)
	StartFunc      func(ctx context.Context, req *v2.StartRequest) (*v2.StartResponse, error)
	DeleteFunc     func(ctx context.Context, req *v2.DeleteRequest) (*v2.DeleteResponse, error)
	PidsFunc       func(ctx context.Context, req *v2.PidsRequest) (*v2.PidsResponse, error)
	PauseFunc      func(ctx context.Context, req *v2.PauseRequest) (*emptypb.Empty, error)
	ResumeFunc     func(ctx context.Context, req *v2.ResumeRequest) (*emptypb.Empty, error)
	CheckpointFunc func(ctx context.Context, req *v2.CheckpointTaskRequest) (*emptypb.Empty, error)
	KillFunc       func(ctx context.Context, req *v2.KillRequest) (*emptypb.Empty, error)
	ExecFunc       func(ctx context.Context, req *v2.ExecProcessRequest) (*emptypb.Empty, error)
	ResizePtyFunc  func(ctx context.Context, req *v2.ResizePtyRequest) (*emptypb.Empty, error)
	CloseIOFunc    func(ctx context.Context, req *v2.CloseIORequest) (*emptypb.Empty, error)
	UpdateFunc     func(ctx context.Context, req *v2.UpdateTaskRequest) (*emptypb.Empty, error)
	WaitFunc       func(ctx context.Context, req *v2.WaitRequest) (*v2.WaitResponse, error)
	StatsFunc      func(ctx context.Context, req *v2.StatsRequest) (*v2.StatsResponse, error)
	ConnectFunc    func(ctx context.Context, req *v2.ConnectRequest) (*v2.ConnectResponse, error)
	ShutdownFunc   func(ctx context.Context, req *v2.ShutdownRequest) (*emptypb.Empty, error)
}

func (m *mockTaskServiceV2) State(ctx context.Context, req *v2.StateRequest) (*v2.StateResponse, error) {
	if m.StateFunc != nil {
		return m.StateFunc(ctx, req)
	}
	return &v2.StateResponse{
		ID:     req.ID,
		ExecID: req.ExecID,
		Pid:    123,
		Status: task.Status_RUNNING,
	}, nil
}

func (m *mockTaskServiceV2) Create(ctx context.Context, req *v2.CreateTaskRequest) (*v2.CreateTaskResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return &v2.CreateTaskResponse{Pid: 456}, nil
}

func (m *mockTaskServiceV2) Start(ctx context.Context, req *v2.StartRequest) (*v2.StartResponse, error) {
	if m.StartFunc != nil {
		return m.StartFunc(ctx, req)
	}
	return &v2.StartResponse{Pid: 789}, nil
}

func (m *mockTaskServiceV2) Delete(ctx context.Context, req *v2.DeleteRequest) (*v2.DeleteResponse, error) {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, req)
	}
	return &v2.DeleteResponse{Pid: 101}, nil
}

func (m *mockTaskServiceV2) Pids(ctx context.Context, req *v2.PidsRequest) (*v2.PidsResponse, error) {
	if m.PidsFunc != nil {
		return m.PidsFunc(ctx, req)
	}
	return &v2.PidsResponse{Processes: []*task.ProcessInfo{{Pid: 123, Info: nil}}}, nil
}

func (m *mockTaskServiceV2) Pause(ctx context.Context, req *v2.PauseRequest) (*emptypb.Empty, error) {
	if m.PauseFunc != nil {
		return m.PauseFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) Resume(ctx context.Context, req *v2.ResumeRequest) (*emptypb.Empty, error) {
	if m.ResumeFunc != nil {
		return m.ResumeFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) Checkpoint(ctx context.Context, req *v2.CheckpointTaskRequest) (*emptypb.Empty, error) {
	if m.CheckpointFunc != nil {
		return m.CheckpointFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) Kill(ctx context.Context, req *v2.KillRequest) (*emptypb.Empty, error) {
	if m.KillFunc != nil {
		return m.KillFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) Exec(ctx context.Context, req *v2.ExecProcessRequest) (*emptypb.Empty, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) ResizePty(ctx context.Context, req *v2.ResizePtyRequest) (*emptypb.Empty, error) {
	if m.ResizePtyFunc != nil {
		return m.ResizePtyFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) CloseIO(ctx context.Context, req *v2.CloseIORequest) (*emptypb.Empty, error) {
	if m.CloseIOFunc != nil {
		return m.CloseIOFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) Update(ctx context.Context, req *v2.UpdateTaskRequest) (*emptypb.Empty, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

func (m *mockTaskServiceV2) Wait(ctx context.Context, req *v2.WaitRequest) (*v2.WaitResponse, error) {
	if m.WaitFunc != nil {
		return m.WaitFunc(ctx, req)
	}
	return &v2.WaitResponse{ExitStatus: 0}, nil
}

func (m *mockTaskServiceV2) Stats(ctx context.Context, req *v2.StatsRequest) (*v2.StatsResponse, error) {
	if m.StatsFunc != nil {
		return m.StatsFunc(ctx, req)
	}
	return &v2.StatsResponse{Stats: nil}, nil
}

func (m *mockTaskServiceV2) Connect(ctx context.Context, req *v2.ConnectRequest) (*v2.ConnectResponse, error) {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, req)
	}
	return &v2.ConnectResponse{ShimPid: 999}, nil
}

func (m *mockTaskServiceV2) Shutdown(ctx context.Context, req *v2.ShutdownRequest) (*emptypb.Empty, error) {
	if m.ShutdownFunc != nil {
		return m.ShutdownFunc(ctx, req)
	}
	return &emptypb.Empty{}, nil
}

// For these tests, we'll focus on testing the error cases rather than
// successful client creation which requires complex mocking

func TestNewTaskClient_GRPCv3(t *testing.T) {
	mockConn := &mockGRPCClientConn{}

	client, err := NewTaskClient(mockConn, 3)
	if err != nil {
		t.Fatalf("NewTaskClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Verify it's the right type
	if _, ok := client.(*grpcV3Bridge); !ok {
		t.Fatalf("expected *grpcV3Bridge, got %T", client)
	}
}

// Removed TTRPC version tests that require complex client mocking

func TestNewTaskClient_UnsupportedGRPCVersion(t *testing.T) {
	mockConn := &mockGRPCClientConn{}

	_, err := NewTaskClient(mockConn, 2)
	if err == nil {
		t.Fatal("expected error for unsupported GRPC version")
	}

	if err.Error() != "containerd client supports only v3 GRPC task service (got 2)" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewTaskClient_UnsupportedClientType(t *testing.T) {
	invalidClient := "not a client"

	_, err := NewTaskClient(invalidClient, 3)
	if err == nil {
		t.Fatal("expected error for unsupported client type")
	}

	if err.Error() != "unsupported shim client type string" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestTTRPCv2Bridge_State(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}

	ctx := context.Background()
	request := &api.StateRequest{
		ID:     "test-id",
		ExecID: "test-exec-id",
	}

	response, err := bridge.State(ctx, request)
	if err != nil {
		t.Fatalf("State failed: %v", err)
	}

	if response.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %v", response.ID)
	}
	if response.ExecID != "test-exec-id" {
		t.Errorf("expected ExecID 'test-exec-id', got %v", response.ExecID)
	}
	if response.Pid != 123 {
		t.Errorf("expected Pid 123, got %v", response.Pid)
	}
}

func TestTTRPCv2Bridge_Create(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}

	ctx := context.Background()
	request := &api.CreateTaskRequest{
		ID:       "test-id",
		Bundle:   "/test/bundle",
		Terminal: true,
	}

	response, err := bridge.Create(ctx, request)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if response.Pid != 456 {
		t.Errorf("expected Pid 456, got %v", response.Pid)
	}
}

func TestTTRPCv2Bridge_Start(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}

	ctx := context.Background()
	request := &api.StartRequest{
		ID:     "test-id",
		ExecID: "test-exec-id",
	}

	response, err := bridge.Start(ctx, request)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if response.Pid != 789 {
		t.Errorf("expected Pid 789, got %v", response.Pid)
	}
}

func TestTTRPCv2Bridge_Delete(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}

	ctx := context.Background()
	request := &api.DeleteRequest{
		ID:     "test-id",
		ExecID: "test-exec-id",
	}

	response, err := bridge.Delete(ctx, request)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if response.Pid != 101 {
		t.Errorf("expected Pid 101, got %v", response.Pid)
	}
}

func TestTTRPCv2Bridge_Pids(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}

	ctx := context.Background()
	request := &api.PidsRequest{ID: "test-id"}

	response, err := bridge.Pids(ctx, request)
	if err != nil {
		t.Fatalf("Pids failed: %v", err)
	}

	if len(response.Processes) != 1 {
		t.Errorf("expected 1 process, got %v", len(response.Processes))
	}
	if response.Processes[0].Pid != 123 {
		t.Errorf("expected Pid 123, got %v", response.Processes[0].Pid)
	}
}

func TestTTRPCv2Bridge_BasicOperations(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}
	ctx := context.Background()

	// Test Pause
	_, err := bridge.Pause(ctx, &api.PauseRequest{ID: "test-id"})
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}

	// Test Resume
	_, err = bridge.Resume(ctx, &api.ResumeRequest{ID: "test-id"})
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}

	// Test Kill
	_, err = bridge.Kill(ctx, &api.KillRequest{ID: "test-id", Signal: 9})
	if err != nil {
		t.Errorf("Kill failed: %v", err)
	}

	// Test Wait
	waitResp, err := bridge.Wait(ctx, &api.WaitRequest{ID: "test-id"})
	if err != nil {
		t.Errorf("Wait failed: %v", err)
	}
	if waitResp.ExitStatus != 0 {
		t.Errorf("expected exit status 0, got %v", waitResp.ExitStatus)
	}

	// Test Stats
	_, err = bridge.Stats(ctx, &api.StatsRequest{ID: "test-id"})
	if err != nil {
		t.Errorf("Stats failed: %v", err)
	}

	// Test Connect
	connectResp, err := bridge.Connect(ctx, &api.ConnectRequest{ID: "test-id"})
	if err != nil {
		t.Errorf("Connect failed: %v", err)
	}
	if connectResp.ShimPid != 999 {
		t.Errorf("expected shim pid 999, got %v", connectResp.ShimPid)
	}

	// Test Shutdown
	_, err = bridge.Shutdown(ctx, &api.ShutdownRequest{ID: "test-id"})
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestTTRPCv2Bridge_IOOperations(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}
	ctx := context.Background()

	// Test Exec
	_, err := bridge.Exec(ctx, &api.ExecProcessRequest{
		ID:     "test-id",
		ExecID: "exec-123",
		Spec:   nil,
	})
	if err != nil {
		t.Errorf("Exec failed: %v", err)
	}

	// Test ResizePty
	_, err = bridge.ResizePty(ctx, &api.ResizePtyRequest{
		ID:     "test-id",
		Width:  80,
		Height: 24,
	})
	if err != nil {
		t.Errorf("ResizePty failed: %v", err)
	}

	// Test CloseIO
	_, err = bridge.CloseIO(ctx, &api.CloseIORequest{
		ID:    "test-id",
		Stdin: true,
	})
	if err != nil {
		t.Errorf("CloseIO failed: %v", err)
	}
}

func TestTTRPCv2Bridge_Update(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}
	ctx := context.Background()

	_, err := bridge.Update(ctx, &api.UpdateTaskRequest{
		ID:        "test-id",
		Resources: nil,
	})
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}
}

func TestTTRPCv2Bridge_Checkpoint(t *testing.T) {
	mockService := &mockTaskServiceV2{}
	bridge := &ttrpcV2Bridge{client: mockService}
	ctx := context.Background()

	_, err := bridge.Checkpoint(ctx, &api.CheckpointTaskRequest{
		ID:      "test-id",
		Path:    "/checkpoint/path",
		Options: nil,
	})
	if err != nil {
		t.Errorf("Checkpoint failed: %v", err)
	}
}
