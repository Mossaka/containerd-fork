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

package uncompress

import (
	"testing"

	"github.com/containerd/containerd/v2/core/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestIsUncompressedType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "docker layer",
			mediaType: images.MediaTypeDockerSchema2Layer,
			expected:  true,
		},
		{
			name:      "docker foreign layer",
			mediaType: images.MediaTypeDockerSchema2LayerForeign,
			expected:  true,
		},
		{
			name:      "oci layer",
			mediaType: ocispec.MediaTypeImageLayer,
			expected:  true,
		},
		{
			name:      "oci non-distributable layer",
			mediaType: ocispec.MediaTypeImageLayerNonDistributable,
			expected:  true,
		},
		{
			name:      "docker compressed layer",
			mediaType: images.MediaTypeDockerSchema2LayerGzip,
			expected:  false,
		},
		{
			name:      "oci compressed layer",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  false,
		},
		{
			name:      "oci zstd layer",
			mediaType: ocispec.MediaTypeImageLayerZstd,
			expected:  false,
		},
		{
			name:      "unknown type",
			mediaType: "unknown/type",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUncompressedType(tt.mediaType)
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestConvertMediaType(t *testing.T) {
	tests := []struct {
		name      string
		inputType string
		expected  string
	}{
		{
			name:      "docker gzip layer",
			inputType: images.MediaTypeDockerSchema2LayerGzip,
			expected:  images.MediaTypeDockerSchema2Layer,
		},
		{
			name:      "docker foreign gzip layer",
			inputType: images.MediaTypeDockerSchema2LayerForeignGzip,
			expected:  images.MediaTypeDockerSchema2LayerForeign,
		},
		{
			name:      "oci gzip layer",
			inputType: ocispec.MediaTypeImageLayerGzip,
			expected:  ocispec.MediaTypeImageLayer,
		},
		{
			name:      "oci zstd layer",
			inputType: ocispec.MediaTypeImageLayerZstd,
			expected:  ocispec.MediaTypeImageLayer,
		},
		{
			name:      "oci non-distributable gzip layer",
			inputType: ocispec.MediaTypeImageLayerNonDistributableGzip,
			expected:  ocispec.MediaTypeImageLayerNonDistributable,
		},
		{
			name:      "oci non-distributable zstd layer",
			inputType: ocispec.MediaTypeImageLayerNonDistributableZstd,
			expected:  ocispec.MediaTypeImageLayerNonDistributable,
		},
		{
			name:      "already uncompressed docker layer",
			inputType: images.MediaTypeDockerSchema2Layer,
			expected:  images.MediaTypeDockerSchema2Layer,
		},
		{
			name:      "already uncompressed oci layer",
			inputType: ocispec.MediaTypeImageLayer,
			expected:  ocispec.MediaTypeImageLayer,
		},
		{
			name:      "unknown type",
			inputType: "unknown/type",
			expected:  "unknown/type",
		},
		{
			name:      "manifest type",
			inputType: ocispec.MediaTypeImageManifest,
			expected:  ocispec.MediaTypeImageManifest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMediaType(tt.inputType)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestLayerConvertFunc_NonLayerType tests that LayerConvertFunc returns nil
// for non-layer media types without error
func TestLayerConvertFunc_NonLayerType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
	}{
		{
			name:      "manifest",
			mediaType: ocispec.MediaTypeImageManifest,
		},
		{
			name:      "index",
			mediaType: ocispec.MediaTypeImageIndex,
		},
		{
			name:      "config",
			mediaType: ocispec.MediaTypeImageConfig,
		},
		{
			name:      "unknown",
			mediaType: "unknown/type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := ocispec.Descriptor{
				MediaType: tt.mediaType,
				Digest:    "sha256:abc123",
				Size:      123,
			}

			result, err := LayerConvertFunc(nil, nil, desc)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
		})
	}
}

// TestLayerConvertFunc_AlreadyUncompressed tests that LayerConvertFunc returns nil
// for already uncompressed layer types without error
func TestLayerConvertFunc_AlreadyUncompressed(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
	}{
		{
			name:      "docker layer",
			mediaType: images.MediaTypeDockerSchema2Layer,
		},
		{
			name:      "docker foreign layer",
			mediaType: images.MediaTypeDockerSchema2LayerForeign,
		},
		{
			name:      "oci layer",
			mediaType: ocispec.MediaTypeImageLayer,
		},
		{
			name:      "oci non-distributable layer",
			mediaType: ocispec.MediaTypeImageLayerNonDistributable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := ocispec.Descriptor{
				MediaType: tt.mediaType,
				Digest:    "sha256:abc123",
				Size:      123,
			}

			result, err := LayerConvertFunc(nil, nil, desc)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
		})
	}
}
