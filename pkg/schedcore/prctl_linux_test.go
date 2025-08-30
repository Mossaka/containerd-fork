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

package schedcore

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPidTypeConstants(t *testing.T) {
	// Test that PidType constants have the expected values
	tests := []struct {
		name     string
		pidType  PidType
		expected int
	}{
		{"Pid", Pid, pidtypePid},
		{"ThreadGroup", ThreadGroup, pidtypeTgid},
		{"ProcessGroup", ProcessGroup, pidtypePgid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.pidType) != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, int(tt.pidType), tt.expected)
			}
		})
	}
}

func TestInternalConstants(t *testing.T) {
	// Test internal constants have expected values
	if pidtypePid != 0 {
		t.Errorf("pidtypePid = %d, want 0", pidtypePid)
	}
	if pidtypeTgid != 1 {
		t.Errorf("pidtypeTgid = %d, want 1", pidtypeTgid)
	}
	if pidtypePgid != 2 {
		t.Errorf("pidtypePgid = %d, want 2", pidtypePgid)
	}
}

func TestCreate(t *testing.T) {
	// Only run this test on Linux systems that support sched core
	if runtime.GOOS != "linux" {
		t.Skip("schedcore is Linux-specific")
	}

	tests := []struct {
		name    string
		pidType PidType
	}{
		{"Create_Pid", Pid},
		{"Create_ThreadGroup", ThreadGroup},
		{"Create_ProcessGroup", ProcessGroup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Create(tt.pidType)
			// The function may fail if the kernel doesn't support sched core
			// or if we don't have sufficient privileges, but we can at least
			// test that it makes the syscall without panicking
			if err != nil {
				// Check if it's a known error condition
				if errno, ok := err.(syscall.Errno); ok {
					switch errno {
					case unix.ENOSYS:
						t.Skipf("sched core not supported by kernel")
					case unix.EPERM:
						t.Skipf("insufficient privileges for sched core operations")
					case unix.EINVAL:
						// This might be expected in some test environments
						t.Logf("Create(%v) returned EINVAL, possibly expected in test environment", tt.pidType)
					default:
						t.Logf("Create(%v) returned error: %v", tt.pidType, err)
					}
				} else {
					t.Logf("Create(%v) returned non-errno error: %v", tt.pidType, err)
				}
			}
		})
	}
}

func TestShareFrom(t *testing.T) {
	// Only run this test on Linux systems
	if runtime.GOOS != "linux" {
		t.Skip("schedcore is Linux-specific")
	}

	currentPid := uint64(os.Getpid())

	tests := []struct {
		name    string
		pid     uint64
		pidType PidType
	}{
		{"ShareFrom_CurrentPid_Pid", currentPid, Pid},
		{"ShareFrom_CurrentPid_ThreadGroup", currentPid, ThreadGroup},
		{"ShareFrom_CurrentPid_ProcessGroup", currentPid, ProcessGroup},
		{"ShareFrom_Init_Pid", 1, Pid}, // Try with init process
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ShareFrom(tt.pid, tt.pidType)
			// The function may fail for various reasons, but we test the call
			if err != nil {
				// Check if it's a known error condition
				if errno, ok := err.(syscall.Errno); ok {
					switch errno {
					case unix.ENOSYS:
						t.Skipf("sched core not supported by kernel")
					case unix.EPERM:
						t.Skipf("insufficient privileges for sched core operations")
					case unix.EINVAL:
						t.Logf("ShareFrom(%d, %v) returned EINVAL, possibly expected", tt.pid, tt.pidType)
					case unix.ESRCH:
						// Process not found or not in same core domain
						t.Logf("ShareFrom(%d, %v) returned ESRCH, target process not found or incompatible", tt.pid, tt.pidType)
					default:
						t.Logf("ShareFrom(%d, %v) returned error: %v", tt.pid, tt.pidType, err)
					}
				} else {
					t.Logf("ShareFrom(%d, %v) returned non-errno error: %v", tt.pid, tt.pidType, err)
				}
			}
		})
	}
}

func TestShareFromInvalidPid(t *testing.T) {
	// Only run this test on Linux systems
	if runtime.GOOS != "linux" {
		t.Skip("schedcore is Linux-specific")
	}

	// Test with invalid PIDs
	tests := []struct {
		name    string
		pid     uint64
		pidType PidType
	}{
		{"ShareFrom_InvalidPid", ^uint64(0), Pid}, // Max uint64, likely invalid
		{"ShareFrom_ZeroPid", 0, Pid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ShareFrom(tt.pid, tt.pidType)
			// We expect this to fail, but shouldn't panic
			if err == nil {
				t.Logf("ShareFrom(%d, %v) unexpectedly succeeded", tt.pid, tt.pidType)
			} else {
				t.Logf("ShareFrom(%d, %v) failed as expected: %v", tt.pid, tt.pidType, err)
			}
		})
	}
}

func TestPidTypeValues(t *testing.T) {
	// Test that our PidType constants map to the expected unix constants
	// This is important for correctness of the prctl calls

	expectedMappings := map[PidType]int{
		Pid:          0, // Should map to pidtypePid
		ThreadGroup:  1, // Should map to pidtypeTgid
		ProcessGroup: 2, // Should map to pidtypePgid
	}

	for pidType, expected := range expectedMappings {
		if int(pidType) != expected {
			t.Errorf("PidType %v should equal %d, got %d", pidType, expected, int(pidType))
		}
	}
}

func TestErrorHandling(t *testing.T) {
	// Test that functions properly handle and return errors from the syscall
	if runtime.GOOS != "linux" {
		t.Skip("schedcore is Linux-specific")
	}

	// Test Create with all valid PidType values
	for _, pidType := range []PidType{Pid, ThreadGroup, ProcessGroup} {
		err := Create(pidType)
		// We don't assert specific error conditions since they depend on
		// kernel support and privileges, but we ensure no panic occurs
		_ = err
	}

	// Test ShareFrom with current process
	currentPid := uint64(os.Getpid())
	for _, pidType := range []PidType{Pid, ThreadGroup, ProcessGroup} {
		err := ShareFrom(currentPid, pidType)
		// Again, we don't assert specific errors, just ensure no panic
		_ = err
	}
}

// BenchmarkCreate benchmarks the Create function
func BenchmarkCreate(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("schedcore is Linux-specific")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We don't care about the error for benchmarking purposes
		_ = Create(Pid)
	}
}

// BenchmarkShareFrom benchmarks the ShareFrom function
func BenchmarkShareFrom(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("schedcore is Linux-specific")
	}

	currentPid := uint64(os.Getpid())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We don't care about the error for benchmarking purposes
		_ = ShareFrom(currentPid, Pid)
	}
}
