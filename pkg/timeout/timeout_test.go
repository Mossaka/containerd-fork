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

package timeout

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	tests := []struct {
		name     string
		key      string
		timeout  time.Duration
		expected time.Duration
	}{
		{
			name:     "basic set and get",
			key:      "test-key",
			timeout:  5 * time.Second,
			expected: 5 * time.Second,
		},
		{
			name:     "zero timeout",
			key:      "zero-key",
			timeout:  0,
			expected: 0,
		},
		{
			name:     "large timeout",
			key:      "large-key",
			timeout:  24 * time.Hour,
			expected: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Set(tt.key, tt.timeout)
			actual := Get(tt.key)
			if actual != tt.expected {
				t.Errorf("Get(%q) = %v, want %v", tt.key, actual, tt.expected)
			}
		})
	}
}

func TestGetDefault(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	// Test getting non-existent key returns default timeout
	nonExistentKey := "non-existent-key"
	actual := Get(nonExistentKey)
	if actual != DefaultTimeout {
		t.Errorf("Get(%q) = %v, want %v (default timeout)", nonExistentKey, actual, DefaultTimeout)
	}
}

func TestWithContext(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	// Set a known timeout
	testKey := "test-context-key"
	testTimeout := 100 * time.Millisecond
	Set(testKey, testTimeout)

	ctx := context.Background()
	timeoutCtx, cancel := WithContext(ctx, testKey)
	defer cancel()

	// Check that the context has a deadline
	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Fatal("WithContext did not set a deadline on the context")
	}

	// The deadline should be approximately testTimeout from now
	expectedDeadline := time.Now().Add(testTimeout)
	if deadline.Sub(expectedDeadline) > 10*time.Millisecond {
		t.Errorf("Deadline is not close to expected: got %v, expected around %v", deadline, expectedDeadline)
	}

	// Test that the context times out
	select {
	case <-timeoutCtx.Done():
		// Context should timeout within reasonable time
	case <-time.After(200 * time.Millisecond):
		t.Error("Context did not timeout within expected duration")
	}
}

func TestWithContextDefault(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	// Use non-existent key which should use default timeout
	nonExistentKey := "non-existent-context-key"
	ctx := context.Background()
	timeoutCtx, cancel := WithContext(ctx, nonExistentKey)
	defer cancel()

	// Check that the context has a deadline based on default timeout
	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Fatal("WithContext did not set a deadline for non-existent key")
	}

	expectedDeadline := time.Now().Add(DefaultTimeout)
	if deadline.Sub(expectedDeadline) > 10*time.Millisecond {
		t.Errorf("Deadline is not close to expected default: got %v, expected around %v", deadline, expectedDeadline)
	}
}

func TestAll(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	// Test empty map
	all := All()
	if len(all) != 0 {
		t.Errorf("All() returned non-empty map when no timeouts set: %v", all)
	}

	// Set some timeouts
	testData := map[string]time.Duration{
		"key1": 1 * time.Second,
		"key2": 2 * time.Second,
		"key3": 3 * time.Second,
	}

	for key, timeout := range testData {
		Set(key, timeout)
	}

	// Get all timeouts
	all = All()
	if len(all) != len(testData) {
		t.Errorf("All() returned %d items, want %d", len(all), len(testData))
	}

	for key, expected := range testData {
		if actual, ok := all[key]; !ok {
			t.Errorf("All() missing key %q", key)
		} else if actual != expected {
			t.Errorf("All()[%q] = %v, want %v", key, actual, expected)
		}
	}

	// Ensure All() returns a copy, not the original map
	all["new-key"] = 99 * time.Second
	if Get("new-key") != DefaultTimeout {
		t.Error("All() returned a reference to the internal map instead of a copy")
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Test concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('A' + id%26)) // A-Z based on goroutine ID
				timeout := time.Duration(id+j) * time.Millisecond
				Set(key, timeout)
			}
		}(i)
	}

	// Test concurrent Get operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := string(rune('A' + id%26))
				Get(key) // Don't care about the result, just testing for races
			}
		}(i)
	}

	// Test concurrent All operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				All() // Don't care about the result, just testing for races
			}
		}()
	}

	wg.Wait()
}

func TestUpdateExistingTimeout(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	key := "update-test-key"
	
	// Set initial timeout
	initialTimeout := 1 * time.Second
	Set(key, initialTimeout)
	
	if actual := Get(key); actual != initialTimeout {
		t.Errorf("Get(%q) after first set = %v, want %v", key, actual, initialTimeout)
	}

	// Update timeout
	updatedTimeout := 5 * time.Second
	Set(key, updatedTimeout)
	
	if actual := Get(key); actual != updatedTimeout {
		t.Errorf("Get(%q) after update = %v, want %v", key, actual, updatedTimeout)
	}
}

func TestMultipleKeys(t *testing.T) {
	// Reset the timeouts map for clean testing
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	timeoutValues := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
		4 * time.Second,
		5 * time.Second,
	}

	// Set all keys
	for i, key := range keys {
		Set(key, timeoutValues[i])
	}

	// Verify all keys return correct values
	for i, key := range keys {
		if actual := Get(key); actual != timeoutValues[i] {
			t.Errorf("Get(%q) = %v, want %v", key, actual, timeoutValues[i])
		}
	}

	// Verify All() contains all keys
	all := All()
	if len(all) != len(keys) {
		t.Errorf("All() returned %d keys, want %d", len(all), len(keys))
	}

	for i, key := range keys {
		if actual, ok := all[key]; !ok {
			t.Errorf("All() missing key %q", key)
		} else if actual != timeoutValues[i] {
			t.Errorf("All()[%q] = %v, want %v", key, actual, timeoutValues[i])
		}
	}
}

func BenchmarkSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Set("benchmark-key", time.Duration(i)*time.Microsecond)
	}
}

func BenchmarkGet(b *testing.B) {
	Set("benchmark-key", 1*time.Second)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		Get("benchmark-key")
	}
}

func BenchmarkGetDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Get("non-existent-key")
	}
}

func BenchmarkAll(b *testing.B) {
	// Setup some data
	for i := 0; i < 100; i++ {
		Set(string(rune('A'+i%26))+string(rune('0'+i%10)), time.Duration(i)*time.Millisecond)
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		All()
	}
}

func BenchmarkWithContext(b *testing.B) {
	Set("benchmark-context-key", 1*time.Second)
	ctx := context.Background()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		timeoutCtx, cancel := WithContext(ctx, "benchmark-context-key")
		cancel()
		_ = timeoutCtx
	}
}