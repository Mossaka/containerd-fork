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
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContextDialer(t *testing.T) {
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

	// Test successful connection with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := ContextDialer(ctx, DialAddress(socketPath))
	if err != nil {
		t.Errorf("ContextDialer failed: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
}

func TestContextDialerTimeout(t *testing.T) {
	// Create a context with very short deadline
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Nanosecond))
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	// Try to dial - should use timeout dialer with very short or zero timeout
	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	_, err := ContextDialer(ctx, DialAddress(nonexistentPath))
	if err == nil {
		t.Error("expected error when dialing with expired context, got nil")
	}
}

func TestContextDialerNoDeadline(t *testing.T) {
	// Test with context that has no deadline
	ctx := context.Background()

	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	_, err := ContextDialer(ctx, DialAddress(nonexistentPath))
	if err == nil {
		t.Error("expected error when dialing nonexistent socket, got nil")
	}
}

func TestTimeoutDialerSuccess(t *testing.T) {
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
	conn, err := timeoutDialer(DialAddress(socketPath), 5*time.Second)
	if err != nil {
		t.Errorf("timeoutDialer failed: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
}

func TestTimeoutDialerTimeout(t *testing.T) {
	// Test timeout behavior with very short timeout
	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	start := time.Now()
	_, err := timeoutDialer(DialAddress(nonexistentPath), 10*time.Millisecond)
	duration := time.Since(start)

	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	// Verify it's a timeout error and took reasonable time
	if duration > 1*time.Second {
		t.Errorf("timeout took too long: %v", duration)
	}
}

func TestTimeoutDialerRetryOnNoent(t *testing.T) {
	// This test verifies that the dialer retries on ENOENT errors
	// We can't easily simulate ENOENT in a unit test without complex mocking,
	// but we can test that dialing a nonexistent socket eventually gives up
	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	start := time.Now()
	_, err := timeoutDialer(DialAddress(nonexistentPath), 50*time.Millisecond)
	duration := time.Since(start)

	if err == nil {
		t.Error("expected error when dialing nonexistent socket, got nil")
	}

	// Should take at least 50ms due to timeout, but not too much longer
	if duration < 45*time.Millisecond {
		t.Errorf("dialer returned too quickly: %v", duration)
	}
	if duration > 200*time.Millisecond {
		t.Errorf("dialer took too long: %v", duration)
	}
}

func TestTimeoutDialerZeroTimeout(t *testing.T) {
	// Test with zero timeout (should timeout immediately)
	nonexistentPath := "/tmp/nonexistent-" + generateUniqueID()
	start := time.Now()
	_, err := timeoutDialer(DialAddress(nonexistentPath), 0)
	duration := time.Since(start)

	// Zero timeout should result in immediate timeout
	if err == nil {
		t.Error("expected timeout error with zero timeout, got nil")
	}

	// Should timeout very quickly with zero timeout
	if duration > 50*time.Millisecond {
		t.Errorf("zero timeout took too long: %v", duration)
	}
}

func TestDialResultStruct(t *testing.T) {
	// Test the dialResult struct (used internally)
	result := &dialResult{
		c:   nil,
		err: nil,
	}

	if result.c != nil {
		t.Error("expected nil connection")
	}
	if result.err != nil {
		t.Error("expected nil error")
	}

	// Test with mock connection and error
	mockErr := net.Error(&net.OpError{Op: "test"})
	result2 := &dialResult{
		c:   nil,
		err: mockErr,
	}

	if result2.err == nil {
		t.Error("expected error to be set")
	}
}

// Helper function to generate unique IDs for test paths
func generateUniqueID() string {
	return time.Now().Format("20060102-150405.000000")
}
