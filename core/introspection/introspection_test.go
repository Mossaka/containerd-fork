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

package introspection

import (
	"context"
	"testing"

	api "github.com/containerd/containerd/api/services/introspection/v1"
)

// Mock implementation of Service for testing
type mockService struct {
	pluginsFunc    func(context.Context, ...string) (*api.PluginsResponse, error)
	serverFunc     func(context.Context) (*api.ServerResponse, error)
	pluginInfoFunc func(context.Context, string, string, any) (*api.PluginInfoResponse, error)
}

func (m *mockService) Plugins(ctx context.Context, filters ...string) (*api.PluginsResponse, error) {
	if m.pluginsFunc != nil {
		return m.pluginsFunc(ctx, filters...)
	}
	return &api.PluginsResponse{}, nil
}

func (m *mockService) Server(ctx context.Context) (*api.ServerResponse, error) {
	if m.serverFunc != nil {
		return m.serverFunc(ctx)
	}
	return &api.ServerResponse{}, nil
}

func (m *mockService) PluginInfo(ctx context.Context, pluginType, id string, options any) (*api.PluginInfoResponse, error) {
	if m.pluginInfoFunc != nil {
		return m.pluginInfoFunc(ctx, pluginType, id, options)
	}
	return &api.PluginInfoResponse{}, nil
}

func TestService_Interface(t *testing.T) {
	// Test that our mock properly implements the Service interface
	var service Service = &mockService{}

	if service == nil {
		t.Fatal("service should not be nil")
	}
}

func TestMockService_Plugins(t *testing.T) {
	expectedResponse := &api.PluginsResponse{
		Plugins: []*api.Plugin{
			{Type: "test", ID: "test-plugin"},
		},
	}

	service := &mockService{
		pluginsFunc: func(ctx context.Context, filters ...string) (*api.PluginsResponse, error) {
			if len(filters) != 1 {
				t.Errorf("expected 1 filter, got %d", len(filters))
			}
			if filters[0] != "type==test" {
				t.Errorf("expected filter 'type==test', got '%s'", filters[0])
			}
			return expectedResponse, nil
		},
	}

	resp, err := service.Plugins(context.Background(), "type==test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("expected %v, got %v", expectedResponse, resp)
	}
}

