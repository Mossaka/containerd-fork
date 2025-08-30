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

package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/containerd/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		address string
		opts    []Opt
		wantErr bool
	}{
		{
			name:    "empty address without services",
			address: "",
			opts:    nil,
			wantErr: true,
		},
		{
			name:    "with default namespace",
			address: "",
			opts:    []Opt{WithDefaultNamespace("test-ns")},
			wantErr: true, // still no connection
		},
		{
			name:    "with timeout",
			address: "",
			opts:    []Opt{WithTimeout(5 * time.Second)},
			wantErr: true, // still no connection
		},
		{
			name:    "with platform",
			address: "",
			opts:    []Opt{WithDefaultPlatform(platforms.Default())},
			wantErr: true, // still no connection
		},
		{
			name:    "with default runtime",
			address: "",
			opts:    []Opt{WithDefaultRuntime("runc")},
			wantErr: true, // still no connection
		},
		{
			name:    "with services",
			address: "",
			opts:    []Opt{WithServices()},
			wantErr: false, // services provided
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.address, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					client.Close()
				}
			}
		})
	}
}

func TestNewWithValidOptions(t *testing.T) {
	// Test with services provided (no connection needed)
	client, err := New("", WithServices())
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// Verify default values are set
	assert.Equal(t, platforms.Default(), client.platform)
}

func TestNewWithConn(t *testing.T) {
	// Create a mock grpc connection
	conn := &grpc.ClientConn{}
	
	client, err := NewWithConn(conn)
	require.NoError(t, err)
	require.NotNil(t, client)
	
	// Verify client configuration
	assert.Equal(t, platforms.Default(), client.platform)
	assert.Equal(t, conn, client.conn)
}

func TestNewWithConnAndOptions(t *testing.T) {
	conn := &grpc.ClientConn{}
	
	client, err := NewWithConn(conn, 
		WithDefaultNamespace("test-ns"),
		WithDefaultRuntime("runc"),
		WithDefaultPlatform(platforms.Only(platforms.DefaultSpec())),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
	
	assert.Equal(t, "test-ns", client.defaultns)
	assert.NotNil(t, client.platform)
}

func TestClientClose(t *testing.T) {
	// Test closing client with services
	client, err := New("", WithServices())
	require.NoError(t, err)
	
	err = client.Close()
	assert.NoError(t, err)
}

func TestClientServiceGetters(t *testing.T) {
	client, err := New("", WithServices())
	require.NoError(t, err)
	defer client.Close()

	// Test that all service getter methods return non-nil services
	assert.NotNil(t, client.NamespaceService())
	assert.NotNil(t, client.ContentStore())
	assert.NotNil(t, client.ImageService())
	assert.NotNil(t, client.ContainerService())
	assert.NotNil(t, client.SnapshotService("overlay"))
	assert.NotNil(t, client.TaskService())
	assert.NotNil(t, client.EventService())
	assert.NotNil(t, client.LeasesService())
	assert.NotNil(t, client.IntrospectionService())
	assert.NotNil(t, client.DiffService())
	assert.NotNil(t, client.HealthService())
	assert.NotNil(t, client.TransferService())
	assert.NotNil(t, client.VersionService())
	
	// Test platform method
	assert.Equal(t, platforms.Default(), client.platform)
}

func TestClientRuntime(t *testing.T) {
	// Test with default runtime set
	client, err := New("", WithServices(), WithDefaultRuntime("runc"))
	require.NoError(t, err)
	defer client.Close()
	
	runtime := client.Runtime()
	assert.Equal(t, "runc", runtime)
}

func TestClientRuntimeEmpty(t *testing.T) {
	client, err := New("", WithServices())
	require.NoError(t, err)
	defer client.Close()
	
	// Test getting runtime when none set - should not panic
	runtime := client.Runtime()
	// Should return some value (empty or default)
	_ = runtime // Just make sure it doesn't panic
}

func TestClientReconnect(t *testing.T) {
	client, err := New("", WithServices())
	require.NoError(t, err)
	defer client.Close()
	
	// Test reconnect without connector
	err = client.Reconnect()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no connector available")
}

func TestWithInvalidOptions(t *testing.T) {
	// Test with invalid option that returns error
	invalidOpt := func(*clientOpts) error {
		return fmt.Errorf("invalid option")
	}
	
	client, err := New("", invalidOpt)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "invalid option")
}

