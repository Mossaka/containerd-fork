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

package diff

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/containerd/typeurl/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestWithCompressor(t *testing.T) {
	compressorFunc := func(dest io.Writer, mediaType string) (io.WriteCloser, error) {
		return &mockWriteCloser{dest}, nil
	}

	config := &Config{}
	opt := WithCompressor(compressorFunc)

	if err := opt(config); err != nil {
		t.Fatalf("WithCompressor failed: %v", err)
	}

	if config.Compressor == nil {
		t.Fatal("Compressor function was not set")
	}

	// Test that the compressor function works
	mockWriter := &mockWriter{}
	wc, err := config.Compressor(mockWriter, "application/vnd.oci.image.layer.v1.tar+gzip")
	if err != nil {
		t.Fatalf("Compressor function failed: %v", err)
	}
	if wc == nil {
		t.Fatal("Compressor function returned nil WriteCloser")
	}
}

func TestWithMediaType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
	}{
		{
			name:      "gzip layer",
			mediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		},
		{
			name:      "zstd layer",
			mediaType: "application/vnd.oci.image.layer.v1.tar+zstd",
		},
		{
			name:      "uncompressed layer",
			mediaType: "application/vnd.oci.image.layer.v1.tar",
		},
		{
			name:      "custom media type",
			mediaType: "application/custom",
		},
		{
			name:      "empty media type",
			mediaType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			opt := WithMediaType(tt.mediaType)

			if err := opt(config); err != nil {
				t.Fatalf("WithMediaType failed: %v", err)
			}

			if config.MediaType != tt.mediaType {
				t.Fatalf("Expected MediaType %q, got %q", tt.mediaType, config.MediaType)
			}
		})
	}
}

func TestWithReference(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{
			name: "standard reference",
			ref:  "sha256:1234567890abcdef",
		},
		{
			name: "custom reference",
			ref:  "my-custom-ref",
		},
		{
			name: "empty reference",
			ref:  "",
		},
		{
			name: "long reference",
			ref:  "very-long-reference-name-with-many-characters-to-test-limits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			opt := WithReference(tt.ref)

			if err := opt(config); err != nil {
				t.Fatalf("WithReference failed: %v", err)
			}

			if config.Reference != tt.ref {
				t.Fatalf("Expected Reference %q, got %q", tt.ref, config.Reference)
			}
		})
	}
}

func TestWithLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "single label",
			labels: map[string]string{
				"key": "value",
			},
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"app":     "containerd",
				"version": "2.0",
				"env":     "test",
			},
		},
		{
			name:   "nil labels",
			labels: nil,
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
		},
		{
			name: "labels with special characters",
			labels: map[string]string{
				"io.containerd.test": "special-value",
				"special/key":        "value with spaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			opt := WithLabels(tt.labels)

			if err := opt(config); err != nil {
				t.Fatalf("WithLabels failed: %v", err)
			}

			if len(config.Labels) != len(tt.labels) {
				t.Fatalf("Expected %d labels, got %d", len(tt.labels), len(config.Labels))
			}

			for key, expectedValue := range tt.labels {
				actualValue, exists := config.Labels[key]
				if !exists {
					t.Fatalf("Expected label %q not found", key)
				}
				if actualValue != expectedValue {
					t.Fatalf("Expected label %q to have value %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestWithSourceDateEpoch(t *testing.T) {
	tests := []struct {
		name string
		time *time.Time
	}{
		{
			name: "specific timestamp",
			time: func() *time.Time {
				tm := time.Unix(1609459200, 0) // 2021-01-01 00:00:00 UTC
				return &tm
			}(),
		},
		{
			name: "nil timestamp",
			time: nil,
		},
		{
			name: "zero timestamp",
			time: func() *time.Time {
				tm := time.Unix(0, 0)
				return &tm
			}(),
		},
		{
			name: "recent timestamp",
			time: func() *time.Time {
				tm := time.Now()
				return &tm
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			opt := WithSourceDateEpoch(tt.time)

			if err := opt(config); err != nil {
				t.Fatalf("WithSourceDateEpoch failed: %v", err)
			}

			if tt.time == nil {
				if config.SourceDateEpoch != nil {
					t.Fatal("Expected SourceDateEpoch to be nil")
				}
			} else {
				if config.SourceDateEpoch == nil {
					t.Fatal("Expected SourceDateEpoch to be set")
				}
				if !config.SourceDateEpoch.Equal(*tt.time) {
					t.Fatalf("Expected SourceDateEpoch %v, got %v", *tt.time, *config.SourceDateEpoch)
				}
			}
		})
	}
}

func TestWithPayloads(t *testing.T) {
	// Create mock payloads for testing
	mockPayload1 := &mockAny{
		typeURL: "type.googleapis.com/test.Payload",
		value:   []byte("test-data"),
	}
	mockPayload2 := &mockAny{
		typeURL: "type.googleapis.com/test.Payload1",
		value:   []byte("data1"),
	}
	mockPayload3 := &mockAny{
		typeURL: "type.googleapis.com/test.Payload2",
		value:   []byte("data2"),
	}

	tests := []struct {
		name     string
		payloads map[string]typeurl.Any
	}{
		{
			name: "single payload",
			payloads: map[string]typeurl.Any{
				"processor1": mockPayload1,
			},
		},
		{
			name: "multiple payloads",
			payloads: map[string]typeurl.Any{
				"processor1": mockPayload2,
				"processor2": mockPayload3,
			},
		},
		{
			name:     "nil payloads",
			payloads: nil,
		},
		{
			name:     "empty payloads",
			payloads: map[string]typeurl.Any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ApplyConfig{}
			ctx := context.Background()
			desc := ocispec.Descriptor{MediaType: "application/test"}

			opt := WithPayloads(tt.payloads)

			if err := opt(ctx, desc, config); err != nil {
				t.Fatalf("WithPayloads failed: %v", err)
			}

			if len(config.ProcessorPayloads) != len(tt.payloads) {
				t.Fatalf("Expected %d payloads, got %d", len(tt.payloads), len(config.ProcessorPayloads))
			}

			for key, expectedPayload := range tt.payloads {
				actualPayload, exists := config.ProcessorPayloads[key]
				if !exists {
					t.Fatalf("Expected payload %q not found", key)
				}
				if actualPayload.GetTypeUrl() != expectedPayload.GetTypeUrl() {
					t.Fatalf("Expected TypeURL %q, got %q", expectedPayload.GetTypeUrl(), actualPayload.GetTypeUrl())
				}
				if string(actualPayload.GetValue()) != string(expectedPayload.GetValue()) {
					t.Fatalf("Expected Value %q, got %q", expectedPayload.GetValue(), actualPayload.GetValue())
				}
			}
		})
	}
}

func TestWithSyncFs(t *testing.T) {
	tests := []struct {
		name string
		sync bool
	}{
		{
			name: "sync enabled",
			sync: true,
		},
		{
			name: "sync disabled",
			sync: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ApplyConfig{}
			ctx := context.Background()
			desc := ocispec.Descriptor{MediaType: "application/test"}

			opt := WithSyncFs(tt.sync)

			if err := opt(ctx, desc, config); err != nil {
				t.Fatalf("WithSyncFs failed: %v", err)
			}

			if config.SyncFs != tt.sync {
				t.Fatalf("Expected SyncFs %v, got %v", tt.sync, config.SyncFs)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	config := &Config{}

	// Test default values
	if config.MediaType != "" {
		t.Fatalf("Expected empty MediaType, got %q", config.MediaType)
	}
	if config.Reference != "" {
		t.Fatalf("Expected empty Reference, got %q", config.Reference)
	}
	if config.Labels != nil {
		t.Fatalf("Expected nil Labels, got %v", config.Labels)
	}
	if config.Compressor != nil {
		t.Fatal("Expected nil Compressor")
	}
	if config.SourceDateEpoch != nil {
		t.Fatal("Expected nil SourceDateEpoch")
	}
}

func TestApplyConfigStruct(t *testing.T) {
	config := &ApplyConfig{}

	// Test default values
	if config.ProcessorPayloads != nil {
		t.Fatalf("Expected nil ProcessorPayloads, got %v", config.ProcessorPayloads)
	}
	if config.SyncFs {
		t.Fatal("Expected SyncFs to be false")
	}
}

func TestMultipleOptions(t *testing.T) {
	config := &Config{}
	labels := map[string]string{"app": "test"}
	timestamp := time.Unix(1234567890, 0)

	opts := []Opt{
		WithMediaType("application/vnd.oci.image.layer.v1.tar+gzip"),
		WithReference("test-ref"),
		WithLabels(labels),
		WithSourceDateEpoch(&timestamp),
	}

	for _, opt := range opts {
		if err := opt(config); err != nil {
			t.Fatalf("Option failed: %v", err)
		}
	}

	// Verify all options were applied
	if config.MediaType != "application/vnd.oci.image.layer.v1.tar+gzip" {
		t.Fatalf("MediaType not set correctly")
	}
	if config.Reference != "test-ref" {
		t.Fatalf("Reference not set correctly")
	}
	if len(config.Labels) != 1 || config.Labels["app"] != "test" {
		t.Fatalf("Labels not set correctly")
	}
	if config.SourceDateEpoch == nil || !config.SourceDateEpoch.Equal(timestamp) {
		t.Fatalf("SourceDateEpoch not set correctly")
	}
}

func TestMultipleApplyOptions(t *testing.T) {
	config := &ApplyConfig{}
	ctx := context.Background()
	desc := ocispec.Descriptor{MediaType: "application/test"}

	mockPayload := &mockAny{
		typeURL: "type.test",
		value:   []byte("data"),
	}

	payloads := map[string]typeurl.Any{
		"test": mockPayload,
	}

	opts := []ApplyOpt{
		WithPayloads(payloads),
		WithSyncFs(true),
	}

	for _, opt := range opts {
		if err := opt(ctx, desc, config); err != nil {
			t.Fatalf("ApplyOpt failed: %v", err)
		}
	}

	// Verify all options were applied
	if len(config.ProcessorPayloads) != 1 {
		t.Fatalf("ProcessorPayloads not set correctly")
	}
	if !config.SyncFs {
		t.Fatalf("SyncFs not set correctly")
	}
}

// Mock implementations for testing

type mockWriter struct {
	data []byte
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

type mockWriteCloser struct {
	io.Writer
}

func (m *mockWriteCloser) Close() error {
	return nil
}

// Mock typeurl.Any implementation
type mockAny struct {
	typeURL string
	value   []byte
}

func (m *mockAny) GetTypeUrl() string {
	return m.typeURL
}

func (m *mockAny) GetValue() []byte {
	return m.value
}
