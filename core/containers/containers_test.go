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

package containers

import (
	"testing"
	"time"

	"github.com/containerd/typeurl/v2"
)

func init() {
	// Register mock types for testing
	typeurl.Register(&MockSpec{}, "containerd.test.MockSpec")
	typeurl.Register(&MockOptions{}, "containerd.test.MockOptions")
	typeurl.Register(&MockExtension{}, "containerd.test.MockExtension")
}

func TestContainer(t *testing.T) {
	now := time.Now()

	spec, err := typeurl.MarshalAny(&MockSpec{Name: "test-spec"})
	if err != nil {
		t.Fatal("failed to marshal spec:", err)
	}

	opts, err := typeurl.MarshalAny(&MockOptions{Debug: true})
	if err != nil {
		t.Fatal("failed to marshal options:", err)
	}

	ext, err := typeurl.MarshalAny(&MockExtension{Value: "test-ext"})
	if err != nil {
		t.Fatal("failed to marshal extension:", err)
	}

	container := Container{
		ID:          "test-container",
		Labels:      map[string]string{"env": "test", "version": "1.0"},
		Image:       "test-image:latest",
		Runtime:     RuntimeInfo{Name: "runc", Options: opts},
		Spec:        spec,
		SnapshotKey: "snapshot-123",
		Snapshotter: "overlayfs",
		CreatedAt:   now,
		UpdatedAt:   now,
		Extensions:  map[string]typeurl.Any{"test-ext": ext},
		SandboxID:   "sandbox-456",
	}

	// Test Container struct fields
	if container.ID != "test-container" {
		t.Errorf("expected container ID 'test-container', got %s", container.ID)
	}

	if container.Image != "test-image:latest" {
		t.Errorf("expected container image 'test-image:latest', got %s", container.Image)
	}

	if len(container.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(container.Labels))
	}

	if container.Labels["env"] != "test" {
		t.Errorf("expected label env=test, got %s", container.Labels["env"])
	}

	if container.Runtime.Name != "runc" {
		t.Errorf("expected runtime name 'runc', got %s", container.Runtime.Name)
	}

	if container.SnapshotKey != "snapshot-123" {
		t.Errorf("expected snapshot key 'snapshot-123', got %s", container.SnapshotKey)
	}

	if container.Snapshotter != "overlayfs" {
		t.Errorf("expected snapshotter 'overlayfs', got %s", container.Snapshotter)
	}

	if container.SandboxID != "sandbox-456" {
		t.Errorf("expected sandbox ID 'sandbox-456', got %s", container.SandboxID)
	}

	if container.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if container.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	// Test Extensions
	if len(container.Extensions) != 1 {
		t.Errorf("expected 1 extension, got %d", len(container.Extensions))
	}

	if _, exists := container.Extensions["test-ext"]; !exists {
		t.Error("expected test-ext extension to exist")
	}

	// Test Spec
	var unmarshaledSpec MockSpec
	err = typeurl.UnmarshalTo(container.Spec, &unmarshaledSpec)
	if err != nil {
		t.Fatal("failed to unmarshal spec:", err)
	}

	if unmarshaledSpec.Name != "test-spec" {
		t.Errorf("expected spec name 'test-spec', got %s", unmarshaledSpec.Name)
	}
}

func TestRuntimeInfo(t *testing.T) {
	opts, err := typeurl.MarshalAny(&MockOptions{Debug: true, LogLevel: "info"})
	if err != nil {
		t.Fatal("failed to marshal options:", err)
	}

	runtime := RuntimeInfo{
		Name:    "runc",
		Options: opts,
	}

	if runtime.Name != "runc" {
		t.Errorf("expected runtime name 'runc', got %s", runtime.Name)
	}

	// Test unmarshaling options
	var unmarshaledOpts MockOptions
	err = typeurl.UnmarshalTo(runtime.Options, &unmarshaledOpts)
	if err != nil {
		t.Fatal("failed to unmarshal options:", err)
	}

	if !unmarshaledOpts.Debug {
		t.Error("expected debug option to be true")
	}

	if unmarshaledOpts.LogLevel != "info" {
		t.Errorf("expected log level 'info', got %s", unmarshaledOpts.LogLevel)
	}
}

func TestContainerWithMinimalFields(t *testing.T) {
	// Test container with only required fields
	container := Container{
		ID: "minimal-container",
		Runtime: RuntimeInfo{
			Name: "runc",
		},
	}

	if container.ID != "minimal-container" {
		t.Errorf("expected container ID 'minimal-container', got %s", container.ID)
	}

	if container.Runtime.Name != "runc" {
		t.Errorf("expected runtime name 'runc', got %s", container.Runtime.Name)
	}

	// Test that optional fields have zero values
	if container.Image != "" {
		t.Errorf("expected empty image, got %s", container.Image)
	}

	if container.SnapshotKey != "" {
		t.Errorf("expected empty snapshot key, got %s", container.SnapshotKey)
	}

	if container.Snapshotter != "" {
		t.Errorf("expected empty snapshotter, got %s", container.Snapshotter)
	}

	if container.SandboxID != "" {
		t.Errorf("expected empty sandbox ID, got %s", container.SandboxID)
	}

	if container.Labels != nil {
		t.Errorf("expected nil labels, got %v", container.Labels)
	}

	if container.Extensions != nil {
		t.Errorf("expected nil extensions, got %v", container.Extensions)
	}

	if !container.CreatedAt.IsZero() {
		t.Error("expected zero CreatedAt time")
	}

	if !container.UpdatedAt.IsZero() {
		t.Error("expected zero UpdatedAt time")
	}
}

func TestContainerWithEmptyCollections(t *testing.T) {
	// Test container with empty but non-nil collections
	container := Container{
		ID:         "empty-collections-container",
		Labels:     map[string]string{},
		Extensions: map[string]typeurl.Any{},
		Runtime:    RuntimeInfo{Name: "runc"},
	}

	if len(container.Labels) != 0 {
		t.Errorf("expected 0 labels, got %d", len(container.Labels))
	}

	if len(container.Extensions) != 0 {
		t.Errorf("expected 0 extensions, got %d", len(container.Extensions))
	}

	// Test that we can add to empty collections
	container.Labels["new"] = "label"
	if container.Labels["new"] != "label" {
		t.Error("failed to add label to empty collection")
	}
}

// Mock types for testing
type MockSpec struct {
	Name string `json:"name"`
}

type MockOptions struct {
	Debug    bool   `json:"debug"`
	LogLevel string `json:"log_level"`
}

type MockExtension struct {
	Value string `json:"value"`
}
