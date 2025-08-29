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

package images

import (
	"context"
	"testing"

	"github.com/containerd/errdefs"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

func TestDiffCompression(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		mediaType   string
		expected    string
		expectedErr bool
	}{
		{
			name:      "OCI layer gzip",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  "gzip",
		},
		{
			name:      "OCI layer zstd",
			mediaType: ocispec.MediaTypeImageLayerZstd,
			expected:  "zstd",
		},
		{
			name:      "Docker layer gzip",
			mediaType: MediaTypeDockerSchema2LayerGzip,
			expected:  "gzip",
		},
		{
			name:      "Docker layer zstd",
			mediaType: MediaTypeDockerSchema2LayerZstd,
			expected:  "zstd",
		},
		{
			name:      "Uncompressed layer",
			mediaType: ocispec.MediaTypeImageLayer,
			expected:  "",
		},
		{
			name:      "Docker uncompressed layer",
			mediaType: MediaTypeDockerSchema2Layer,
			expected:  "unknown",
		},
		{
			name:        "Invalid media type",
			mediaType:   "invalid/type",
			expectedErr: true,
		},
		{
			name:        "Config media type",
			mediaType:   ocispec.MediaTypeImageConfig,
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			compression, err := DiffCompression(ctx, tc.mediaType)

			if tc.expectedErr {
				assert.Error(t, err)
				assert.True(t, errdefs.IsNotImplemented(err))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, compression)
			}
		})
	}
}

