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
	"sync"
	"testing"
	"time"
)

func TestIsEnabled_Initial(t *testing.T) {
	// Reset the enabled state before test
	enabledMu.Lock()
	enabled = false
	enabledMu.Unlock()

	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to return false initially, got true")
	}
}

func TestIsEnabled_Concurrency(t *testing.T) {
	// Reset state
	enabledMu.Lock()
	enabled = false
	enabledMu.Unlock()

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

	// All results should be consistent
	expected := results[0]
	for i, result := range results {
		if result != expected {
			t.Errorf("Concurrent call %d returned %v, expected %v", i, result, expected)
		}
	}
}

func TestSetConfig_EmptyPath(t *testing.T) {
	// Reset state
	enabledMu.Lock()
	enabled = true // Start with true to verify it gets set to false
	enabledMu.Unlock()

	err := SetConfig("")
	if err != nil {
		t.Errorf("SetConfig with empty path should not return error, got: %v", err)
	}

	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to return false after SetConfig with empty path, got true")
	}
}

func TestSetConfig_NonExistentFile(t *testing.T) {
	// Reset state
	enabledMu.Lock()
	enabled = false
	enabledMu.Unlock()

	nonExistentPath := "/tmp/non-existent-blockio-config-file-12345.conf"
	err := SetConfig(nonExistentPath)

	// Should return an error for non-existent file
	if err == nil {
		t.Errorf("Expected SetConfig to return error for non-existent file, got nil")
	}

	// enabled should remain false after error
	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to return false after SetConfig error, got true")
	}
}

func TestSetConfig_InvalidConfigFile(t *testing.T) {
	// Create a temporary invalid config file
	tmpDir, err := os.MkdirTemp("", "blockio-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "invalid.conf")
	err = os.WriteFile(configFile, []byte("invalid config content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Reset state
	enabledMu.Lock()
	enabled = false
	enabledMu.Unlock()

	err = SetConfig(configFile)

	// Should return an error for invalid config
	if err == nil {
		t.Errorf("Expected SetConfig to return error for invalid config file, got nil")
	}

	// enabled should remain false after error
	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to return false after SetConfig with invalid config, got true")
	}
}

func TestSetConfig_Concurrency(t *testing.T) {
	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	// Test concurrent calls to SetConfig
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = SetConfig("")
		}(i)
	}

	wg.Wait()

	// All calls should succeed (empty path is valid)
	for i, err := range errors {
		if err != nil {
			t.Errorf("Concurrent SetConfig call %d returned error: %v", i, err)
		}
	}

	// Final state should be consistent
	if IsEnabled() != false {
		t.Errorf("Expected IsEnabled() to return false after concurrent SetConfig calls, got true")
	}
}

func TestClassNameToLinuxOCI_EmptyClassName(t *testing.T) {
	result, err := ClassNameToLinuxOCI("")

	// This delegates to the goresctrl library, so we test that it doesn't panic
	// and that the result is consistent with the library behavior
	if err != nil && result != nil {
		t.Errorf("Expected either error or nil result, got both: result=%v, err=%v", result, err)
	}
}

func TestClassNameToLinuxOCI_InvalidClassName(t *testing.T) {
	invalidClassName := "invalid-class-name-12345"
	result, err := ClassNameToLinuxOCI(invalidClassName)

	// This delegates to the goresctrl library, so we test that it doesn't panic
	// For invalid class names, we expect either an error or nil result
	if err != nil && result != nil {
		t.Errorf("Expected either error or nil result for invalid class name, got both: result=%v, err=%v", result, err)
	}
}

func TestContainerClassFromAnnotations_EmptyInputs(t *testing.T) {
	result, err := ContainerClassFromAnnotations("", nil, nil)

	// This delegates to the goresctrl library, test that it doesn't panic
	if err != nil && result != "" {
		t.Errorf("Expected either error or empty result, got both: result=%s, err=%v", result, err)
	}
}

func TestContainerClassFromAnnotations_WithAnnotations(t *testing.T) {
	containerName := "test-container"
	containerAnnotations := map[string]string{
		"io.kubernetes.cri.container-name": "test",
		"test-annotation":                  "test-value",
	}
	podAnnotations := map[string]string{
		"pod-annotation": "pod-value",
	}

	result, err := ContainerClassFromAnnotations(containerName, containerAnnotations, podAnnotations)

	// This delegates to the goresctrl library, test that it doesn't panic
	// and handles the inputs properly
	if err != nil && result != "" {
		t.Errorf("Expected either error or empty result, got both: result=%s, err=%v", result, err)
	}
}

func TestStateConsistency(t *testing.T) {
	// Test that the enabled state remains consistent across operations

	// Start with disabled state
	err := SetConfig("")
	if err != nil {
		t.Fatalf("SetConfig with empty path failed: %v", err)
	}

	if IsEnabled() {
		t.Error("Expected IsEnabled() to be false after SetConfig with empty path")
	}

	// Try multiple reads
	for i := 0; i < 5; i++ {
		if IsEnabled() {
			t.Errorf("IsEnabled() inconsistent on call %d", i)
		}
	}
}

func TestConcurrentSetConfigAndIsEnabled(t *testing.T) {
	const duration = 100 * time.Millisecond
	const numReaders = 5
	const numWriters = 2

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Start reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					IsEnabled() // Just call it, don't check result due to concurrent writers
				}
			}
		}(i)
	}

	// Start writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					SetConfig("") // Only use empty path to avoid file system issues
				}
			}
		}(i)
	}

	// Run for a short duration
	time.Sleep(duration)
	close(done)
	wg.Wait()

	// Test should complete without race conditions or panics
}
