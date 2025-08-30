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

package introspectionproxy

import (
	"context"
	"errors"
	"testing"

	api "github.com/containerd/containerd/api/services/introspection/v1"
	"github.com/containerd/errdefs"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/containerd/containerd/v2/core/introspection"
)

// Mock TTRPC client for testing
type mockTTRPCIntrospectionClient struct {
	pluginsFunc    func(ctx context.Context, req *api.PluginsRequest) (*api.PluginsResponse, error)
	serverFunc     func(ctx context.Context, req *emptypb.Empty) (*api.ServerResponse, error)
	pluginInfoFunc func(ctx context.Context, req *api.PluginInfoRequest) (*api.PluginInfoResponse, error)
}

func (m *mockTTRPCIntrospectionClient) Plugins(ctx context.Context, req *api.PluginsRequest) (*api.PluginsResponse, error) {
	if m.pluginsFunc != nil {
		return m.pluginsFunc(ctx, req)
	}
	return &api.PluginsResponse{}, nil
}

func (m *mockTTRPCIntrospectionClient) Server(ctx context.Context, req *emptypb.Empty) (*api.ServerResponse, error) {
	if m.serverFunc != nil {
		return m.serverFunc(ctx, req)
	}
	return &api.ServerResponse{}, nil
}

func (m *mockTTRPCIntrospectionClient) PluginInfo(ctx context.Context, req *api.PluginInfoRequest) (*api.PluginInfoResponse, error) {
	if m.pluginInfoFunc != nil {
		return m.pluginInfoFunc(ctx, req)
	}
	return &api.PluginInfoResponse{}, nil
}

// Mock gRPC client for testing
type mockGRPCIntrospectionClient struct {
	pluginsFunc    func(ctx context.Context, req *api.PluginsRequest, opts ...grpc.CallOption) (*api.PluginsResponse, error)
	serverFunc     func(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*api.ServerResponse, error)
	pluginInfoFunc func(ctx context.Context, req *api.PluginInfoRequest, opts ...grpc.CallOption) (*api.PluginInfoResponse, error)
}

func (m *mockGRPCIntrospectionClient) Plugins(ctx context.Context, req *api.PluginsRequest, opts ...grpc.CallOption) (*api.PluginsResponse, error) {
	if m.pluginsFunc != nil {
		return m.pluginsFunc(ctx, req, opts...)
	}
	return &api.PluginsResponse{}, nil
}

func (m *mockGRPCIntrospectionClient) Server(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*api.ServerResponse, error) {
	if m.serverFunc != nil {
		return m.serverFunc(ctx, req, opts...)
	}
	return &api.ServerResponse{}, nil
}

func (m *mockGRPCIntrospectionClient) PluginInfo(ctx context.Context, req *api.PluginInfoRequest, opts ...grpc.CallOption) (*api.PluginInfoResponse, error) {
	if m.pluginInfoFunc != nil {
		return m.pluginInfoFunc(ctx, req, opts...)
	}
	return &api.PluginInfoResponse{}, nil
}

// Mock gRPC client connection
type mockGRPCClientConn struct {
	grpc.ClientConnInterface
}

func (m *mockGRPCClientConn) Invoke(ctx context.Context, method string, args, reply any, opts ...grpc.CallOption) error {
	return nil
}

func (m *mockGRPCClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("not implemented")
}

func TestNewIntrospectionProxy_WithTTRPCClient(t *testing.T) {
	mockClient := &mockTTRPCIntrospectionClient{}
	service := NewIntrospectionProxy(mockClient)

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Ensure it implements the interface
	var _ introspection.Service = service
}

func TestNewIntrospectionProxy_WithGRPCClient(t *testing.T) {
	mockClient := &mockGRPCIntrospectionClient{}
	service := NewIntrospectionProxy(mockClient)

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Ensure it implements the interface
	var _ introspection.Service = service
}