func TestIsNonDistributable(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "OCI non-distributable",
			mediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar+gzip",
			expected:  true,
		},
		{
			name:      "Docker foreign layer",
			mediaType: MediaTypeDockerSchema2LayerForeign,
			expected:  true,
		},
		{
			name:      "Docker foreign layer gzip",
			mediaType: MediaTypeDockerSchema2LayerForeignGzip,
			expected:  true,
		},
		{
			name:      "Regular OCI layer",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  false,
		},
		{
			name:      "Regular Docker layer",
			mediaType: MediaTypeDockerSchema2LayerGzip,
			expected:  false,
		},
		{
			name:      "Config type",
			mediaType: ocispec.MediaTypeImageConfig,
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsNonDistributable(tc.mediaType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsDockerType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "Docker manifest",
			mediaType: MediaTypeDockerSchema2Manifest,
			expected:  true,
		},
		{
			name:      "Docker layer",
			mediaType: MediaTypeDockerSchema2LayerGzip,
			expected:  true,
		},
		{
			name:      "Docker config",
			mediaType: MediaTypeDockerSchema2Config,
			expected:  true,
		},
		{
			name:      "OCI manifest",
			mediaType: ocispec.MediaTypeImageManifest,
			expected:  false,
		},
		{
			name:      "OCI layer",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  false,
		},
		{
			name:      "Random type",
			mediaType: "application/json",
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsDockerType(tc.mediaType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsConfigType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "OCI config",
			mediaType: ocispec.MediaTypeImageConfig,
			expected:  true,
		},
		{
			name:      "Docker config",
			mediaType: MediaTypeDockerSchema2Config,
			expected:  true,
		},
		{
			name:      "Containerd checkpoint config",
			mediaType: MediaTypeContainerd1CheckpointConfig,
			expected:  false, // IsConfigType returns false for checkpoint configs
		},
		{
			name:      "OCI manifest",
			mediaType: ocispec.MediaTypeImageManifest,
			expected:  false,
		},
		{
			name:      "Layer",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsConfigType(tc.mediaType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsKnownConfig(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "OCI config",
			mediaType: ocispec.MediaTypeImageConfig,
			expected:  true,
		},
		{
			name:      "Docker config",
			mediaType: MediaTypeDockerSchema2Config,
			expected:  true,
		},
		{
			name:      "Containerd checkpoint",
			mediaType: MediaTypeContainerd1Checkpoint,
			expected:  true,
		},
		{
			name:      "Containerd checkpoint config",
			mediaType: MediaTypeContainerd1CheckpointConfig,
			expected:  true,
		},
		{
			name:      "OCI manifest",
			mediaType: ocispec.MediaTypeImageManifest,
			expected:  false,
		},
		{
			name:      "Layer",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsKnownConfig(tc.mediaType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsAttestationType(t *testing.T) {
	// First check if the function exists
	tests := []struct {
		name      string
		mediaType string
		expected  bool
	}{
		{
			name:      "Regular manifest",
			mediaType: ocispec.MediaTypeImageManifest,
			expected:  false,
		},
		{
			name:      "Regular layer",
			mediaType: ocispec.MediaTypeImageLayerGzip,
			expected:  false,
		},
		{
			name:      "Config",
			mediaType: ocispec.MediaTypeImageConfig,
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsAttestationType(tc.mediaType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestChildGCLabels(t *testing.T) {
	tests := []struct {
		name           string
		desc           ocispec.Descriptor
		expectedLabels []string
	}{
		{
			name: "Manifest descriptor",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    digest.FromString("manifest"),
				Size:      100,
			},
			expectedLabels: []string{"containerd.io/gc.ref.content.m."},
		},
		{
			name: "Index descriptor",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageIndex,
				Digest:    digest.FromString("index"),
				Size:      200,
			},
			expectedLabels: []string{"containerd.io/gc.ref.content."},
		},
		{
			name: "Layer descriptor",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageLayerGzip,
				Digest:    digest.FromString("layer"),
				Size:      300,
			},
			expectedLabels: []string{"containerd.io/gc.ref.content.l."},
		},
		{
			name: "Config descriptor",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageConfig,
				Digest:    digest.FromString("config"),
				Size:      150,
			},
			expectedLabels: []string{"containerd.io/gc.ref.content.config"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			labels := ChildGCLabels(tc.desc)
			assert.Equal(t, tc.expectedLabels, labels)
		})
	}
}

func TestChildGCLabelsFilterLayers(t *testing.T) {
	tests := []struct {
		name           string
		desc           ocispec.Descriptor
		expectedLabels []string
		expectNil      bool
	}{
		{
			name: "Manifest descriptor",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    digest.FromString("manifest"),
				Size:      100,
			},
			expectedLabels: []string{"containerd.io/gc.ref.content.m."},
		},
		{
			name: "Layer descriptor (should be filtered)",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageLayerGzip,
				Digest:    digest.FromString("layer"),
				Size:      300,
			},
			expectNil: true,
		},
		{
			name: "Config descriptor",
			desc: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageConfig,
				Digest:    digest.FromString("config"),
				Size:      150,
			},
			expectedLabels: []string{"containerd.io/gc.ref.content.config"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			labels := ChildGCLabelsFilterLayers(tc.desc)

			if tc.expectNil {
				assert.Nil(t, labels)
			} else {
				assert.Equal(t, tc.expectedLabels, labels)
			}
		})
	}
}

func TestParseMediaTypes(t *testing.T) {
	tests := []struct {
		name             string
		mediaType        string
		expectedBase     string
		expectedSuffixes []string
	}{
		{
			name:             "Simple media type",
			mediaType:        "application/vnd.oci.image.layer.v1.tar",
			expectedBase:     "application/vnd.oci.image.layer.v1.tar",
			expectedSuffixes: []string{},
		},
		{
			name:             "Media type with single suffix",
			mediaType:        "application/vnd.oci.image.layer.v1.tar+gzip",
			expectedBase:     "application/vnd.oci.image.layer.v1.tar",
			expectedSuffixes: []string{"gzip"},
		},
		{
			name:             "Media type with multiple suffixes",
			mediaType:        "application/vnd.oci.image.layer.v1.tar+gzip+encrypted",
			expectedBase:     "application/vnd.oci.image.layer.v1.tar",
			expectedSuffixes: []string{"encrypted", "gzip"}, // Should be sorted
		},
		{
			name:             "Empty media type",
			mediaType:        "",
			expectedBase:     "",
			expectedSuffixes: []string{},
		},
		{
			name:             "Complex suffixes",
			mediaType:        "application/test+zstd+encrypted+signed",
			expectedBase:     "application/test",
			expectedSuffixes: []string{"encrypted", "signed", "zstd"}, // Should be sorted
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base, suffixes := parseMediaTypes(tc.mediaType)

			assert.Equal(t, tc.expectedBase, base)
			assert.Equal(t, tc.expectedSuffixes, suffixes)
		})
	}
}
