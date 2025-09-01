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
	"fmt"
	"testing"
	"time"

	"github.com/containerd/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestWithDefaultNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{"empty namespace", ""},
		{"default namespace", "default"},
		{"custom namespace", "my-namespace"},
		{"k8s namespace", "kube-system"},
		{"special chars", "test-namespace_123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts clientOpts
			opt := WithDefaultNamespace(tt.namespace)
			
			err := opt(&opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.namespace, opts.defaultns)
		})
	}
}

func TestWithDefaultRuntime(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
	}{
		{"empty runtime", ""},
		{"runc", "runc"},
		{"kata", "kata"},
		{"gvisor", "gvisor"},
		{"custom runtime", "my-custom-runtime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts clientOpts
			opt := WithDefaultRuntime(tt.runtime)
			
			err := opt(&opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.runtime, opts.defaultRuntime)
		})
	}
}

func TestWithDefaultPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform platforms.MatchComparer
	}{
		{"default platform", platforms.Default()},
		{"linux/amd64", platforms.Only(platforms.DefaultSpec())},
		{"custom platform", platforms.OnlyStrict(platforms.DefaultSpec())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts clientOpts
			opt := WithDefaultPlatform(tt.platform)
			
			err := opt(&opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.platform, opts.defaultPlatform)
		})
	}
}

func TestWithServices(t *testing.T) {
	var opts clientOpts
	opt := WithServices()
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.NotNil(t, opts.services)
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero timeout", 0},
		{"1 second", 1 * time.Second},
		{"default timeout", 10 * time.Second},
		{"long timeout", 5 * time.Minute},
		{"very long timeout", 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts clientOpts
			opt := WithTimeout(tt.timeout)
			
			err := opt(&opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.timeout, opts.timeout)
		})
	}
}

func TestWithDialOpts(t *testing.T) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}
	
	var opts clientOpts
	opt := WithDialOpts(dialOpts)
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.Equal(t, dialOpts, opts.dialOptions)
}

func TestWithExtraDialOpts(t *testing.T) {
	extraDialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	
	var opts clientOpts
	opt := WithExtraDialOpts(extraDialOpts)
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.Equal(t, extraDialOpts, opts.extraDialOpts)
}

func TestWithCallOpts(t *testing.T) {
	callOpts := []grpc.CallOption{
		grpc.MaxCallRecvMsgSize(1024),
		grpc.MaxCallSendMsgSize(1024),
	}
	
	var opts clientOpts
	opt := WithCallOpts(callOpts)
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.Equal(t, callOpts, opts.callOptions)
}

func TestMultipleOptions(t *testing.T) {
	var opts clientOpts
	
	// Apply multiple options
	options := []Opt{
		WithDefaultNamespace("test-ns"),
		WithDefaultRuntime("runc"),
		WithTimeout(30 * time.Second),
		WithDefaultPlatform(platforms.Default()),
	}
	
	for _, opt := range options {
		err := opt(&opts)
		require.NoError(t, err)
	}
	
	// Verify all options were applied
	assert.Equal(t, "test-ns", opts.defaultns)
	assert.Equal(t, "runc", opts.defaultRuntime)
	assert.Equal(t, 30*time.Second, opts.timeout)
	assert.Equal(t, platforms.Default(), opts.defaultPlatform)
}

func TestOptionsComposition(t *testing.T) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	callOpts := []grpc.CallOption{
		grpc.MaxCallRecvMsgSize(2048),
	}
	
	var opts clientOpts
	
	// Apply complex combination of options
	options := []Opt{
		WithDefaultNamespace("production"),
		WithDefaultRuntime("kata"),
		WithTimeout(45 * time.Second),
		WithDefaultPlatform(platforms.OnlyStrict(platforms.DefaultSpec())),
		WithServices(),
		WithDialOpts(dialOpts),
		WithCallOpts(callOpts),
	}
	
	for _, opt := range options {
		err := opt(&opts)
		require.NoError(t, err)
	}
	
	// Verify comprehensive configuration
	assert.Equal(t, "production", opts.defaultns)
	assert.Equal(t, "kata", opts.defaultRuntime)
	assert.Equal(t, 45*time.Second, opts.timeout)
	assert.NotNil(t, opts.defaultPlatform)
	assert.NotNil(t, opts.services)
	assert.Equal(t, dialOpts, opts.dialOptions)
	assert.Equal(t, callOpts, opts.callOptions)
}

