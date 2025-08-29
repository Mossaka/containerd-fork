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

package v2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	client "github.com/containerd/containerd/v2/pkg/shim"
)

func TestParseStartResponse_JSONFormat(t *testing.T) {
	params := client.BootstrapParams{
		Version:  3,
		Address:  "unix:///tmp/test.sock",
		Protocol: "grpc",
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal test params: %v", err)
	}

	result, err := parseStartResponse(jsonData)
	if err != nil {
		t.Fatalf("parseStartResponse failed: %v", err)
	}

	if result.Version != 3 {
		t.Errorf("expected version 3, got %d", result.Version)
	}
	if result.Address != "unix:///tmp/test.sock" {
		t.Errorf("expected address 'unix:///tmp/test.sock', got %q", result.Address)
	}
	if result.Protocol != "grpc" {
		t.Errorf("expected protocol 'grpc', got %q", result.Protocol)
	}
}

func TestParseStartResponse_LegacyFormat(t *testing.T) {
	legacyResponse := []byte("unix:///tmp/legacy.sock")

	result, err := parseStartResponse(legacyResponse)
	if err != nil {
		t.Fatalf("parseStartResponse failed: %v", err)
	}

	if result.Version != 2 {
		t.Errorf("expected version 2 for legacy format, got %d", result.Version)
	}
	if result.Address != "unix:///tmp/legacy.sock" {
		t.Errorf("expected address 'unix:///tmp/legacy.sock', got %q", result.Address)
	}
	if result.Protocol != "ttrpc" {
		t.Errorf("expected protocol 'ttrpc' for legacy format, got %q", result.Protocol)
	}
}

func TestParseStartResponse_InvalidJSON(t *testing.T) {
	invalidJSON := []byte("{invalid json}")

	result, err := parseStartResponse(invalidJSON)
	if err != nil {
		t.Fatalf("parseStartResponse failed: %v", err)
	}

	// Should fall back to legacy format
	if result.Version != 2 {
		t.Errorf("expected version 2 for invalid JSON, got %d", result.Version)
	}
	if result.Protocol != "ttrpc" {
		t.Errorf("expected protocol 'ttrpc' for invalid JSON, got %q", result.Protocol)
	}
}

func TestParseStartResponse_UnsupportedVersion(t *testing.T) {
	params := client.BootstrapParams{
		Version:  999, // Very high version
		Address:  "unix:///tmp/test.sock",
		Protocol: "grpc",
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal test params: %v", err)
	}

	_, err = parseStartResponse(jsonData)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}

	if !containsString(err.Error(), "unsupported shim version") {
		t.Errorf("expected error about unsupported version, got: %v", err)
	}
}

func TestWriteAndReadBootstrapParams(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "bootstrap-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test data
	params := client.BootstrapParams{
		Version:  3,
		Address:  "unix:///tmp/test.sock",
		Protocol: "grpc",
	}

	filePath := filepath.Join(tempDir, "bootstrap.json")

	// Write bootstrap params
	err = writeBootstrapParams(filePath, params)
	if err != nil {
		t.Fatalf("writeBootstrapParams failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("bootstrap file was not created")
	}

	// Read bootstrap params back
	readParams, err := readBootstrapParams(filePath)
	if err != nil {
		t.Fatalf("readBootstrapParams failed: %v", err)
	}

	// Verify the data matches
	if readParams.Version != params.Version {
		t.Errorf("version mismatch: expected %d, got %d", params.Version, readParams.Version)
	}
	if readParams.Address != params.Address {
		t.Errorf("address mismatch: expected %q, got %q", params.Address, readParams.Address)
	}
	if readParams.Protocol != params.Protocol {
		t.Errorf("protocol mismatch: expected %q, got %q", params.Protocol, readParams.Protocol)
	}
}

func TestReadBootstrapParams_NonexistentFile(t *testing.T) {
	_, err := readBootstrapParams("/nonexistent/path/bootstrap.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWriteBootstrapParams_InvalidPath(t *testing.T) {
	params := client.BootstrapParams{
		Version:  3,
		Address:  "unix:///tmp/test.sock",
		Protocol: "grpc",
	}

	// Try to write to an invalid path (directory that doesn't exist)
	err := writeBootstrapParams("/nonexistent/dir/bootstrap.json", params)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