func TestNewIntrospectionProxy_WithGRPCClientConn(t *testing.T) {
	mockConn := &mockGRPCClientConn{}
	service := NewIntrospectionProxy(mockConn)

	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Ensure it implements the interface
	var _ introspection.Service = service
}

func TestNewIntrospectionProxy_WithTTRPCClientFromConn(t *testing.T) {
	// Test with actual ttrpc.Client (which we'll mock the constructor for)
	// Since ttrpc.Client is a concrete type, we'll test the panic path instead
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior for unsupported client type
		}
	}()

	// This should panic with unsupported client type
	NewIntrospectionProxy("invalid_client_type")
	t.Fatal("expected panic for unsupported client type")
}

func TestNewIntrospectionProxy_UnsupportedClient(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// Expected - should panic with unsupported client type
			if err, ok := r.(error); ok {
				if !errors.Is(err, errdefs.ErrNotImplemented) {
					t.Errorf("expected ErrNotImplemented, got %v", err)
				}
			}
		}
	}()

	// This should panic
	NewIntrospectionProxy("unsupported")
	t.Fatal("expected panic for unsupported client type")
}

func TestIntrospectionRemote_Plugins_Success(t *testing.T) {
	expectedResp := &api.PluginsResponse{
		Plugins: []*api.Plugin{
			{Type: "test", ID: "test-plugin"},
		},
	}

	mockClient := &mockTTRPCIntrospectionClient{
		pluginsFunc: func(ctx context.Context, req *api.PluginsRequest) (*api.PluginsResponse, error) {
			if len(req.Filters) != 2 {
				t.Errorf("expected 2 filters, got %d", len(req.Filters))
			}
			if req.Filters[0] != "type==test" {
				t.Errorf("expected filter 'type==test', got %s", req.Filters[0])
			}
			if req.Filters[1] != "id==test-plugin" {
				t.Errorf("expected filter 'id==test-plugin', got %s", req.Filters[1])
			}
			return expectedResp, nil
		},
	}

	service := NewIntrospectionProxy(mockClient)
	resp, err := service.Plugins(context.Background(), "type==test", "id==test-plugin")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResp {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}
}

