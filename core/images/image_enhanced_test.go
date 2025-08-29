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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/errdefs"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockContentProvider provides a mock implementation of content.Provider for testing
type mockContentProvider struct {
	blobs map[digest.Digest][]byte
}

func newMockContentProvider() *mockContentProvider {
	return &mockContentProvider{
		blobs: make(map[digest.Digest][]byte),
	}
}

func (m *mockContentProvider) addBlob(mediaType string, data []byte) ocispec.Descriptor {
	d := digest.FromBytes(data)
	m.blobs[d] = data
	return ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    d,
		Size:      int64(len(data)),
	}
}

func (m *mockContentProvider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	data, exists := m.blobs[desc.Digest]
	if !exists {
		return nil, fmt.Errorf("blob not found: %w", errdefs.ErrNotFound)
	}
	return &mockReaderAt{data: data}, nil
}

type mockReaderAt struct {
	data []byte
}

func (m *mockReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, m.data[off:])
	return n, nil
}

func (m *mockReaderAt) Close() error {
	return nil
}

func (m *mockReaderAt) Size() int64 {
	return int64(len(m.data))
}

func TestDeleteOptions(t *testing.T) {
	t.Run("SynchronousDelete", func(t *testing.T) {
		ctx := context.Background()
		opts := &DeleteOptions{}

		deleteOpt := SynchronousDelete()
		err := deleteOpt(ctx, opts)

		assert.NoError(t, err)
		assert.True(t, opts.Synchronous)
	})

	t.Run("DeleteTarget", func(t *testing.T) {
		ctx := context.Background()
		opts := &DeleteOptions{}

		targetDesc := &ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    digest.FromString("test"),
			Size:      100,
		}

		deleteOpt := DeleteTarget(targetDesc)
		err := deleteOpt(ctx, opts)

		assert.NoError(t, err)
		assert.Equal(t, targetDesc, opts.Target)
	})

	t.Run("CombinedDeleteOptions", func(t *testing.T) {
		ctx := context.Background()
		opts := &DeleteOptions{}

		targetDesc := &ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    digest.FromString("test"),
			Size:      100,
		}

		// Apply multiple delete options
		err := SynchronousDelete()(ctx, opts)
		assert.NoError(t, err)

		err = DeleteTarget(targetDesc)(ctx, opts)
		assert.NoError(t, err)

		assert.True(t, opts.Synchronous)
		assert.Equal(t, targetDesc, opts.Target)
	})
}

func TestImageStruct(t *testing.T) {
	t.Run("BasicImageCreation", func(t *testing.T) {
		now := time.Now()
		img := Image{
			Name:   "test/image:latest",
			Labels: map[string]string{"version": "1.0"},
			Target: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    digest.FromString("test"),
				Size:      100,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, "test/image:latest", img.Name)
		assert.Equal(t, "1.0", img.Labels["version"])
		assert.Equal(t, ocispec.MediaTypeImageManifest, img.Target.MediaType)
		assert.Equal(t, now, img.CreatedAt)
		assert.Equal(t, now, img.UpdatedAt)
	})
}

func TestConfigPlatformFunction(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		// Create a mock image config
		config := ocispec.Image{
			Platform: ocispec.Platform{
				Architecture: "amd64",
				OS:           "linux",
			},
		}
		configData, err := json.Marshal(config)
		require.NoError(t, err)

		configDesc := provider.addBlob(ocispec.MediaTypeImageConfig, configData)

		platform, err := ConfigPlatform(ctx, provider, configDesc)

		assert.NoError(t, err)
		assert.Equal(t, "amd64", platform.Architecture)
		assert.Equal(t, "linux", platform.OS)
	})

	t.Run("ConfigNotFound", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		configDesc := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    digest.FromString("missing"),
			Size:      100,
		}

		_, err := ConfigPlatform(ctx, provider, configDesc)

		assert.Error(t, err)
		assert.True(t, errdefs.IsNotFound(err))
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		configDesc := provider.addBlob(ocispec.MediaTypeImageConfig, []byte("invalid json"))

		_, err := ConfigPlatform(ctx, provider, configDesc)

		assert.Error(t, err)
	})
}

func TestRootFSFunction(t *testing.T) {
	t.Run("ValidRootFS", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		// Create a mock image config with RootFS
		diffIDs := []digest.Digest{
			digest.FromString("layer1"),
			digest.FromString("layer2"),
		}
		config := ocispec.Image{
			RootFS: ocispec.RootFS{
				Type:    "layers",
				DiffIDs: diffIDs,
			},
		}
		configData, err := json.Marshal(config)
		require.NoError(t, err)

		configDesc := provider.addBlob(ocispec.MediaTypeImageConfig, configData)

		rootFS, err := RootFS(ctx, provider, configDesc)

		assert.NoError(t, err)
		assert.Equal(t, diffIDs, rootFS)
	})

	t.Run("ConfigNotFound", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		configDesc := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    digest.FromString("missing"),
			Size:      100,
		}

		_, err := RootFS(ctx, provider, configDesc)

		assert.Error(t, err)
		assert.True(t, errdefs.IsNotFound(err))
	})
}

