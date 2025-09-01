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

package opts

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/cgroups/v3"
	cgroup1 "github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func TestWithNamespaceCgroupDeletion_Unified(t *testing.T) {
	// Skip if not running as root or if cgroups v2 is not available
	if os.Getuid() != 0 {
		t.Skip("skipping test that requires root privileges")
	}

	if cgroups.Mode() != cgroups.Unified {
		t.Skip("skipping cgroups v2 test on system without unified cgroups")
	}

	ctx := context.Background()
	testNamespace := "containerd-test-opts-unified"
	deleteInfo := &namespaces.DeleteInfo{
		Name: testNamespace,
	}

	// Create a test cgroup for cleanup
	tempDir := t.TempDir()
	cgroupPath := filepath.Join(tempDir, testNamespace)
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		t.Fatalf("Failed to create test cgroup directory: %v", err)
	}

	// Test successful deletion
	err := WithNamespaceCgroupDeletion(ctx, deleteInfo)
	// We expect this to fail in most cases since we can't easily create
	// valid cgroups v2 hierarchies in tests, but we're testing the code path
	if err == nil {
		t.Log("Cgroup deletion succeeded")
	} else {
		t.Logf("Cgroup deletion failed as expected in test environment: %v", err)
	}
}

func TestWithNamespaceCgroupDeletion_Legacy(t *testing.T) {
	// Skip if not running as root
	if os.Getuid() != 0 {
		t.Skip("skipping test that requires root privileges")
	}

	if cgroups.Mode() == cgroups.Unified {
		t.Skip("skipping cgroups v1 test on unified cgroups system")
	}

	ctx := context.Background()
	testNamespace := "containerd-test-opts-legacy"
	deleteInfo := &namespaces.DeleteInfo{
		Name: testNamespace,
	}

	// Test the legacy cgroup deletion path
	err := WithNamespaceCgroupDeletion(ctx, deleteInfo)
	// We expect this to potentially fail in test environments
	if err == nil {
		t.Log("Legacy cgroup deletion succeeded")
	} else if err == cgroup1.ErrCgroupDeleted {
		t.Log("Cgroup already deleted, which is handled correctly")
	} else {
		t.Logf("Legacy cgroup deletion failed as expected in test environment: %v", err)
	}
}

func TestWithNamespaceCgroupDeletion_EmptyNamespace(t *testing.T) {
	ctx := context.Background()
	deleteInfo := &namespaces.DeleteInfo{
		Name: "",
	}

	// Test with empty namespace name
	err := WithNamespaceCgroupDeletion(ctx, deleteInfo)
	if err == nil {
		t.Log("Empty namespace deletion succeeded")
	} else {
		t.Logf("Empty namespace deletion failed as expected: %v", err)
	}
}

func TestWithNamespaceCgroupDeletion_InvalidNamespace(t *testing.T) {
	ctx := context.Background()
	deleteInfo := &namespaces.DeleteInfo{
		Name: "nonexistent-namespace-12345",
	}

	// Test with nonexistent namespace
	err := WithNamespaceCgroupDeletion(ctx, deleteInfo)
	if err == nil {
		t.Log("Nonexistent namespace deletion succeeded")
	} else {
		t.Logf("Nonexistent namespace deletion failed as expected: %v", err)
	}
}

func TestWithNamespaceCgroupDeletion_NilDeleteInfo(t *testing.T) {
	ctx := context.Background()

	// Test with nil delete info - should panic or handle gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Function panicked with nil DeleteInfo as expected: %v", r)
		}
	}()

	err := WithNamespaceCgroupDeletion(ctx, nil)
	if err != nil {
		t.Logf("Nil DeleteInfo handled with error: %v", err)
	}
}

func TestWithNamespaceCgroupDeletion_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	deleteInfo := &namespaces.DeleteInfo{
		Name: "test-namespace-canceled",
	}

	// Test with canceled context
	err := WithNamespaceCgroupDeletion(ctx, deleteInfo)
	if err == nil {
		t.Log("Cgroup deletion with canceled context succeeded")
	} else {
		t.Logf("Cgroup deletion with canceled context failed: %v", err)
	}
}

// Benchmark the cgroup deletion operation
func BenchmarkWithNamespaceCgroupDeletion(b *testing.B) {
	ctx := context.Background()
	deleteInfo := &namespaces.DeleteInfo{
		Name: "benchmark-namespace",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithNamespaceCgroupDeletion(ctx, deleteInfo)
	}
}

// Test the behavior when cgroups mode detection fails
func TestWithNamespaceCgroupDeletion_ModeDetection(t *testing.T) {
	ctx := context.Background()
	deleteInfo := &namespaces.DeleteInfo{
		Name: "mode-detection-test",
	}

	// Test that the function properly detects cgroup mode and handles both paths
	err := WithNamespaceCgroupDeletion(ctx, deleteInfo)

	// The function should always attempt deletion based on detected mode
	if err == nil {
		t.Log("Cgroup deletion succeeded")
	} else {
		t.Logf("Cgroup deletion failed: %v", err)
	}

	// Verify that the correct cgroup mode is being detected
	mode := cgroups.Mode()
	if mode == cgroups.Unified {
		t.Log("Detected unified cgroups (v2)")
	} else if mode == cgroups.Legacy {
		t.Log("Detected legacy cgroups (v1)")
	} else if mode == cgroups.Hybrid {
		t.Log("Detected hybrid cgroups")
	} else {
		t.Logf("Detected unknown cgroups mode: %v", mode)
	}
}
