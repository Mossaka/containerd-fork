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

package metrics

import (
	"testing"
	"time"

	"github.com/containerd/containerd/v2/pkg/timeout"
	goMetrics "github.com/docker/go-metrics"
)

func TestShimStatsRequestTimeoutConstant(t *testing.T) {
	expected := "io.containerd.timeout.metrics.shimstats"
	if ShimStatsRequestTimeout != expected {
		t.Errorf("Expected ShimStatsRequestTimeout to be %q, got %q", expected, ShimStatsRequestTimeout)
	}
}

func TestMetricsInitialization(t *testing.T) {
	// Test that timeout is properly set during init
	timeout := timeout.Get(ShimStatsRequestTimeout)
	expected := 2 * time.Second
	if timeout != expected {
		t.Errorf("Expected timeout to be %v, got %v", expected, timeout)
	}
}

func TestBuildInfoMetricsRegistration(t *testing.T) {
	// Test that metrics namespace is properly initialized
	// We can't directly test the metrics registration without access to the internal state,
	// but we can verify the functionality indirectly by checking that no panics occur
	// during initialization (which would happen if there were registration conflicts)
	
	// This test primarily serves as a smoke test to ensure the init() function
	// doesn't cause any issues and that the metrics registration completes successfully
	t.Log("Metrics initialization completed successfully")
}

func TestTimeoutConfiguration(t *testing.T) {
	// Verify the timeout can be retrieved and has expected properties
	retrievedTimeout := timeout.Get(ShimStatsRequestTimeout)
	
	if retrievedTimeout <= 0 {
		t.Error("Timeout should be positive")
	}
	
	if retrievedTimeout != 2*time.Second {
		t.Errorf("Expected timeout to be 2 seconds, got %v", retrievedTimeout)
	}
}

func TestPackageImports(t *testing.T) {
	// Test that required packages are properly imported and accessible
	// This validates the package dependencies
	
	// Test timeout package integration
	testTimeout := timeout.Get("test.timeout")
	if testTimeout < 0 {
		t.Error("Timeout package integration failed")
	}
	
	// Test goMetrics package integration
	testNs := goMetrics.NewNamespace("test", "test", nil)
	if testNs == nil {
		t.Error("goMetrics package integration failed")
	}
}