func TestChildrenFunction(t *testing.T) {
	t.Run("ManifestChildren", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		// Create a manifest with config and layers
		manifest := ocispec.Manifest{
			Config: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageConfig,
				Digest:    digest.FromString("config"),
				Size:      100,
			},
			Layers: []ocispec.Descriptor{
				{
					MediaType: ocispec.MediaTypeImageLayerGzip,
					Digest:    digest.FromString("layer1"),
					Size:      200,
				},
				{
					MediaType: ocispec.MediaTypeImageLayerGzip,
					Digest:    digest.FromString("layer2"),
					Size:      300,
				},
			},
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)

		manifestDesc := provider.addBlob(ocispec.MediaTypeImageManifest, manifestData)

		children, err := Children(ctx, provider, manifestDesc)

		assert.NoError(t, err)
		assert.Len(t, children, 3) // config + 2 layers
		assert.Equal(t, manifest.Config, children[0])
		assert.Equal(t, manifest.Layers[0], children[1])
		assert.Equal(t, manifest.Layers[1], children[2])
	})

	t.Run("IndexChildren", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		// Create an index with manifests
		index := ocispec.Index{
			Manifests: []ocispec.Descriptor{
				{
					MediaType: ocispec.MediaTypeImageManifest,
					Digest:    digest.FromString("manifest1"),
					Size:      100,
				},
				{
					MediaType: ocispec.MediaTypeImageManifest,
					Digest:    digest.FromString("manifest2"),
					Size:      200,
				},
			},
		}
		indexData, err := json.Marshal(index)
		require.NoError(t, err)

		indexDesc := provider.addBlob(ocispec.MediaTypeImageIndex, indexData)

		children, err := Children(ctx, provider, indexDesc)

		assert.NoError(t, err)
		assert.Len(t, children, 2)
		assert.Equal(t, index.Manifests[0], children[0])
		assert.Equal(t, index.Manifests[1], children[1])
	})

	t.Run("LayerNoChildren", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		layerDesc := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageLayerGzip,
			Digest:    digest.FromString("layer"),
			Size:      100,
		}

		children, err := Children(ctx, provider, layerDesc)

		assert.NoError(t, err)
		assert.Nil(t, children)
	})

	t.Run("InvalidMediaType", func(t *testing.T) {
		ctx := context.Background()
		provider := newMockContentProvider()

		invalidDesc := provider.addBlob(ocispec.MediaTypeImageManifest, []byte("invalid json"))

		_, err := Children(ctx, provider, invalidDesc)

		assert.Error(t, err)
	})
}

func TestValidateMediaTypeExtended(t *testing.T) {
	t.Run("ValidManifest", func(t *testing.T) {
		manifest := ocispec.Manifest{
			Config: ocispec.Descriptor{Size: 1},
			Layers: []ocispec.Descriptor{{Size: 2}},
		}
		b, err := json.Marshal(manifest)
		require.NoError(t, err)

		err = validateMediaType(b, ocispec.MediaTypeImageManifest)
		assert.NoError(t, err)
	})

	t.Run("ValidIndex", func(t *testing.T) {
		index := ocispec.Index{
			Manifests: []ocispec.Descriptor{{Size: 1}},
		}
		b, err := json.Marshal(index)
		require.NoError(t, err)

		err = validateMediaType(b, ocispec.MediaTypeImageIndex)
		assert.NoError(t, err)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		err := validateMediaType([]byte("invalid json"), ocispec.MediaTypeImageManifest)
		assert.Error(t, err)
	})

	t.Run("Schema1Rejected", func(t *testing.T) {
		schema1 := struct {
			FSLayers []string `json:"fsLayers"`
		}{FSLayers: []string{"layer1"}}
		b, err := json.Marshal(schema1)
		require.NoError(t, err)

		err = validateMediaType(b, ocispec.MediaTypeImageManifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema 1 not supported")
	})
}

func TestMediaTypeUtilities(t *testing.T) {
	t.Run("IsManifestType", func(t *testing.T) {
		assert.True(t, IsManifestType(ocispec.MediaTypeImageManifest))
		assert.True(t, IsManifestType(MediaTypeDockerSchema2Manifest))
		assert.False(t, IsManifestType(ocispec.MediaTypeImageIndex))
		assert.False(t, IsManifestType(ocispec.MediaTypeImageLayerGzip))
	})

	t.Run("IsIndexType", func(t *testing.T) {
		assert.True(t, IsIndexType(ocispec.MediaTypeImageIndex))
		assert.True(t, IsIndexType(MediaTypeDockerSchema2ManifestList))
		assert.False(t, IsIndexType(ocispec.MediaTypeImageManifest))
		assert.False(t, IsIndexType(ocispec.MediaTypeImageLayerGzip))
	})

	t.Run("IsLayerType", func(t *testing.T) {
		assert.True(t, IsLayerType(ocispec.MediaTypeImageLayerGzip))
		assert.True(t, IsLayerType(MediaTypeDockerSchema2LayerGzip))
		assert.False(t, IsLayerType(ocispec.MediaTypeImageManifest))
		assert.False(t, IsLayerType(ocispec.MediaTypeImageIndex))
	})
}

func TestHandlerErrors(t *testing.T) {
	t.Run("ErrorConstants", func(t *testing.T) {
		assert.Equal(t, "skip descriptor", ErrSkipDesc.Error())
		assert.Equal(t, "stop handler", ErrStopHandler.Error())
		assert.Equal(t, "image might be filtered out", ErrEmptyWalk.Error())
	})
}
