//go:build !linux

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
	"sync"
	"testing"
)

func TestIsEnabled_NonLinux(t *testing.T) {
	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to always return false on non-Linux platforms, got true")
	}
}

func TestIsEnabled_Concurrency_NonLinux(t *testing.T) {
	const numGoroutines = 10
	results := make([]bool, numGoroutines)
	var wg sync.WaitGroup

	// Test concurrent access to IsEnabled()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = IsEnabled()
		}(i)
	}

	wg.Wait()

	// All results should be false
	for i, result := range results {
		if result != false {
			t.Errorf("Concurrent call %d returned %v, expected false", i, result)
		}
	}
}

func TestSetConfig_NonLinux(t *testing.T) {
	testCases := []string{
		"",
		"/path/to/config",
		"invalid-path",
		"/tmp/test-config.conf",
	}

	for _, configPath := range testCases {
		err := SetConfig(configPath)
		if err != nil {
			t.Errorf("SetConfig(%q) returned error on non-Linux platform: %v", configPath, err)
		}

		// IsEnabled should still return false
		if IsEnabled() != false {
			t.Errorf("Expected IsEnabled() to remain false after SetConfig(%q) on non-Linux", configPath)
		}
	}
}

func TestSetConfig_Concurrency_NonLinux(t *testing.T) {
	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	// Test concurrent calls to SetConfig
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = SetConfig("/test/path")
		}(i)
	}

	wg.Wait()

	// All calls should succeed (no-op on non-Linux)
	for i, err := range errors {
		if err != nil {
			t.Errorf("Concurrent SetConfig call %d returned error on non-Linux: %v", i, err)
		}
	}

	// IsEnabled should still be false
	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to remain false after concurrent SetConfig calls on non-Linux")
	}
}

func TestClassNameToLinuxOCI_NonLinux(t *testing.T) {
	testCases := []string{
		"",
		"test-class",
		"best-effort",
		"guaranteed",
		"burstable",
	}

	for _, className := range testCases {
		result, err := ClassNameToLinuxOCI(className)
		if err != nil {
			t.Errorf("ClassNameToLinuxOCI(%q) returned error on non-Linux platform: %v", className, err)
		}

		if result != nil {
			t.Errorf("ClassNameToLinuxOCI(%q) should return nil on non-Linux platform, got: %v", className, result)
		}
	}
}

func TestContainerClassFromAnnotations_NonLinux(t *testing.T) {
	testCases := []struct {
		name                 string
		containerName        string
		containerAnnotations map[string]string
		podAnnotations       map[string]string
	}{
		{
			name:                 "empty inputs",
			containerName:        "",
			containerAnnotations: nil,
			podAnnotations:       nil,
		},
		{
			name:                 "with container name",
			containerName:        "test-container",
			containerAnnotations: nil,
			podAnnotations:       nil,
		},
		{
			name:          "with annotations",
			containerName: "test-container",
			containerAnnotations: map[string]string{
				"io.kubernetes.cri.container-name": "test",
				"blockio.class":                    "best-effort",
			},
			podAnnotations: map[string]string{
				"pod.blockio.class": "guaranteed",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ContainerClassFromAnnotations(tc.containerName, tc.containerAnnotations, tc.podAnnotations)
			if err != nil {
				t.Errorf("ContainerClassFromAnnotations returned error on non-Linux platform: %v", err)
			}

			if result != "" {
				t.Errorf("ContainerClassFromAnnotations should return empty string on non-Linux platform, got: %q", result)
			}
		})
	}
}

func TestConcurrentOperations_NonLinux(t *testing.T) {
	const numGoroutines = 10
	var wg sync.WaitGroup

	// Test concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Mix of operations
			switch index % 4 {
			case 0:
				IsEnabled()
			case 1:
				SetConfig("/test/path")
			case 2:
				ClassNameToLinuxOCI("test-class")
			case 3:
				ContainerClassFromAnnotations("container", nil, nil)
			}
		}(i)
	}

	wg.Wait()

	// Final state checks
	if IsEnabled() != false {
		t.Error("Expected IsEnabled() to be false after concurrent operations on non-Linux")
	}

	// Additional operations should still work
	err := SetConfig("/another/path")
	if err != nil {
		t.Errorf("SetConfig failed after concurrent operations: %v", err)
	}

	result, err := ClassNameToLinuxOCI("final-test")
	if err != nil {
		t.Errorf("ClassNameToLinuxOCI failed after concurrent operations: %v", err)
	}
	if result != nil {
		t.Errorf("ClassNameToLinuxOCI should return nil on non-Linux, got: %v", result)
	}
}