func TestMockService_Plugins_NoFilters(t *testing.T) {
	service := &mockService{
		pluginsFunc: func(ctx context.Context, filters ...string) (*api.PluginsResponse, error) {
			if len(filters) != 0 {
				t.Errorf("expected no filters, got %d", len(filters))
			}
			return &api.PluginsResponse{}, nil
		},
	}

	_, err := service.Plugins(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockService_Plugins_MultipleFilters(t *testing.T) {
	service := &mockService{
		pluginsFunc: func(ctx context.Context, filters ...string) (*api.PluginsResponse, error) {
			if len(filters) != 3 {
				t.Errorf("expected 3 filters, got %d", len(filters))
			}
			expectedFilters := []string{"type==snapshot", "id==native", "enabled==true"}
			for i, expected := range expectedFilters {
				if i < len(filters) && filters[i] != expected {
					t.Errorf("expected filter[%d] '%s', got '%s'", i, expected, filters[i])
				}
			}
			return &api.PluginsResponse{}, nil
		},
	}

	_, err := service.Plugins(context.Background(), "type==snapshot", "id==native", "enabled==true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMockService_Server(t *testing.T) {
	expectedResponse := &api.ServerResponse{
		UUID: "test-server-uuid",
		Pid:  1234,
	}

	service := &mockService{
		serverFunc: func(ctx context.Context) (*api.ServerResponse, error) {
			return expectedResponse, nil
		},
	}

	resp, err := service.Server(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("expected %v, got %v", expectedResponse, resp)
	}
}

func TestMockService_PluginInfo(t *testing.T) {
	expectedResponse := &api.PluginInfoResponse{
		Plugin: &api.Plugin{
			Type: "snapshot",
			ID:   "native",
		},
	}

	service := &mockService{
		pluginInfoFunc: func(ctx context.Context, pluginType, id string, options any) (*api.PluginInfoResponse, error) {
			if pluginType != "snapshot" {
				t.Errorf("expected pluginType 'snapshot', got '%s'", pluginType)
			}
			if id != "native" {
				t.Errorf("expected id 'native', got '%s'", id)
			}
			if options != nil {
				t.Errorf("expected nil options, got %v", options)
			}
			return expectedResponse, nil
		},
	}

	resp, err := service.PluginInfo(context.Background(), "snapshot", "native", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("expected %v, got %v", expectedResponse, resp)
	}
}

func TestMockService_PluginInfo_WithOptions(t *testing.T) {
	testOptions := struct {
		RootPath string `json:"root_path"`
		Debug    bool   `json:"debug"`
	}{
		RootPath: "/var/lib/containerd/snapshots",
		Debug:    true,
	}

	service := &mockService{
		pluginInfoFunc: func(ctx context.Context, pluginType, id string, options any) (*api.PluginInfoResponse, error) {
			if pluginType != "content" {
				t.Errorf("expected pluginType 'content', got '%s'", pluginType)
			}
			if id != "local" {
				t.Errorf("expected id 'local', got '%s'", id)
			}
			if options == nil {
				t.Error("expected non-nil options")
			}

			// Verify options structure if possible
			if opts, ok := options.(*struct {
				RootPath string `json:"root_path"`
				Debug    bool   `json:"debug"`
			}); ok {
				if opts.RootPath != "/var/lib/containerd/snapshots" {
					t.Errorf("expected root_path '/var/lib/containerd/snapshots', got %v", opts.RootPath)
				}
				if opts.Debug != true {
					t.Errorf("expected debug true, got %v", opts.Debug)
				}
			}

			return &api.PluginInfoResponse{}, nil
		},
	}

	_, err := service.PluginInfo(context.Background(), "content", "local", &testOptions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_MethodSignatures(t *testing.T) {
	// This test validates that the Service interface methods have the expected signatures
	// by attempting to assign function types that match the interface

	var service Service = &mockService{}

	// Test Plugins method signature
	pluginsFunc := func(ctx context.Context, filters ...string) (*api.PluginsResponse, error) {
		return service.Plugins(ctx, filters...)
	}

	if pluginsFunc == nil {
		t.Error("Plugins method signature mismatch")
	}

	// Test Server method signature
	serverFunc := func(ctx context.Context) (*api.ServerResponse, error) {
		return service.Server(ctx)
	}

	if serverFunc == nil {
		t.Error("Server method signature mismatch")
	}

	// Test PluginInfo method signature
	pluginInfoFunc := func(ctx context.Context, pluginType, id string, options any) (*api.PluginInfoResponse, error) {
		return service.PluginInfo(ctx, pluginType, id, options)
	}

	if pluginInfoFunc == nil {
		t.Error("PluginInfo method signature mismatch")
	}
}

func TestService_ContextHandling(t *testing.T) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	service := &mockService{
		pluginsFunc: func(receivedCtx context.Context, filters ...string) (*api.PluginsResponse, error) {
			if receivedCtx != ctxWithCancel {
				t.Error("context not properly passed to Plugins method")
			}
			return &api.PluginsResponse{}, nil
		},
		serverFunc: func(receivedCtx context.Context) (*api.ServerResponse, error) {
			if receivedCtx != ctxWithCancel {
				t.Error("context not properly passed to Server method")
			}
			return &api.ServerResponse{}, nil
		},
		pluginInfoFunc: func(receivedCtx context.Context, pluginType, id string, options any) (*api.PluginInfoResponse, error) {
			if receivedCtx != ctxWithCancel {
				t.Error("context not properly passed to PluginInfo method")
			}
			return &api.PluginInfoResponse{}, nil
		},
	}

	// Test context passing for all methods
	service.Plugins(ctxWithCancel, "test-filter")
	service.Server(ctxWithCancel)
	service.PluginInfo(ctxWithCancel, "test-type", "test-id", nil)
}
