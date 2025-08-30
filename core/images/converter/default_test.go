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

package converter

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/platforms"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestConvertDockerMediaTypeToOCI(t *testing.T) {
	tests := []struct {
		name       string
		dockerType string
		expected   string
	}{
		{
			name:       "manifest list",
			dockerType: images.MediaTypeDockerSchema2ManifestList,
			expected:   ocispec.MediaTypeImageIndex,
		},
		{
			name:       "manifest",
			dockerType: images.MediaTypeDockerSchema2Manifest,
			expected:   ocispec.MediaTypeImageManifest,
		},
		{
			name:       "layer gzip",
			dockerType: images.MediaTypeDockerSchema2LayerGzip,
			expected:   ocispec.MediaTypeImageLayerGzip,
		},
		{
			name:       "foreign layer gzip",
			dockerType: images.MediaTypeDockerSchema2LayerForeignGzip,
			expected:   ocispec.MediaTypeImageLayerNonDistributableGzip,
		},
		{
			name:       "layer",
			dockerType: images.MediaTypeDockerSchema2Layer,
			expected:   ocispec.MediaTypeImageLayer,
		},
		{
			name:       "foreign layer",
			dockerType: images.MediaTypeDockerSchema2LayerForeign,
			expected:   ocispec.MediaTypeImageLayerNonDistributable,
		},
		{
			name:       "config",
			dockerType: images.MediaTypeDockerSchema2Config,
			expected:   ocispec.MediaTypeImageConfig,
		},
		{
			name:       "unknown type",
			dockerType: "unknown/type",
			expected:   "unknown/type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertDockerMediaTypeToOCI(tt.dockerType)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClearGCLabels(t *testing.T) {
	dgst := digest.FromString("test")
	labels := map[string]string{
		"containerd.io/gc.ref.content.l.0":    dgst.String(),
		"containerd.io/gc.ref.content.m.1":    dgst.String(),
		"containerd.io/gc.ref.content.config": dgst.String(),
		"other.label":                         "value",
		"containerd.io/gc.ref.content.x":      "other-digest",
	}

	ClearGCLabels(labels, dgst)

	expectedLabels := map[string]string{
		"other.label":                    "value",
		"containerd.io/gc.ref.content.x": "other-digest",
	}

	if len(labels) != len(expectedLabels) {
		t.Errorf("expected %d labels, got %d", len(expectedLabels), len(labels))
	}

	for k, v := range expectedLabels {
		if labels[k] != v {
			t.Errorf("expected label %s=%s, got %s", k, v, labels[k])
		}
	}
}

func TestDefaultIndexConvertFunc(t *testing.T) {
	layerConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	convertFunc := DefaultIndexConvertFunc(layerConvertFunc, true, platforms.All)
	if convertFunc == nil {
		t.Fatal("expected convert function, got nil")
	}
}

func TestIndexConvertFuncWithHook(t *testing.T) {
	layerConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	postHook := func(ctx context.Context, cs content.Store, orgDesc ocispec.Descriptor, newDesc *ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return newDesc, nil
	}

	hooks := ConvertHooks{
		PostConvertHook: postHook,
	}

	convertFunc := IndexConvertFuncWithHook(layerConvertFunc, true, platforms.All, hooks)
	if convertFunc == nil {
		t.Fatal("expected convert function, got nil")
	}
}

func TestClearDockerV1DummyID(t *testing.T) {
	// Test case 1: Config with Image field
	imageJSON := []byte(`"dummy-image-id"`)
	cmdJSON := []byte(`["/bin/sh"]`)
	configField := map[string]*json.RawMessage{
		"Image": (*json.RawMessage)(&imageJSON),
		"Cmd":   (*json.RawMessage)(&cmdJSON),
	}
	configBytes, _ := json.Marshal(configField)

	cfg := DualConfig{
		"config": (*json.RawMessage)(&configBytes),
	}

	modified, err := clearDockerV1DummyID(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Fatal("expected config to be modified")
	}

	// Verify Image field was removed
	var updatedConfigField map[string]*json.RawMessage
	err = json.Unmarshal(*cfg["config"], &updatedConfigField)
	if err != nil {
		t.Fatalf("failed to unmarshal updated config: %v", err)
	}
	if _, exists := updatedConfigField["Image"]; exists {
		t.Fatal("expected Image field to be removed")
	}
	if _, exists := updatedConfigField["Cmd"]; !exists {
		t.Fatal("expected Cmd field to remain")
	}
}

func TestClearDockerV1DummyID_ContainerConfig(t *testing.T) {
	// Test case 2: container_config with Image field
	imageJSON := []byte(`"dummy-image-id"`)
	envJSON := []byte(`["PATH=/usr/bin"]`)
	containerConfigField := map[string]*json.RawMessage{
		"Image": (*json.RawMessage)(&imageJSON),
		"Env":   (*json.RawMessage)(&envJSON),
	}
	containerConfigBytes, _ := json.Marshal(containerConfigField)

	cfg := DualConfig{
		"container_config": (*json.RawMessage)(&containerConfigBytes),
	}

	modified, err := clearDockerV1DummyID(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Fatal("expected container_config to be modified")
	}

	// Verify Image field was removed
	var updatedContainerConfigField map[string]*json.RawMessage
	err = json.Unmarshal(*cfg["container_config"], &updatedContainerConfigField)
	if err != nil {
		t.Fatalf("failed to unmarshal updated container_config: %v", err)
	}
	if _, exists := updatedContainerConfigField["Image"]; exists {
		t.Fatal("expected Image field to be removed")
	}
	if _, exists := updatedContainerConfigField["Env"]; !exists {
		t.Fatal("expected Env field to remain")
	}
}

func TestClearDockerV1DummyID_NoModification(t *testing.T) {
	// Test case 3: No config fields to modify
	rootfsJSON := []byte(`{"type": "layers"}`)
	cfg := DualConfig{
		"rootfs": (*json.RawMessage)(&rootfsJSON),
	}

	modified, err := clearDockerV1DummyID(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Fatal("expected no modification")
	}
}

func TestClearDockerV1DummyID_BothFields(t *testing.T) {
	// Test case 4: Both config and container_config with Image fields
	imageJSON1 := []byte(`"dummy-image-id"`)
	cmdJSON := []byte(`["/bin/sh"]`)
	configField := map[string]*json.RawMessage{
		"Image": (*json.RawMessage)(&imageJSON1),
		"Cmd":   (*json.RawMessage)(&cmdJSON),
	}
	configBytes, _ := json.Marshal(configField)

	imageJSON2 := []byte(`"dummy-image-id"`)
	envJSON := []byte(`["PATH=/usr/bin"]`)
	containerConfigField := map[string]*json.RawMessage{
		"Image": (*json.RawMessage)(&imageJSON2),
		"Env":   (*json.RawMessage)(&envJSON),
	}
	containerConfigBytes, _ := json.Marshal(containerConfigField)

	cfg := DualConfig{
		"config":           (*json.RawMessage)(&configBytes),
		"container_config": (*json.RawMessage)(&containerConfigBytes),
	}

	modified, err := clearDockerV1DummyID(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Fatal("expected config to be modified")
	}

	// Verify both Image fields were removed
	var updatedConfigField map[string]*json.RawMessage
	err = json.Unmarshal(*cfg["config"], &updatedConfigField)
	if err != nil {
		t.Fatalf("failed to unmarshal updated config: %v", err)
	}
	if _, exists := updatedConfigField["Image"]; exists {
		t.Fatal("expected config Image field to be removed")
	}

	var updatedContainerConfigField map[string]*json.RawMessage
	err = json.Unmarshal(*cfg["container_config"], &updatedContainerConfigField)
	if err != nil {
		t.Fatalf("failed to unmarshal updated container_config: %v", err)
	}
	if _, exists := updatedContainerConfigField["Image"]; exists {
		t.Fatal("expected container_config Image field to be removed")
	}
}

func TestClearDockerV1DummyID_InvalidJSON(t *testing.T) {
	// Test case 5: Invalid JSON in config field
	invalidJSON := []byte(`{invalid json}`)

	cfg := DualConfig{
		"config": (*json.RawMessage)(&invalidJSON),
	}

	_, err := clearDockerV1DummyID(cfg)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCopyDesc(t *testing.T) {
	original := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:abc123",
		Size:      123,
		Annotations: map[string]string{
			"test": "value",
		},
	}

	copied := copyDesc(original)

	if copied == &original {
		t.Fatal("expected different pointer")
	}

	if copied.MediaType != original.MediaType {
		t.Errorf("expected MediaType %s, got %s", original.MediaType, copied.MediaType)
	}
	if copied.Digest != original.Digest {
		t.Errorf("expected Digest %s, got %s", original.Digest, copied.Digest)
	}
	if copied.Size != original.Size {
		t.Errorf("expected Size %d, got %d", original.Size, copied.Size)
	}

	// Modify original to ensure deep copy
	original.MediaType = "changed"
	if copied.MediaType == "changed" {
		t.Fatal("expected copied descriptor to be independent")
	}
}

func TestDualConfig_JSONRoundTrip(t *testing.T) {
	// Test that DualConfig can marshal/unmarshal JSON properly
	originalData := map[string]interface{}{
		"architecture": "amd64",
		"config": map[string]interface{}{
			"Env": []string{"PATH=/usr/local/sbin:/usr/local/bin"},
			"Cmd": []string{"/bin/sh"},
		},
		"rootfs": map[string]interface{}{
			"type": "layers",
			"diff_ids": []string{
				"sha256:abc123",
				"sha256:def456",
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalData)
	if err != nil {
		t.Fatalf("failed to marshal original data: %v", err)
	}

	// Unmarshal into DualConfig
	var cfg DualConfig
	err = json.Unmarshal(jsonData, &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal into DualConfig: %v", err)
	}

	// Verify all fields are present
	if _, exists := cfg["architecture"]; !exists {
		t.Fatal("expected architecture field")
	}
	if _, exists := cfg["config"]; !exists {
		t.Fatal("expected config field")
	}
	if _, exists := cfg["rootfs"]; !exists {
		t.Fatal("expected rootfs field")
	}

	// Marshal back to JSON and verify roundtrip
	remarkshaled, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal DualConfig: %v", err)
	}

	var roundtrip map[string]interface{}
	err = json.Unmarshal(remarkshaled, &roundtrip)
	if err != nil {
		t.Fatalf("failed to unmarshal roundtrip: %v", err)
	}

	// Basic validation that structure is preserved
	if roundtrip["architecture"] != originalData["architecture"] {
		t.Errorf("architecture field mismatch")
	}
}
