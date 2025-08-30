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
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestDefaultTimeout(t *testing.T) {
	expected := 1 * time.Second
	if DefaultTimeout != expected {
		t.Errorf("Expected DefaultTimeout to be %v, got %v", expected, DefaultTimeout)
	}
}

func TestSetAndGet(t *testing.T) {
	key := "test-key"
	duration := 5 * time.Second

	Set(key, duration)
	result := Get(key)

	if result != duration {
		t.Errorf("Expected Get(%q) to return %v, got %v", key, duration, result)
	}
}

func TestGetNonExistentKey(t *testing.T) {
	key := "non-existent-key-12345"
	result := Get(key)

	if result != DefaultTimeout {
		t.Errorf("Expected Get(%q) to return DefaultTimeout (%v), got %v", key, DefaultTimeout, result)
	}
}

func TestSetOverwritesExistingKey(t *testing.T) {
	key := "overwrite-key"
	firstDuration := 2 * time.Second
	secondDuration := 10 * time.Second

	Set(key, firstDuration)
	firstResult := Get(key)
	if firstResult != firstDuration {
		t.Errorf("Expected first Get(%q) to return %v, got %v", key, firstDuration, firstResult)
	}

	Set(key, secondDuration)
	secondResult := Get(key)
	if secondResult != secondDuration {
		t.Errorf("Expected second Get(%q) to return %v, got %v", key, secondDuration, secondResult)
	}
}

func TestSetZeroDuration(t *testing.T) {
	key := "zero-duration-key"
	duration := 0 * time.Second

	Set(key, duration)
	result := Get(key)

	if result != duration {
		t.Errorf("Expected Get(%q) to return %v, got %v", key, duration, result)
	}
}

func TestSetNegativeDuration(t *testing.T) {
	key := "negative-duration-key"
	duration := -1 * time.Second

	Set(key, duration)
	result := Get(key)

	if result != duration {
		t.Errorf("Expected Get(%q) to return %v, got %v", key, duration, result)
	}
}

func TestConcurrentSetAndGet(t *testing.T) {
	const numGoroutines = 10
	const numOperations = 100
	var wg sync.WaitGroup

	// Test concurrent Set operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d", id)
				duration := time.Duration(id+j) * time.Millisecond
				Set(key, duration)
			}
		}(i)
	}

	// Test concurrent Get operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d", id)
				Get(key) // Just call it, results may vary due to concurrent writes
			}
		}(i)
	}

	wg.Wait()

	// Test that the final state is consistent
	for i := 0; i < numGoroutines; i++ {
		key := fmt.Sprintf("concurrent-key-%d", i)
		result := Get(key)
		if result == DefaultTimeout {
			t.Errorf("Expected key %q to be set, but got DefaultTimeout", key)
		}
	}
}

func TestWithContext(t *testing.T) {
	key := "context-test-key"
	duration := 100 * time.Millisecond
	Set(key, duration)

	parentCtx := context.Background()
	ctx, cancel := WithContext(parentCtx, key)
	defer cancel()

	// Verify context has the expected deadline
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected context to have a deadline")
	}

	// Check that deadline is approximately correct (within 10ms tolerance)
	expectedDeadline := time.Now().Add(duration)
	timeDiff := deadline.Sub(expectedDeadline)
	if timeDiff > 10*time.Millisecond || timeDiff < -10*time.Millisecond {
		t.Errorf("Context deadline %v is not close to expected %v (diff: %v)", deadline, expectedDeadline, timeDiff)
	}
}

func TestWithContextNonExistentKey(t *testing.T) {
	key := "non-existent-context-key"
	parentCtx := context.Background()
	ctx, cancel := WithContext(parentCtx, key)
	defer cancel()

	// Should use DefaultTimeout
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		t.Error("Expected context to have a deadline")
	}

	expectedDeadline := time.Now().Add(DefaultTimeout)
	timeDiff := deadline.Sub(expectedDeadline)
	if timeDiff > 10*time.Millisecond || timeDiff < -10*time.Millisecond {
		t.Errorf("Context deadline %v is not close to expected default %v (diff: %v)", deadline, expectedDeadline, timeDiff)
	}
}

func TestWithContextCancellation(t *testing.T) {
	key := "cancellation-test-key"
	duration := 50 * time.Millisecond
	Set(key, duration)

	parentCtx := context.Background()
	ctx, cancel := WithContext(parentCtx, key)

	// Cancel immediately
	cancel()

	select {
	case <-ctx.Done():
		// Expected
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context error to be Canceled, got %v", ctx.Err())
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("Context should have been cancelled immediately")
	}
}

func TestWithContextTimeout(t *testing.T) {
	key := "timeout-test-key"
	duration := 10 * time.Millisecond
	Set(key, duration)

	parentCtx := context.Background()
	ctx, cancel := WithContext(parentCtx, key)
	defer cancel()

	// Wait for timeout
	select {
	case <-ctx.Done():
		// Expected timeout
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected context error to be DeadlineExceeded, got %v", ctx.Err())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

func TestAll(t *testing.T) {
	// Clear any existing state
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	// Set up test data
	testData := map[string]time.Duration{
		"key1": 1 * time.Second,
		"key2": 2 * time.Second,
		"key3": 500 * time.Millisecond,
	}

	for key, duration := range testData {
		Set(key, duration)
	}

	result := All()

	if !reflect.DeepEqual(result, testData) {
		t.Errorf("Expected All() to return %v, got %v", testData, result)
	}
}

func TestAllEmptyMap(t *testing.T) {
	// Clear any existing state
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	result := All()
	expected := make(map[string]time.Duration)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected All() to return empty map %v, got %v", expected, result)
	}
}

func TestAllReturnsCopy(t *testing.T) {
	// Clear and set up test data
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	key := "copy-test-key"
	duration := 3 * time.Second
	Set(key, duration)

	result1 := All()
	result2 := All()

	// Modify one result
	result1["new-key"] = 5 * time.Second

	// Other result should be unaffected
	if _, exists := result2["new-key"]; exists {
		t.Error("All() should return a copy, modifications should not affect other calls")
	}

	// Original timeouts should be unaffected
	original := Get(key)
	if original != duration {
		t.Errorf("Original timeout should be unchanged, got %v expected %v", original, duration)
	}
}

func TestConcurrentAll(t *testing.T) {
	// Set up test data
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	testKeys := []string{"concurrent1", "concurrent2", "concurrent3"}
	for _, key := range testKeys {
		Set(key, time.Duration(len(key))*time.Second)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make([]map[string]time.Duration, numGoroutines)

	// Concurrent calls to All()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = All()
		}(i)
	}

	wg.Wait()

	// All results should be consistent
	expected := results[0]
	for i, result := range results {
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Concurrent All() call %d returned different result: got %v, expected %v", i, result, expected)
		}
	}
}

func TestMixedConcurrentOperations(t *testing.T) {
	// Clear state
	mu.Lock()
	timeouts = make(map[string]time.Duration)
	mu.Unlock()

	const duration = 50 * time.Millisecond
	const numGoroutines = 5
	var wg sync.WaitGroup

	// Mixed concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("mixed-key-%d", id)

			// Set
			Set(key, time.Duration(id)*time.Millisecond)

			// Get
			Get(key)

			// WithContext
			_, cancel := WithContext(context.Background(), key)
			cancel()

			// All
			All()
		}(i)
	}

	wg.Wait()

	// Verify final state
	all := All()
	if len(all) != numGoroutines {
		t.Errorf("Expected %d keys in final state, got %d", numGoroutines, len(all))
	}
}
