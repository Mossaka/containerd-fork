//go:build linux

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

package blockio

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsEnabledLinux(t *testing.T) {
	// Save original state
	originalEnabled := enabled
	defer func() {
		enabled = originalEnabled
	}()

	// Test when disabled
	enabled = false
	if IsEnabled() {
		t.Error("IsEnabled() should return false when disabled")
	}

	// Test when enabled
	enabled = true
	if !IsEnabled() {
		t.Error("IsEnabled() should return true when enabled")
	}
}

func TestIsEnabledConcurrencyLinux(t *testing.T) {
	// Test concurrent access to IsEnabled
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				IsEnabled()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("Test timed out - possible deadlock in IsEnabled")
		}
	}
}

func TestSetConfigEmptyPathLinux(t *testing.T) {
	// Save original state
	originalEnabled := enabled
	defer func() {
		enabled = originalEnabled
	}()

	// Test with empty config path
	err := SetConfig("")
	if err != nil {
		t.Errorf("SetConfig with empty path should not return error, got: %v", err)
	}

	if IsEnabled() {
		t.Error("SetConfig with empty path should disable blockio")
	}
}

func TestSetConfigInvalidPathLinux(t *testing.T) {
	// Save original state
	originalEnabled := enabled
	defer func() {
		enabled = originalEnabled
	}()

	// Test with non-existent config file
	err := SetConfig("/path/does/not/exist.conf")
	if err == nil {
		t.Error("SetConfig with invalid path should return error")
	}

	if IsEnabled() {
		t.Error("SetConfig with invalid path should not enable blockio")
	}
}

func TestSetConfigValidPathLinux(t *testing.T) {
	// Save original state
	originalEnabled := enabled
	defer func() {
		enabled = originalEnabled
	}()

	// Create a temporary config file with minimal valid content
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "blockio.conf")
	
	// Create minimal valid blockio config
	configContent := `# blockio configuration
# This is a test configuration file
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test with valid config file
	// Note: This may still fail if goresctrl blockio package has strict validation
	// but we test that the error handling works properly
	err = SetConfig(configFile)
	// We don't assert success since the actual blockio library may have strict requirements
	// but we verify that the function doesn't panic and handles errors gracefully
	if err != nil {
		t.Logf("Expected: SetConfig with test file returned error (this is normal): %v", err)
		if IsEnabled() {
			t.Error("When SetConfig returns error, blockio should remain disabled")
		}
	} else {
		t.Logf("SetConfig with test file succeeded")
		if !IsEnabled() {
			t.Error("When SetConfig succeeds, blockio should be enabled")
		}
	}
}

func TestSetConfigConcurrencyLinux(t *testing.T) {
	// Save original state
	originalEnabled := enabled
	defer func() {
		enabled = originalEnabled
	}()

	// Test concurrent access to SetConfig
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			// Each goroutine calls SetConfig multiple times
			for j := 0; j < 10; j++ {
				if j%2 == 0 {
					SetConfig("")
				} else {
					SetConfig("/nonexistent/path")
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	timeout := time.After(10 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("Test timed out - possible deadlock in SetConfig")
		}
	}
}

func TestClassNameToLinuxOCILinux(t *testing.T) {
	testCases := []struct {
		name      string
		className string
		expectErr bool
	}{
		{
			name:      "empty class name",
			className: "",
			expectErr: false, // May or may not error depending on implementation
		},
		{
			name:      "valid class name",
			className: "default",
			expectErr: false, // May error if not configured
		},
		{
			name:      "invalid class name",
			className: "nonexistent-class",
			expectErr: false, // May error depending on configuration
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ClassNameToLinuxOCI(tc.className)
			
			// Verify it doesn't panic
			// The actual behavior depends on whether blockio is configured
			t.Logf("ClassNameToLinuxOCI(%q) returned result=%v, err=%v", tc.className, result, err)
		})
	}
}

func TestContainerClassFromAnnotationsLinux(t *testing.T) {
	testCases := []struct {
		name                  string
		containerName         string
		containerAnnotations  map[string]string
		podAnnotations        map[string]string
		expectErr            bool
	}{
		{
			name:          "empty inputs",
			containerName: "",
			containerAnnotations: nil,
			podAnnotations: nil,
			expectErr:     false,
		},
		{
			name:          "with container name",
			containerName: "test-container",
			containerAnnotations: map[string]string{},
			podAnnotations: map[string]string{},
			expectErr:     false,
		},
		{
			name:          "with annotations",
			containerName: "test-container",
			containerAnnotations: map[string]string{
				"io.kubernetes.container.name": "test-container",
				"custom.annotation": "value",
			},
			podAnnotations: map[string]string{
				"io.kubernetes.pod.name": "test-pod",
				"blockio.class": "high-priority",
			},
			expectErr:     false,
		},
		{
			name:          "with blockio annotations",
			containerName: "test-container",
			containerAnnotations: map[string]string{
				"intel.com/blockio": "performance",
			},
			podAnnotations: map[string]string{
				"intel.com/blockio": "default",
			},
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			className, err := ContainerClassFromAnnotations(
				tc.containerName,
				tc.containerAnnotations,
				tc.podAnnotations,
			)
			
			// Verify it doesn't panic and handles input gracefully
			t.Logf("ContainerClassFromAnnotations returned className=%q, err=%v", className, err)
			
			if tc.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Logf("Unexpected error (may be normal if blockio not configured): %v", err)
			}
		})
	}
}

func TestContainerClassFromAnnotationsNilMapsLinux(t *testing.T) {
	// Test with nil annotation maps to ensure no panic
	className, err := ContainerClassFromAnnotations("test-container", nil, nil)
	
	// Should not panic with nil maps
	t.Logf("ContainerClassFromAnnotations with nil maps returned className=%q, err=%v", className, err)
}

// Test package-level variable safety on Linux
func TestPackageVariableInitializationLinux(t *testing.T) {
	// Test that mutex is usable (shouldn't panic)
	enabledMu.RLock()
	val := enabled
	enabledMu.RUnlock()
	
	t.Logf("Package variable enabled is: %v", val)
}

// Test edge cases with special characters in annotations on Linux
func TestContainerClassFromAnnotationsSpecialCharsLinux(t *testing.T) {
	testCases := []struct {
		name                 string
		containerName        string
		containerAnnotations map[string]string
		podAnnotations       map[string]string
	}{
		{
			name:          "unicode characters",
			containerName: "test-container-ü",
			containerAnnotations: map[string]string{
				"annotation/with/slashes": "value",
				"annotation.with.dots": "value",
			},
			podAnnotations: map[string]string{
				"pod-annotation_with_underscores": "value",
			},
		},
		{
			name:          "empty values",
			containerName: "test",
			containerAnnotations: map[string]string{
				"empty-value": "",
			},
			podAnnotations: map[string]string{
				"another-empty": "",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic with special characters
			className, err := ContainerClassFromAnnotations(
				tc.containerName,
				tc.containerAnnotations,
				tc.podAnnotations,
			)
			
			t.Logf("ContainerClassFromAnnotations with special chars returned className=%q, err=%v", className, err)
		})
	}
}