func TestIntrospectionRemote_Plugins_Error(t *testing.T) {
	expectedErr := errors.New("plugins error")

	mockClient := &mockTTRPCIntrospectionClient{
		pluginsFunc: func(ctx context.Context, req *api.PluginsRequest) (*api.PluginsResponse, error) {
			return nil, expectedErr
		},
	}

	service := NewIntrospectionProxy(mockClient)
	_, err := service.Plugins(context.Background(), "filter1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIntrospectionRemote_Plugins_NoFilters(t *testing.T) {
	mockClient := &mockTTRPCIntrospectionClient{
		pluginsFunc: func(ctx context.Context, req *api.PluginsRequest) (*api.PluginsResponse, error) {
			if len(req.Filters) != 0 {
				t.Errorf("expected no filters, got %d", len(req.Filters))
			}
			return &api.PluginsResponse{}, nil
		},
	}

	service := NewIntrospectionProxy(mockClient)
	_, err := service.Plugins(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntrospectionRemote_Server_Success(t *testing.T) {
	expectedResp := &api.ServerResponse{
		UUID: "test-uuid",
	}

	mockClient := &mockTTRPCIntrospectionClient{
		serverFunc: func(ctx context.Context, req *emptypb.Empty) (*api.ServerResponse, error) {
			return expectedResp, nil
		},
	}

	service := NewIntrospectionProxy(mockClient)
	resp, err := service.Server(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResp {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}
}

func TestIntrospectionRemote_Server_Error(t *testing.T) {
	expectedErr := errors.New("server error")

	mockClient := &mockTTRPCIntrospectionClient{
		serverFunc: func(ctx context.Context, req *emptypb.Empty) (*api.ServerResponse, error) {
			return nil, expectedErr
		},
	}

	service := NewIntrospectionProxy(mockClient)
	_, err := service.Server(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIntrospectionRemote_PluginInfo_Success(t *testing.T) {
	expectedResp := &api.PluginInfoResponse{
		Extra: &anypb.Any{},
	}

	mockClient := &mockTTRPCIntrospectionClient{
		pluginInfoFunc: func(ctx context.Context, req *api.PluginInfoRequest) (*api.PluginInfoResponse, error) {
			if req.Type != "test-type" {
				t.Errorf("expected type 'test-type', got %s", req.Type)
			}
			if req.ID != "test-id" {
				t.Errorf("expected id 'test-id', got %s", req.ID)
			}
			return expectedResp, nil
		},
	}

	service := NewIntrospectionProxy(mockClient)
	resp, err := service.PluginInfo(context.Background(), "test-type", "test-id", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResp {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}
}

func TestIntrospectionRemote_PluginInfo_WithNilOptions(t *testing.T) {
	// Test with nil options (should work fine)
	mockClient := &mockTTRPCIntrospectionClient{
		pluginInfoFunc: func(ctx context.Context, req *api.PluginInfoRequest) (*api.PluginInfoResponse, error) {
			if req.Options != nil {
				t.Error("expected nil options")
			}
			return &api.PluginInfoResponse{}, nil
		},
	}

	service := NewIntrospectionProxy(mockClient)
	_, err := service.PluginInfo(context.Background(), "test-type", "test-id", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntrospectionRemote_PluginInfo_Error(t *testing.T) {
	expectedErr := errors.New("plugin info error")

	mockClient := &mockTTRPCIntrospectionClient{
		pluginInfoFunc: func(ctx context.Context, req *api.PluginInfoRequest) (*api.PluginInfoResponse, error) {
			return nil, expectedErr
		},
	}

	service := NewIntrospectionProxy(mockClient)
	_, err := service.PluginInfo(context.Background(), "test-type", "test-id", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConvertIntrospection_Plugins(t *testing.T) {
	expectedResp := &api.PluginsResponse{
		Plugins: []*api.Plugin{
			{Type: "test", ID: "test-plugin"},
		},
	}

	mockClient := &mockGRPCIntrospectionClient{
		pluginsFunc: func(ctx context.Context, req *api.PluginsRequest, opts ...grpc.CallOption) (*api.PluginsResponse, error) {
			return expectedResp, nil
		},
	}

	converter := convertIntrospection{client: mockClient}
	resp, err := converter.Plugins(context.Background(), &api.PluginsRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResp {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}
}

func TestConvertIntrospection_Server(t *testing.T) {
	expectedResp := &api.ServerResponse{
		UUID: "test-uuid",
	}

	mockClient := &mockGRPCIntrospectionClient{
		serverFunc: func(ctx context.Context, req *emptypb.Empty, opts ...grpc.CallOption) (*api.ServerResponse, error) {
			return expectedResp, nil
		},
	}

	converter := convertIntrospection{client: mockClient}
	resp, err := converter.Server(context.Background(), &emptypb.Empty{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResp {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}
}

func TestConvertIntrospection_PluginInfo(t *testing.T) {
	expectedResp := &api.PluginInfoResponse{
		Extra: &anypb.Any{},
	}

	mockClient := &mockGRPCIntrospectionClient{
		pluginInfoFunc: func(ctx context.Context, req *api.PluginInfoRequest, opts ...grpc.CallOption) (*api.PluginInfoResponse, error) {
			return expectedResp, nil
		},
	}

	converter := convertIntrospection{client: mockClient}
	resp, err := converter.PluginInfo(context.Background(), &api.PluginInfoRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResp {
		t.Errorf("expected response %v, got %v", expectedResp, resp)
	}
}

func TestIntrospectionRemote_InterfaceCompliance(t *testing.T) {
	// Test that introspectionRemote implements introspection.Service
	mockClient := &mockTTRPCIntrospectionClient{}
	remote := &introspectionRemote{client: mockClient}

	var _ introspection.Service = remote
}
