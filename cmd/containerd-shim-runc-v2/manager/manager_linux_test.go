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

package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func TestNewShimManager(t *testing.T) {
	name := "test-runtime"
	mgr := NewShimManager(name)

	if mgr == nil {
		t.Fatal("NewShimManager returned nil")
	}

	if mgr.Name() != name {
		t.Errorf("expected name %q, got %q", name, mgr.Name())
	}
}

func TestManagerName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"runc.v2", "runc.v2"},
		{"", ""},
		{"custom-runtime", "custom-runtime"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("name_%s", strings.ReplaceAll(tt.name, ".", "_")), func(t *testing.T) {
			mgr := &manager{name: tt.name}
			if got := mgr.Name(); got != tt.expected {
				t.Errorf("Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestReadSpecInvalidFile(t *testing.T) {
	// Test with non-existent config.json file
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Create a temporary directory without config.json
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	_, err = readSpec()
	if err == nil {
		t.Fatal("readSpec() should fail when config.json doesn't exist")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error, got %v", err)
	}
}

func TestReadSpecValidFile(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Create a temporary directory with valid config.json
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create valid spec data
	specData := spec{
		Annotations: map[string]string{
			"io.containerd.runc.v2.group":  "test-group",
			"io.kubernetes.cri.sandbox-id": "sandbox-123",
			"custom.annotation":            "custom-value",
		},
	}

	// Write config.json file
	configFile := filepath.Join(tmpDir, "config.json")
	data, err := json.Marshal(specData)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Test readSpec
	result, err := readSpec()
	if err != nil {
		t.Fatalf("readSpec() failed: %v", err)
	}

	if result == nil {
		t.Fatal("readSpec() returned nil spec")
	}

	if result.Annotations == nil {
		t.Fatal("readSpec() returned spec with nil annotations")
	}

	// Verify annotations were read correctly
	expected := map[string]string{
		"io.containerd.runc.v2.group":  "test-group",
		"io.kubernetes.cri.sandbox-id": "sandbox-123",
		"custom.annotation":            "custom-value",
	}

	for key, expectedValue := range expected {
		if actualValue, ok := result.Annotations[key]; !ok {
			t.Errorf("missing annotation %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("annotation %q = %q, want %q", key, actualValue, expectedValue)
		}
	}
}

func TestReadSpecInvalidJSON(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Create a temporary directory with invalid config.json
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Write invalid JSON
	configFile := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configFile, []byte("invalid json {"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test readSpec should fail
	_, err = readSpec()
	if err == nil {
		t.Fatal("readSpec() should fail with invalid JSON")
	}

	// Should be a JSON parsing error
	var jsonErr *json.SyntaxError
	if !errors.As(err, &jsonErr) {
		t.Errorf("expected json.SyntaxError, got %T: %v", err, err)
	}
}

func TestNewCommandBasic(t *testing.T) {
	ctx := namespaces.WithNamespace(context.Background(), "test-namespace")

	cmd, err := newCommand(ctx, "test-id", "test-address", "test-ttrpc-address", false)
	if err != nil {
		t.Fatalf("newCommand() failed: %v", err)
	}

	if cmd == nil {
		t.Fatal("newCommand() returned nil command")
	}

	// Verify command arguments contain expected values
	args := cmd.Args
	if len(args) < 6 {
		t.Fatalf("expected at least 6 arguments, got %d: %v", len(args), args)
	}

	// Check for required arguments
	foundNamespace := false
	foundID := false
	foundAddress := false

	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "-namespace":
			if args[i+1] == "test-namespace" {
				foundNamespace = true
			}
		case "-id":
			if args[i+1] == "test-id" {
				foundID = true
			}
		case "-address":
			if args[i+1] == "test-address" {
				foundAddress = true
			}
		}
	}

	if !foundNamespace {
		t.Error("command missing -namespace test-namespace")
	}
	if !foundID {
		t.Error("command missing -id test-id")
	}
	if !foundAddress {
		t.Error("command missing -address test-address")
	}

	// Verify environment variables
	foundGOMAXPROCS := false
	foundOTEL := false
	for _, env := range cmd.Env {
		if env == "GOMAXPROCS=4" {
			foundGOMAXPROCS = true
		}
		if strings.HasPrefix(env, "OTEL_SERVICE_NAME=containerd-shim-test-id") {
			foundOTEL = true
		}
	}

	if !foundGOMAXPROCS {
		t.Error("command missing GOMAXPROCS=4 environment variable")
	}
	if !foundOTEL {
		t.Error("command missing OTEL_SERVICE_NAME environment variable")
	}

	// Verify SysProcAttr
	if cmd.SysProcAttr == nil {
		t.Error("command missing SysProcAttr")
	} else if !cmd.SysProcAttr.Setpgid {
		t.Error("command SysProcAttr.Setpgid should be true")
	}
}

func TestNewCommandWithDebug(t *testing.T) {
	ctx := namespaces.WithNamespace(context.Background(), "test-namespace")

	cmd, err := newCommand(ctx, "test-id", "test-address", "test-ttrpc-address", true)
	if err != nil {
		t.Fatalf("newCommand() with debug failed: %v", err)
	}

	// Check that -debug flag is present
	foundDebug := false
	for _, arg := range cmd.Args {
		if arg == "-debug" {
			foundDebug = true
			break
		}
	}

	if !foundDebug {
		t.Error("command with debug=true missing -debug flag")
	}
}

func TestNewCommandNoNamespace(t *testing.T) {
	ctx := context.Background() // No namespace set

	_, err := newCommand(ctx, "test-id", "test-address", "test-ttrpc-address", false)
	if err == nil {
		t.Fatal("newCommand() should fail when namespace is not set in context")
	}

	// Should be a namespace required error
	if !strings.Contains(err.Error(), "namespace") {
		t.Errorf("expected namespace error, got: %v", err)
	}
}

func TestShimSocketClose(t *testing.T) {
	// Test shimSocket.Close() with nil values
	s := &shimSocket{}
	s.Close() // Should not panic

	// Test with mock values (we can't easily create real sockets in unit tests)
	s = &shimSocket{
		addr: "/tmp/test.sock",
	}
	s.Close() // Should not panic
}

func TestGroupLabels(t *testing.T) {
	// Verify the group labels are as expected
	expected := []string{
		"io.containerd.runc.v2.group",
		"io.kubernetes.cri.sandbox-id",
	}

	if len(groupLabels) != len(expected) {
		t.Fatalf("expected %d group labels, got %d", len(expected), len(groupLabels))
	}

	for i, label := range expected {
		if groupLabels[i] != label {
			t.Errorf("groupLabels[%d] = %q, want %q", i, groupLabels[i], label)
		}
	}
}

func TestSpecStruct(t *testing.T) {
	// Test spec struct marshaling/unmarshaling
	original := spec{
		Annotations: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var restored spec
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if len(restored.Annotations) != len(original.Annotations) {
		t.Errorf("restored annotations length %d, want %d", len(restored.Annotations), len(original.Annotations))
	}

	for key, value := range original.Annotations {
		if restored.Annotations[key] != value {
			t.Errorf("restored annotation %q = %q, want %q", key, restored.Annotations[key], value)
		}
	}
}

func TestSpecEmptyAnnotations(t *testing.T) {
	// Test spec with nil annotations
	s := spec{Annotations: nil}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var restored spec
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Annotations should be nil or empty map after unmarshaling
	if restored.Annotations != nil && len(restored.Annotations) != 0 {
		t.Errorf("expected nil or empty annotations, got %v", restored.Annotations)
	}
}
