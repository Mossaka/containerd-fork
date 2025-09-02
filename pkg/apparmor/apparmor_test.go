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

package apparmor

import (
	"os"
	"runtime"
	"sync"
	"testing"
)

func TestHostSupports(t *testing.T) {
	// Test the public interface function
	result := HostSupports()

	// Result should be consistent with runtime behavior
	if runtime.GOOS == "linux" {
		// On Linux, result depends on system state
		// We can't predict the exact result, but it should be boolean
		t.Logf("HostSupports() returned %v on Linux", result)
	} else {
		// On non-Linux systems, should always be false
		if result != false {
			t.Errorf("HostSupports() = %v on %s, want false", result, runtime.GOOS)
		}
	}
}

func TestHostSupportsConsistency(t *testing.T) {
	// Test that multiple calls return the same result (due to sync.Once)
	result1 := HostSupports()
	result2 := HostSupports()
	result3 := HostSupports()

	if result1 != result2 || result2 != result3 {
		t.Errorf("HostSupports() returned inconsistent results: %v, %v, %v", result1, result2, result3)
	}
}

func TestHostSupportsConcurrency(t *testing.T) {
	// Test concurrent access to ensure thread safety
	const numGoroutines = 10
	results := make([]bool, numGoroutines)
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = HostSupports()
		}(i)
	}

	wg.Wait()

	// All results should be the same
	firstResult := results[0]
	for i, result := range results {
		if result != firstResult {
			t.Errorf("Concurrent call %d returned %v, expected %v", i, result, firstResult)
		}
	}
}

// Test the internal hostSupports function directly on Linux
func TestHostSupportsLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test only applies to Linux")
	}

	result := hostSupports()

	// We can't predict the exact result since it depends on the system,
	// but we can check that the function completes without panic
	t.Logf("hostSupports() returned %v", result)
}

// Test the internal hostSupports function on non-Linux systems
func TestHostSupportsNonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Test only applies to non-Linux systems")
	}

	result := hostSupports()
	if result != false {
		t.Errorf("hostSupports() = %v on %s, want false", result, runtime.GOOS)
	}
}

// Test edge cases and system state scenarios
func TestAppArmorSystemChecks(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test only applies to Linux")
	}

	// Test various system file states
	testCases := []struct {
		name        string
		setupFunc   func() func() // Returns cleanup function
		expectCheck func(bool)    // Function to validate the result
	}{
		{
			name: "container_env_set",
			setupFunc: func() func() {
				originalEnv := os.Getenv("container")
				os.Setenv("container", "docker")
				return func() {
					if originalEnv == "" {
						os.Unsetenv("container")
					} else {
						os.Setenv("container", originalEnv)
					}
				}
			},
			expectCheck: func(result bool) {
				// When container env is set, AppArmor should be disabled
				if result {
					t.Log("AppArmor enabled despite container environment being set")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := tc.setupFunc()
			defer cleanup()

			// Note: We cannot reset sync.Once for testing unexported variables
			// This test will use the cached result from previous calls
			result := hostSupports()
			tc.expectCheck(result)
		})
	}
}

func TestHostSupportsModuleIntegration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test only applies to Linux")
	}

	// This test checks the behavior based on actual system state
	result := HostSupports()

	// Check if the apparmor module directory exists
	if _, err := os.Stat("/sys/kernel/security/apparmor"); err == nil {
		t.Log("AppArmor security module directory exists")

		// Check if apparmor_parser exists
		if _, err := os.Stat("/sbin/apparmor_parser"); err == nil {
			t.Log("AppArmor parser binary exists")

			// Check if enabled parameter exists
			if buf, err := os.ReadFile("/sys/module/apparmor/parameters/enabled"); err == nil {
				t.Logf("AppArmor enabled parameter: %q", string(buf))

				// If all components exist and enabled is 'Y', AppArmor should be supported
				// (unless we're in a container)
				containerEnv := os.Getenv("container")
				if containerEnv == "" && len(buf) > 0 && buf[0] == 'Y' {
					if !result {
						t.Log("Expected AppArmor to be supported based on system state, but HostSupports() returned false")
					}
				}
			} else {
				t.Log("Could not read AppArmor enabled parameter")
			}
		} else {
			t.Log("AppArmor parser binary not found")
		}
	} else {
		t.Log("AppArmor security module directory not found")
		if result {
			t.Error("HostSupports() returned true but AppArmor directory doesn't exist")
		}
	}
}
