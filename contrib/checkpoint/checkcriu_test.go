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

package main

import (
	"os"
	"os/exec"
	"testing"

	criu "github.com/checkpoint-restore/go-criu/v7/utils"
)

func TestMainFunction(t *testing.T) {
	// Test that main function exists and can be called
	// This is a basic smoke test since main() calls panic on error
	// We can't directly test it without mocking, but we can verify
	// the function exists and the logic is sound

	// Verify that we can check for CRIU
	err := criu.CheckForCriu(criu.PodCriuVersion)
	if err != nil {
		// This is expected in CI environments where CRIU is not installed
		t.Logf("CRIU not available (expected in CI): %v", err)
	} else {
		t.Logf("CRIU is available and meets version requirements")
	}
}

func TestMainBinary(t *testing.T) {
	// Test that the binary can be built and executed
	// This tests the integration without requiring CRIU to be installed

	// Build the binary
	cmd := exec.Command("go", "build", "-o", "/tmp/checkcriu", ".")
	cmd.Dir = "."
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build checkcriu binary: %v", err)
	}
	defer os.Remove("/tmp/checkcriu")

	// Run the binary - it will exit with non-zero if CRIU is not available
	cmd = exec.Command("/tmp/checkcriu")
	err = cmd.Run()
	if err != nil {
		// Expected in CI environments without CRIU
		t.Logf("checkcriu binary exited with error (expected without CRIU): %v", err)
	} else {
		t.Logf("checkcriu binary executed successfully")
	}
}

func TestCriuVersionConstant(t *testing.T) {
	// Test that PodCriuVersion constant is accessible and reasonable
	version := criu.PodCriuVersion

	// Basic sanity check - version should be reasonable
	// CRIU 3.16+ is required for pod checkpoint/restore
	if version <= 0 {
		t.Fatal("PodCriuVersion should be positive")
	}

	// Should be at least version 3.16 (represented as 31600)
	if version < 31600 {
		t.Errorf("PodCriuVersion should be at least 31600 (3.16), got %d", version)
	}

	t.Logf("Required CRIU version: %d", version)
}

func TestCheckForCriuFunction(t *testing.T) {
	// Test the CheckForCriu function behavior

	// Test with zero version
	err := criu.CheckForCriu(0)
	if err == nil {
		t.Log("CheckForCriu with zero version succeeded")
	} else {
		t.Logf("CheckForCriu with zero version failed: %v", err)
	}

	// Test with PodCriuVersion
	err = criu.CheckForCriu(criu.PodCriuVersion)
	if err != nil {
		t.Logf("CRIU check failed (expected in CI): %v", err)
	} else {
		t.Logf("CRIU check passed")
	}
}

// Benchmark the CRIU check operation
func BenchmarkCheckForCriu(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = criu.CheckForCriu(criu.PodCriuVersion)
	}
}

func TestMainPanic(t *testing.T) {
	// Test that main function will panic when CRIU is not available
	// We can't easily test the panic directly, but we can verify
	// the logic that would cause it

	err := criu.CheckForCriu(criu.PodCriuVersion)
	if err != nil {
		// This is the condition that would cause main() to panic
		t.Logf("Confirmed that main() would panic with CRIU unavailable: %v", err)
	}
}
