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

package truncindex

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestNewTruncIndex(t *testing.T) {
	// Test with empty slice
	idx := NewTruncIndex([]string{})
	if idx == nil {
		t.Fatal("NewTruncIndex returned nil")
	}
	if idx.ids == nil {
		t.Fatal("NewTruncIndex did not initialize ids map")
	}
	if idx.trie == nil {
		t.Fatal("NewTruncIndex did not initialize trie")
	}
	if len(idx.ids) != 0 {
		t.Errorf("expected empty ids map, got %d items", len(idx.ids))
	}
}

func TestNewTruncIndexWithIDs(t *testing.T) {
	ids := []string{
		"abc123def456",
		"def456ghi789",
		"ghi789jkl012",
	}

	idx := NewTruncIndex(ids)
	if idx == nil {
		t.Fatal("NewTruncIndex returned nil")
	}

	if len(idx.ids) != len(ids) {
		t.Errorf("expected %d ids, got %d", len(ids), len(idx.ids))
	}

	// Verify all IDs are stored
	for _, id := range ids {
		if _, exists := idx.ids[id]; !exists {
			t.Errorf("ID %q not found in index", id)
		}
	}
}

func TestAdd(t *testing.T) {
	idx := NewTruncIndex([]string{})

	// Test adding valid ID
	id := "abc123def456ghi789jkl012"
	if err := idx.Add(id); err != nil {
		t.Errorf("Add(%q) failed: %v", id, err)
	}

	// Verify ID was added
	if _, exists := idx.ids[id]; !exists {
		t.Errorf("ID %q was not added to index", id)
	}

	// Test adding duplicate ID
	if err := idx.Add(id); err == nil {
		t.Errorf("Add(%q) should fail for duplicate ID", id)
	} else if !strings.Contains(err.Error(), "id already exists") {
		t.Errorf("expected 'id already exists' error, got: %v", err)
	}
}

func TestAddInvalidIDs(t *testing.T) {
	idx := NewTruncIndex([]string{})

	tests := []struct {
		name        string
		id          string
		expectedErr error
	}{
		{
			name:        "empty ID",
			id:          "",
			expectedErr: ErrEmptyPrefix,
		},
		{
			name:        "ID with space",
			id:          "abc 123",
			expectedErr: ErrIllegalChar,
		},
		{
			name:        "ID with multiple spaces",
			id:          "abc 123 def",
			expectedErr: ErrIllegalChar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := idx.Add(tt.id)
			if err == nil {
				t.Errorf("Add(%q) should fail", tt.id)
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	id := "abc123def456ghi789jkl012"
	idx := NewTruncIndex([]string{id})

	// Verify ID exists before deletion
	if _, exists := idx.ids[id]; !exists {
		t.Fatalf("ID %q should exist before deletion", id)
	}

	// Test successful deletion
	if err := idx.Delete(id); err != nil {
		t.Errorf("Delete(%q) failed: %v", id, err)
	}

	// Verify ID was deleted
	if _, exists := idx.ids[id]; exists {
		t.Errorf("ID %q should not exist after deletion", id)
	}

	// Test deleting non-existent ID
	if err := idx.Delete(id); err == nil {
		t.Errorf("Delete(%q) should fail for non-existent ID", id)
	} else if !strings.Contains(err.Error(), "no such id") {
		t.Errorf("expected 'no such id' error, got: %v", err)
	}
}

func TestDeleteEmptyID(t *testing.T) {
	idx := NewTruncIndex([]string{})

	err := idx.Delete("")
	if err == nil {
		t.Error("Delete(\"\") should fail")
	}
	if !strings.Contains(err.Error(), "no such id") {
		t.Errorf("expected 'no such id' error, got: %v", err)
	}
}

func TestGet(t *testing.T) {
	ids := []string{
		"abc123def456ghi789jkl012",
		"def456ghi789jkl012mno345",
		"xyz789abc123def456ghi789",
	}
	idx := NewTruncIndex(ids)

	// Test getting exact ID
	for _, id := range ids {
		result, err := idx.Get(id)
		if err != nil {
			t.Errorf("Get(%q) failed: %v", id, err)
		}
		if result != id {
			t.Errorf("Get(%q) = %q, want %q", id, result, id)
		}
	}

	// Test getting with prefix
	result, err := idx.Get("abc123")
	if err != nil {
		t.Errorf("Get(\"abc123\") failed: %v", err)
	}
	if result != "abc123def456ghi789jkl012" {
		t.Errorf("Get(\"abc123\") = %q, want %q", result, "abc123def456ghi789jkl012")
	}

	// Test getting with unique short prefix
	result, err = idx.Get("xyz")
	if err != nil {
		t.Errorf("Get(\"xyz\") failed: %v", err)
	}
	if result != "xyz789abc123def456ghi789" {
		t.Errorf("Get(\"xyz\") = %q, want %q", result, "xyz789abc123def456ghi789")
	}
}

func TestGetEmptyPrefix(t *testing.T) {
	idx := NewTruncIndex([]string{"abc123"})

	_, err := idx.Get("")
	if err == nil {
		t.Error("Get(\"\") should fail")
	}
	if !errors.Is(err, ErrEmptyPrefix) {
		t.Errorf("expected ErrEmptyPrefix, got %v", err)
	}
}

func TestGetNotExist(t *testing.T) {
	idx := NewTruncIndex([]string{"abc123def456"})

	_, err := idx.Get("notexist")
	if err == nil {
		t.Error("Get(\"notexist\") should fail")
	}
	if !errors.Is(err, ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestGetAmbiguousPrefix(t *testing.T) {
	// Create IDs with common prefixes
	ids := []string{
		"abc123def456",
		"abc123ghi789",
		"abc123jkl012",
	}
	idx := NewTruncIndex(ids)

	// Test ambiguous prefix
	_, err := idx.Get("abc123")
	if err == nil {
		t.Error("Get(\"abc123\") should fail due to ambiguity")
	}

	var ambiguousErr ErrAmbiguousPrefix
	if !errors.As(err, &ambiguousErr) {
		t.Errorf("expected ErrAmbiguousPrefix, got %T: %v", err, err)
	}
	if ambiguousErr.prefix != "abc123" {
		t.Errorf("expected prefix 'abc123', got %q", ambiguousErr.prefix)
	}

	// Test that we can still get with longer, unique prefixes
	result, err := idx.Get("abc123d")
	if err != nil {
		t.Errorf("Get(\"abc123d\") failed: %v", err)
	}
	if result != "abc123def456" {
		t.Errorf("Get(\"abc123d\") = %q, want %q", result, "abc123def456")
	}
}

func TestIterate(t *testing.T) {
	ids := []string{
		"abc123def456",
		"def456ghi789",
		"ghi789jkl012",
	}
	idx := NewTruncIndex(ids)

	var found []string
	idx.Iterate(func(id string) {
		found = append(found, id)
	})

	if len(found) != len(ids) {
		t.Errorf("expected %d IDs from iteration, got %d", len(ids), len(found))
	}

	// Create a map for easy lookup
	foundMap := make(map[string]bool)
	for _, id := range found {
		foundMap[id] = true
	}

	// Verify all original IDs were found
	for _, id := range ids {
		if !foundMap[id] {
			t.Errorf("ID %q not found during iteration", id)
		}
	}
}

func TestIterateEmpty(t *testing.T) {
	idx := NewTruncIndex([]string{})

	var count int
	idx.Iterate(func(id string) {
		count++
	})

	if count != 0 {
		t.Errorf("expected 0 iterations for empty index, got %d", count)
	}
}

func TestErrAmbiguousPrefixError(t *testing.T) {
	err := ErrAmbiguousPrefix{prefix: "test123"}
	expected := "Multiple IDs found with provided prefix: test123"
	if err.Error() != expected {
		t.Errorf("ErrAmbiguousPrefix.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrEmptyPrefix",
			err:      ErrEmptyPrefix,
			expected: "prefix can't be empty",
		},
		{
			name:     "ErrIllegalChar",
			err:      ErrIllegalChar,
			expected: "illegal character: ' '",
		},
		{
			name:     "ErrNotExist",
			err:      ErrNotExist,
			expected: "ID does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	idx := NewTruncIndex([]string{})

	// Number of goroutines and operations per goroutine
	const numGoroutines = 10
	const numOpsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // Add, Get, Delete operations

	// Concurrent Add operations
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				id := fmt.Sprintf("id%d-%d", i, j)
				if err := idx.Add(id); err != nil {
					t.Errorf("Add(%q) failed: %v", id, err)
				}
			}
		}(i)
	}

	// Concurrent Get operations (after some adds)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				prefix := fmt.Sprintf("id%d", i)
				// This may fail due to ambiguity or not found, which is expected
				_, _ = idx.Get(prefix)
			}
		}(i)
	}

	// Concurrent Delete operations (some may fail if ID doesn't exist)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				id := fmt.Sprintf("id%d-%d", i, j)
				// May fail if already deleted or never existed, which is expected
				_ = idx.Delete(id)
			}
		}(i)
	}

	wg.Wait()

	// Test should complete without deadlock or panic
}