func TestNamespaceContext(t *testing.T) {
	ctx := context.Background()
	
	// Test with namespace
	ctxWithNS := namespaces.WithNamespace(ctx, "test-namespace")
	ns, ok := namespaces.Namespace(ctxWithNS)
	assert.True(t, ok)
	assert.Equal(t, "test-namespace", ns)
}

// Integration-style tests that don't require actual grpc connections
func TestClientCreationFlow(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Client, error)
		teardown func(*Client)
		verify   func(t *testing.T, client *Client)
	}{
		{
			name: "client with services",
			setup: func() (*Client, error) {
				return New("", WithServices())
			},
			teardown: func(c *Client) { c.Close() },
			verify: func(t *testing.T, client *Client) {
				assert.NotNil(t, client)
				assert.NotNil(t, client.ContentStore())
				assert.NotNil(t, client.ImageService())
			},
		},
		{
			name: "client with custom options",
			setup: func() (*Client, error) {
				return New("", 
					WithServices(),
					WithDefaultNamespace("custom-ns"),
					WithDefaultRuntime("custom-runtime"),
					WithTimeout(30*time.Second),
				)
			},
			teardown: func(c *Client) { c.Close() },
			verify: func(t *testing.T, client *Client) {
				assert.NotNil(t, client)
				assert.Equal(t, "custom-ns", client.defaultns)
				assert.Equal(t, "custom-runtime", client.Runtime())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setup()
			require.NoError(t, err)
			require.NotNil(t, client)
			
			if tt.teardown != nil {
				defer tt.teardown(client)
			}
			
			if tt.verify != nil {
				tt.verify(t, client)
			}
		})
	}
}

// Test client options functionality
func TestClientOptions(t *testing.T) {
	t.Run("WithDefaultNamespace", func(t *testing.T) {
		client, err := New("", WithServices(), WithDefaultNamespace("test-ns"))
		require.NoError(t, err)
		defer client.Close()
		assert.Equal(t, "test-ns", client.defaultns)
	})
	
	t.Run("WithDefaultRuntime", func(t *testing.T) {
		client, err := New("", WithServices(), WithDefaultRuntime("test-runtime"))
		require.NoError(t, err)
		defer client.Close()
		assert.Equal(t, "test-runtime", client.Runtime())
	})
	
	t.Run("WithDefaultPlatform", func(t *testing.T) {
		testPlatform := platforms.Only(platforms.DefaultSpec())
		client, err := New("", WithServices(), WithDefaultPlatform(testPlatform))
		require.NoError(t, err)
		defer client.Close()
		assert.Equal(t, testPlatform, client.platform)
	})
}

// Test multiple client creation and cleanup
func TestMultipleClientCreation(t *testing.T) {
	for i := 0; i < 5; i++ {
		client, err := New("", WithServices(), WithDefaultNamespace(fmt.Sprintf("ns-%d", i)))
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("ns-%d", i), client.defaultns)
		client.Close()
	}
}

// Test client field access
func TestClientFields(t *testing.T) {
	client, err := New("", WithServices(), 
		WithDefaultNamespace("field-test"),
		WithDefaultRuntime("test-runtime"),
	)
	require.NoError(t, err)
	defer client.Close()

	// Test accessible fields
	assert.Equal(t, "field-test", client.defaultns)
	assert.Equal(t, platforms.Default(), client.platform)
	
	// Test runtime access
	assert.Equal(t, "test-runtime", client.Runtime())
}

// Test client with timeout
func TestClientTimeout(t *testing.T) {
	client, err := New("", WithServices(), WithTimeout(5*time.Second))
	require.NoError(t, err)
	defer client.Close()
	
	// Just verify client was created successfully with timeout option
	assert.NotNil(t, client)
}