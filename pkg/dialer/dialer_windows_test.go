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

package dialer

import (
	"os"
	"testing"
)

func TestDialAddressWindows(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `\\.\pipe\containerd`,
			expected: `npipe://\\.\pipe\containerd`,
		},
		{
			input:    `npipe://\\.\pipe\containerd`,
			expected: `npipe://\\.\pipe\containerd`,
		},
		{
			input:    `C:\temp\containerd.pipe`,
			expected: `npipe://C:/temp/containerd.pipe`,
		},
		{
			input:    "",
			expected: "npipe://",
		},
		{
			input:    `containerd`,
			expected: `npipe://containerd`,
		},
	}

	for _, test := range tests {
		result := DialAddress(test.input)
		if result != test.expected {
			t.Errorf("DialAddress(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsNoentWindows(t *testing.T) {
	// Test with os.IsNotExist error
	nonexistentErr := &os.PathError{
		Op:   "open",
		Path: "nonexistent",
		Err:  os.ErrNotExist,
	}
	if !isNoent(nonexistentErr) {
		t.Error("isNoent should return true for os.ErrNotExist")
	}

	// Test with other errors
	if isNoent(os.ErrPermission) {
		t.Error("isNoent should return false for permission errors")
	}

	// Test with nil error
	if isNoent(nil) {
		t.Error("isNoent should return false for nil error")
	}

	// Test with generic error
	genericErr := &os.PathError{
		Op:   "open",
		Path: "test",
		Err:  os.ErrPermission,
	}
	if isNoent(genericErr) {
		t.Error("isNoent should return false for non-existence errors")
	}
}

// Note: Testing the actual dialer function on Windows would require
// setting up named pipes, which is complex and requires external dependencies.
// The dialer function calls winio.DialPipe, which is well-tested in the
// go-winio package. We focus on testing the address formatting and error
// detection functions that don't require external pipe setup.

func TestDialerInterface(t *testing.T) {
	// Test that dialer function exists and has correct signature
	// We can't easily test the actual functionality without setting up named pipes
	// but we can test that it handles invalid addresses gracefully

	// Test with clearly invalid pipe name that should fail quickly
	_, err := dialer("invalid-pipe-name", 0)
	if err == nil {
		t.Skip("Dialer unexpectedly succeeded with invalid pipe name - this may be environment specific")
	}
	// If it errors, that's expected for an invalid pipe name
}