func TestRealWorldScenario(t *testing.T) {
	// Simulate container/image ID scenario
	containerIDs := []string{
		"sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"sha256:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
	}

	idx := NewTruncIndex(containerIDs)

	// Test common prefix scenarios
	tests := []struct {
		name       string
		prefix     string
		expectErr  bool
		expectedID string
		errorType  error
	}{
		{
			name:       "unique short prefix",
			prefix:     "sha256:123",
			expectErr:  false,
			expectedID: "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			name:       "unique medium prefix",
			prefix:     "sha256:abc",
			expectErr:  false,
			expectedID: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
		{
			name:      "ambiguous prefix",
			prefix:    "sha256:",
			expectErr: true,
			errorType: ErrAmbiguousPrefix{},
		},
		{
			name:      "non-existent prefix",
			prefix:    "sha256:999",
			expectErr: true,
			errorType: ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := idx.Get(tt.prefix)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Get(%q) should fail", tt.prefix)
					return
				}

				switch tt.errorType.(type) {
				case ErrAmbiguousPrefix:
					var ambErr ErrAmbiguousPrefix
					if !errors.As(err, &ambErr) {
						t.Errorf("expected ErrAmbiguousPrefix, got %T: %v", err, err)
					}
				default:
					if !errors.Is(err, tt.errorType) {
						t.Errorf("expected error %v, got %v", tt.errorType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Get(%q) failed: %v", tt.prefix, err)
					return
				}
				if result != tt.expectedID {
					t.Errorf("Get(%q) = %q, want %q", tt.prefix, result, tt.expectedID)
				}
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with very long IDs (64+ characters as mentioned in NewTruncIndex)
	longID := strings.Repeat("a", 128)
	idx := NewTruncIndex([]string{longID})

	result, err := idx.Get("aaaa")
	if err != nil {
		t.Errorf("Get with long ID failed: %v", err)
	}
	if result != longID {
		t.Errorf("Get with long ID returned wrong result")
	}

	// Test with single character IDs
	idx2 := NewTruncIndex([]string{"a", "b", "c"})

	result, err = idx2.Get("a")
	if err != nil {
		t.Errorf("Get single char ID failed: %v", err)
	}
	if result != "a" {
		t.Errorf("Get single char ID = %q, want 'a'", result)
	}
}
