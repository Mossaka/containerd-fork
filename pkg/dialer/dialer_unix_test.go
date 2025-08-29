//go:build !windows

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
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDialAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/tmp/test.sock",
			expected: "unix:///tmp/test.sock",
		},
		{
			input:    "test.sock",
			expected: "unix://test.sock",
		},
		{
			input:    "/var/run/containerd/containerd.sock",
			expected: "unix:///var/run/containerd/containerd.sock",
		},
		{
			input:    "",
			expected: "unix://",
		},
	}

	for _, test := range tests {
		result := DialAddress(test.input)
		if result != test.expected {
			t.Errorf("DialAddress(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsNoent(t *testing.T) {
	// Test with syscall.ENOENT error
	enoentErr := syscall.ENOENT
	if !isNoent(enoentErr) {
		t.Error("isNoent should return true for syscall.ENOENT")
	}

	// Test with wrapped ENOENT error
	wrappedErr := &net.OpError{
		Op:   "dial",
		Net:  "unix",
		Addr: nil,
		Err:  syscall.ENOENT,
	}
	if !isNoent(wrappedErr) {
		t.Error("isNoent should return true for wrapped ENOENT")
	}

	// Test with other errors
	otherErr := syscall.ECONNREFUSED
	if isNoent(otherErr) {
		t.Error("isNoent should return false for non-ENOENT errors")
	}

	// Test with nil error
	if isNoent(nil) {
		t.Error("isNoent should return false for nil error")
	}

	// Test with generic error
	genericErr := &net.OpError{
		Op:   "dial",
		Net:  "unix",
		Addr: nil,
		Err:  syscall.ECONNREFUSED,
	}
	if isNoent(genericErr) {
		t.Error("isNoent should return false for non-ENOENT network errors")
	}
}

func TestDialer(t *testing.T) {
	// Create a temporary directory for test sockets
	tmpDir, err := os.MkdirTemp("", "containerd-dialer-test")
	if err != nil {
		t.Fatal("failed to create temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create a test server
	server, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal("failed to create test server:", err)
	}
	defer server.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Test successful connection
	conn, err := dialer(socketPath, 5*time.Second)
	if err != nil {
		t.Errorf("dialer failed: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
}

func TestDialerWithUnixPrefix(t *testing.T) {
	// Create a temporary directory for test sockets
	tmpDir, err := os.MkdirTemp("", "containerd-dialer-test")
	if err != nil {
		t.Fatal("failed to create temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create a test server
	server, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal("failed to create test server:", err)
	}
	defer server.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Test with unix:// prefix (should be stripped)
	conn, err := dialer("unix://"+socketPath, 5*time.Second)
	if err != nil {
		t.Errorf("dialer with unix:// prefix failed: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
}

func TestDialerNonexistentSocket(t *testing.T) {
	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	_, err := dialer(nonexistentPath, 100*time.Millisecond)
	if err == nil {
		t.Error("expected error when dialing nonexistent socket, got nil")
	}
}

func TestDialerTimeout(t *testing.T) {
	// Test that dialer respects timeout parameter
	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	start := time.Now()
	_, err := dialer(nonexistentPath, 50*time.Millisecond)
	duration := time.Since(start)

	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	// Should complete within reasonable time (timeout + some overhead)
	if duration > 200*time.Millisecond {
		t.Errorf("dialer took too long: %v", duration)
	}
}

func TestDialerZeroTimeout(t *testing.T) {
	// Test that dialer with zero timeout works correctly
	// The net.DialTimeout function treats zero timeout as no timeout
	tmpDir, err := os.MkdirTemp("", "containerd-dialer-test")
	if err != nil {
		t.Fatal("failed to create temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create a test server
	server, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal("failed to create test server:", err)
	}
	defer server.Close()

	// Accept connections in background
	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Test with zero timeout - net.DialTimeout treats 0 as no timeout
	conn, err := dialer(socketPath, 0)
	if err != nil {
		t.Errorf("dialer with zero timeout failed: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
}