// Test edge cases and error handling
func TestOptionsDefaults(t *testing.T) {
	var opts clientOpts
	
	// Verify default zero values
	assert.Equal(t, "", opts.defaultns)
	assert.Equal(t, "", opts.defaultRuntime)
	assert.Equal(t, time.Duration(0), opts.timeout)
	assert.Nil(t, opts.defaultPlatform)
	assert.Nil(t, opts.services)
	assert.Nil(t, opts.dialOptions)
	assert.Nil(t, opts.callOptions)
}

func TestOptionsValidation(t *testing.T) {
	// All current options should not return errors
	// This test ensures that option functions don't have hidden validation
	
	options := []Opt{
		WithDefaultNamespace(""),
		WithDefaultNamespace("valid-namespace"),
		WithDefaultRuntime(""),
		WithDefaultRuntime("valid-runtime"),
		WithTimeout(0),
		WithTimeout(time.Second),
		WithDefaultPlatform(nil),
		WithDefaultPlatform(platforms.Default()),
		WithServices(),
		WithDialOpts(nil),
		WithDialOpts([]grpc.DialOption{}),
		WithCallOpts(nil),
		WithCallOpts([]grpc.CallOption{}),
	}
	
	for i, opt := range options {
		t.Run(fmt.Sprintf("option_%d", i), func(t *testing.T) {
			var opts clientOpts
			err := opt(&opts)
			assert.NoError(t, err, "Option should not return error")
		})
	}
}

// Test that options can be applied in any order
func TestOptionsOrdering(t *testing.T) {
	tests := []struct {
		name string
		opts []Opt
	}{
		{
			name: "namespace first",
			opts: []Opt{
				WithDefaultNamespace("test"),
				WithDefaultRuntime("runc"),
				WithTimeout(10 * time.Second),
			},
		},
		{
			name: "timeout first",
			opts: []Opt{
				WithTimeout(10 * time.Second),
				WithDefaultNamespace("test"),
				WithDefaultRuntime("runc"),
			},
		},
		{
			name: "runtime first",
			opts: []Opt{
				WithDefaultRuntime("runc"),
				WithTimeout(10 * time.Second),
				WithDefaultNamespace("test"),
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts clientOpts
			
			for _, opt := range tt.opts {
				err := opt(&opts)
				require.NoError(t, err)
			}
			
			// All should result in the same final state
			assert.Equal(t, "test", opts.defaultns)
			assert.Equal(t, "runc", opts.defaultRuntime)
			assert.Equal(t, 10*time.Second, opts.timeout)
		})
	}
}

func TestMultipleExtraDialOpts(t *testing.T) {
	var opts clientOpts
	
	// Apply multiple extra dial options
	opt1 := WithExtraDialOpts([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})
	opt2 := WithExtraDialOpts([]grpc.DialOption{
		grpc.WithBlock(),
	})
	
	err := opt1(&opts)
	require.NoError(t, err)
	err = opt2(&opts)
	require.NoError(t, err)
	
	// Should have both options appended
	assert.Len(t, opts.extraDialOpts, 2)
}

// Test specific option behaviors
func TestWithDefaultPlatformNil(t *testing.T) {
	var opts clientOpts
	opt := WithDefaultPlatform(nil)
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.Nil(t, opts.defaultPlatform)
}

func TestWithEmptyDialOpts(t *testing.T) {
	var opts clientOpts
	opt := WithDialOpts([]grpc.DialOption{})
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.Empty(t, opts.dialOptions)
}

func TestWithEmptyCallOpts(t *testing.T) {
	var opts clientOpts
	opt := WithCallOpts([]grpc.CallOption{})
	
	err := opt(&opts)
	assert.NoError(t, err)
	assert.Empty(t, opts.callOptions)
}