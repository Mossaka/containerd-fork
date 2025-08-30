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

package archive

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/platforms"
	"github.com/containerd/errdefs"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestWithPlatform(t *testing.T) {
	ctx := context.Background()
	platform := platforms.MustParse("linux/amd64")
	matcher := platforms.OnlyStrict(platform)
	
	opt := WithPlatform(matcher)
	var opts exportOptions
	
	err := opt(ctx, &opts)
	if err != nil {
		t.Fatalf("WithPlatform option failed: %v", err)
	}
	
	if opts.platform == nil {
		t.Error("Platform matcher not set")
	}
}

func TestWithAllPlatforms(t *testing.T) {
	ctx := context.Background()
	opt := WithAllPlatforms()
	var opts exportOptions
	
	err := opt(ctx, &opts)
	if err != nil {
		t.Fatalf("WithAllPlatforms option failed: %v", err)
	}
	
	if !opts.allPlatforms {
		t.Error("AllPlatforms not set to true")
	}
}

func TestWithSkipDockerManifest(t *testing.T) {
	ctx := context.Background()
	opt := WithSkipDockerManifest()
	var opts exportOptions
	
	err := opt(ctx, &opts)
	if err != nil {
		t.Fatalf("WithSkipDockerManifest option failed: %v", err)
	}
	
	if !opts.skipDockerManifest {
		t.Error("SkipDockerManifest not set to true")
	}
}

func TestWithImages(t *testing.T) {
	ctx := context.Background()
	
	imgs := []images.Image{
		{
			Name: "test-image:latest",
			Target: ocispec.Descriptor{
				MediaType: ocispec.MediaTypeImageManifest,
				Digest:    digest.FromString("test-manifest"),
				Size:      100,
			},
		},
	}
	
	opt := WithImages(imgs)
	var opts exportOptions
	
	err := opt(ctx, &opts)
	if err != nil {
		t.Fatalf("WithImages option failed: %v", err)
	}
	
	if len(opts.manifests) != 1 {
		t.Fatalf("Expected 1 manifest, got %d", len(opts.manifests))
	}
	
	manifest := opts.manifests[0]
	if manifest.Annotations[images.AnnotationImageName] != "test-image:latest" {
		t.Errorf("Expected image name annotation 'test-image:latest', got %s", 
			manifest.Annotations[images.AnnotationImageName])
	}
}

func TestWithManifest(t *testing.T) {
	ctx := context.Background()
	
	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromString("test-manifest"),
		Size:      100,
	}
	
	t.Run("WithoutNames", func(t *testing.T) {
		opt := WithManifest(desc)
		var opts exportOptions
		
		err := opt(ctx, &opts)
		if err != nil {
			t.Fatalf("WithManifest option failed: %v", err)
		}
		
		if len(opts.manifests) != 1 {
			t.Fatalf("Expected 1 manifest, got %d", len(opts.manifests))
		}
		
		if opts.manifests[0].Digest != desc.Digest {
			t.Error("Manifest digest mismatch")
		}
	})
	
	t.Run("WithNames", func(t *testing.T) {
		names := []string{"test1:latest", "test2:v1"}
		opt := WithManifest(desc, names...)
		var opts exportOptions
		
		err := opt(ctx, &opts)
		if err != nil {
			t.Fatalf("WithManifest option with names failed: %v", err)
		}
		
		if len(opts.manifests) != 2 {
			t.Fatalf("Expected 2 manifests, got %d", len(opts.manifests))
		}
		
		for i, manifest := range opts.manifests {
			expectedName := names[i]
			if manifest.Annotations[images.AnnotationImageName] != expectedName {
				t.Errorf("Expected name %s, got %s", expectedName, 
					manifest.Annotations[images.AnnotationImageName])
			}
		}
	})
}

func TestWithSkipNonDistributableBlobs(t *testing.T) {
	ctx := context.Background()
	opt := WithSkipNonDistributableBlobs()
	var opts exportOptions
	
	err := opt(ctx, &opts)
	if err != nil {
		t.Fatalf("WithSkipNonDistributableBlobs option failed: %v", err)
	}
	
	if opts.blobRecordOptions.blobFilter == nil {
		t.Error("BlobFilter not set")
		return
	}
	
	// Test with distributable blob
	distributableDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
	}
	if !opts.blobRecordOptions.blobFilter(distributableDesc) {
		t.Error("Distributable blob should be included")
	}
	
	// Test with non-distributable blob  
	nonDistributableDesc := ocispec.Descriptor{
		MediaType: images.MediaTypeDockerSchema2LayerForeign,
	}
	if opts.blobRecordOptions.blobFilter(nonDistributableDesc) {
		t.Error("Non-distributable blob should be excluded")
	}
}

func TestWithBlobFilter(t *testing.T) {
	ctx := context.Background()
	
	filter := func(desc ocispec.Descriptor) bool {
		return desc.Size > 100
	}
	
	opt := WithBlobFilter(filter)
	var opts exportOptions
	
	err := opt(ctx, &opts)
	if err != nil {
		t.Fatalf("WithBlobFilter option failed: %v", err)
	}
	
	if opts.blobRecordOptions.blobFilter == nil {
		t.Error("BlobFilter not set")
		return
	}
	
	// Test filter function
	smallBlob := ocispec.Descriptor{Size: 50}
	if opts.blobRecordOptions.blobFilter(smallBlob) {
		t.Error("Small blob should be filtered out")
	}
	
	largeBlob := ocispec.Descriptor{Size: 200}
	if !opts.blobRecordOptions.blobFilter(largeBlob) {
		t.Error("Large blob should be included")
	}
}

func TestAddNameAnnotation(t *testing.T) {
	tests := []struct {
		name        string
		imageName   string
		baseAnnotations map[string]string
		expectedName    string
		expectedRef     string
	}{
		{
			name:      "SimpleTag",
			imageName: "alpine:latest",
			baseAnnotations: nil,
			expectedName: "alpine:latest",
			expectedRef:  "alpine:latest", // ociReferenceName returns the full name if parsing fails
		},
		{
			name:      "FullReference", 
			imageName: "docker.io/library/alpine:v1.0",
			baseAnnotations: nil,
			expectedName: "docker.io/library/alpine:v1.0",
			expectedRef:  "v1.0",
		},
		{
			name:      "WithExistingAnnotations",
			imageName: "test:v1",
			baseAnnotations: map[string]string{
				"custom.annotation": "value",
			},
			expectedName: "test:v1",
			expectedRef:  "test:v1", // ociReferenceName returns the full name if parsing fails
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addNameAnnotation(tt.imageName, tt.baseAnnotations)
			
			if result[images.AnnotationImageName] != tt.expectedName {
				t.Errorf("Expected image name %s, got %s", 
					tt.expectedName, result[images.AnnotationImageName])
			}
			
			if result[ocispec.AnnotationRefName] != tt.expectedRef {
				t.Errorf("Expected ref name %s, got %s", 
					tt.expectedRef, result[ocispec.AnnotationRefName])
			}
			
			// Check existing annotations are preserved
			for k, v := range tt.baseAnnotations {
				if result[k] != v {
					t.Errorf("Expected annotation %s=%s, got %s", k, v, result[k])
				}
			}
		})
	}
}

type mockStore struct {
	blobs map[digest.Digest][]byte
	info  map[digest.Digest]content.Info
}

func newMockStore() *mockStore {
	return &mockStore{
		blobs: make(map[digest.Digest][]byte),
		info:  make(map[digest.Digest]content.Info),
	}
}

func (m *mockStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	data, exists := m.blobs[desc.Digest]
	if !exists {
		return nil, errdefs.ErrNotFound
	}
	return &mockReaderAt{data: data}, nil
}

func (m *mockStore) Info(ctx context.Context, dgst digest.Digest) (content.Info, error) {
	info, exists := m.info[dgst]
	if !exists {
		return content.Info{}, errdefs.ErrNotFound
	}
	return info, nil
}

func (m *mockStore) addBlob(data []byte, labels map[string]string) ocispec.Descriptor {
	dgst := digest.FromBytes(data)
	m.blobs[dgst] = data
	m.info[dgst] = content.Info{
		Digest: dgst,
		Size:   int64(len(data)),
		Labels: labels,
	}
	return ocispec.Descriptor{
		Digest: dgst,
		Size:   int64(len(data)),
	}
}

type mockReaderAt struct {
	data []byte
	pos  int64
}

func (m *mockReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[off:])
	if off+int64(n) >= int64(len(m.data)) {
		err = io.EOF
	}
	return n, err
}

func (m *mockReaderAt) Close() error {
	return nil
}

func (m *mockReaderAt) Size() int64 {
	return int64(len(m.data))
}

func TestCopySourceLabels(t *testing.T) {
	ctx := context.Background()
	
	store := newMockStore()
	data := []byte("test blob data")
	labels := map[string]string{
		"containerd.io/distribution.source.docker.io": "docker.io/library/alpine",
		"other.label": "should-not-copy",
	}
	desc := store.addBlob(data, labels)
	
	result, err := copySourceLabels(ctx, store, desc)
	if err != nil {
		t.Fatalf("copySourceLabels failed: %v", err)
	}
	
	expected := "containerd.io/distribution.source.docker.io"
	if result.Annotations[expected] != labels[expected] {
		t.Errorf("Expected annotation %s=%s, got %s", 
			expected, labels[expected], result.Annotations[expected])
	}
	
	if _, exists := result.Annotations["other.label"]; exists {
		t.Error("Non-source label should not be copied")
	}
}

func TestOciLayoutFile(t *testing.T) {
	t.Run("DefaultVersion", func(t *testing.T) {
		record := ociLayoutFile("")
		
		if record.Header.Name != ocispec.ImageLayoutFile {
			t.Errorf("Expected name %s, got %s", ocispec.ImageLayoutFile, record.Header.Name)
		}
		
		var buf bytes.Buffer
		_, err := record.CopyTo(context.Background(), &buf)
		if err != nil {
			t.Fatalf("CopyTo failed: %v", err)
		}
		
		var layout ocispec.ImageLayout
		if err := json.Unmarshal(buf.Bytes(), &layout); err != nil {
			t.Fatalf("Failed to unmarshal layout: %v", err)
		}
		
		if layout.Version != ocispec.ImageLayoutVersion {
			t.Errorf("Expected version %s, got %s", ocispec.ImageLayoutVersion, layout.Version)
		}
	})
	
	t.Run("CustomVersion", func(t *testing.T) {
		customVersion := "1.0.0"
		record := ociLayoutFile(customVersion)
		
		var buf bytes.Buffer
		_, err := record.CopyTo(context.Background(), &buf)
		if err != nil {
			t.Fatalf("CopyTo failed: %v", err)
		}
		
		var layout ocispec.ImageLayout
		if err := json.Unmarshal(buf.Bytes(), &layout); err != nil {
			t.Fatalf("Failed to unmarshal layout: %v", err)
		}
		
		if layout.Version != customVersion {
			t.Errorf("Expected version %s, got %s", customVersion, layout.Version)
		}
	})
}

func TestOciIndexRecord(t *testing.T) {
	manifests := []ocispec.Descriptor{
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
	}
	
	record := ociIndexRecord(manifests)
	
	if record.Header.Name != ocispec.ImageIndexFile {
		t.Errorf("Expected name %s, got %s", ocispec.ImageIndexFile, record.Header.Name)
	}
	
	var buf bytes.Buffer
	_, err := record.CopyTo(context.Background(), &buf)
	if err != nil {
		t.Fatalf("CopyTo failed: %v", err)
	}
	
	var index ocispec.Index
	if err := json.Unmarshal(buf.Bytes(), &index); err != nil {
		t.Fatalf("Failed to unmarshal index: %v", err)
	}
	
	if index.MediaType != ocispec.MediaTypeImageIndex {
		t.Errorf("Expected media type %s, got %s", ocispec.MediaTypeImageIndex, index.MediaType)
	}
	
	if len(index.Manifests) != len(manifests) {
		t.Errorf("Expected %d manifests, got %d", len(manifests), len(index.Manifests))
	}
	
	for i, manifest := range index.Manifests {
		if manifest.Digest != manifests[i].Digest {
			t.Errorf("Manifest %d digest mismatch: expected %s, got %s", 
				i, manifests[i].Digest, manifest.Digest)
		}
	}
}

func TestBlobRecord(t *testing.T) {
	store := newMockStore()
	data := []byte("test blob content for verification")
	desc := store.addBlob(data, nil)
	desc.MediaType = ocispec.MediaTypeImageLayer
	
	t.Run("WithoutFilter", func(t *testing.T) {
		record := blobRecord(store, desc, nil)
		
		expectedPath := "blobs/" + desc.Digest.Algorithm().String() + "/" + desc.Digest.Encoded()
		if record.Header.Name != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, record.Header.Name)
		}
		
		if record.Header.Size != desc.Size {
			t.Errorf("Expected size %d, got %d", desc.Size, record.Header.Size)
		}
		
		var buf bytes.Buffer
		n, err := record.CopyTo(context.Background(), &buf)
		if err != nil {
			t.Fatalf("CopyTo failed: %v", err)
		}
		
		if n != desc.Size {
			t.Errorf("Expected %d bytes copied, got %d", desc.Size, n)
		}
		
		if !bytes.Equal(buf.Bytes(), data) {
			t.Error("Copied data doesn't match original")
		}
	})
	
	t.Run("WithFilter", func(t *testing.T) {
		// Filter that excludes this blob
		filter := func(d ocispec.Descriptor) bool {
			return false
		}
		opts := &blobRecordOptions{blobFilter: filter}
		
		record := blobRecord(store, desc, opts)
		
		// Should return empty record when filtered out
		if record.Header != nil {
			t.Error("Expected nil header when blob is filtered out")
		}
	})
}

func TestDirectoryRecord(t *testing.T) {
	name := "test-dir/"
	mode := int64(0755)
	
	record := directoryRecord(name, mode)
	
	if record.Header.Name != name {
		t.Errorf("Expected name %s, got %s", name, record.Header.Name)
	}
	
	if record.Header.Mode != mode {
		t.Errorf("Expected mode %o, got %o", mode, record.Header.Mode)
	}
	
	if record.Header.Typeflag != tar.TypeDir {
		t.Errorf("Expected typeflag %c, got %c", tar.TypeDir, record.Header.Typeflag)
	}
	
	if record.CopyTo != nil {
		t.Error("Directory record should not have CopyTo function")
	}
}

func TestWriteTar(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	
	records := []tarRecord{
		directoryRecord("test-dir/", 0755),
		{
			Header: &tar.Header{
				Name:     "test-file.txt",
				Size:     5,
				Mode:     0644,
				Typeflag: tar.TypeReg,
			},
			CopyTo: func(ctx context.Context, w io.Writer) (int64, error) {
				n, err := w.Write([]byte("hello"))
				return int64(n), err
			},
		},
		// Empty record that should be filtered out
		{},
		// Duplicate file name (should be deduplicated)
		{
			Header: &tar.Header{
				Name:     "test-file.txt",
				Size:     5,
				Mode:     0644,
				Typeflag: tar.TypeReg,
			},
			CopyTo: func(ctx context.Context, w io.Writer) (int64, error) {
				n, err := w.Write([]byte("world"))
				return int64(n), err
			},
		},
	}
	
	err := writeTar(context.Background(), tw, records)
	if err != nil {
		t.Fatalf("writeTar failed: %v", err)
	}
	
	tw.Close()
	
	// Read back the tar and verify
	tr := tar.NewReader(&buf)
	
	// Should have directory first (alphabetical order)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}
	if hdr.Name != "test-dir/" || hdr.Typeflag != tar.TypeDir {
		t.Errorf("Expected directory test-dir/, got %s (type %c)", hdr.Name, hdr.Typeflag)
	}
	
	// Should have file second
	hdr, err = tr.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}
	if hdr.Name != "test-file.txt" || hdr.Typeflag != tar.TypeReg {
		t.Errorf("Expected file test-file.txt, got %s (type %c)", hdr.Name, hdr.Typeflag)
	}
	
	// Read file content
	content := make([]byte, hdr.Size)
	_, err = io.ReadFull(tr, content)
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("Expected file content 'hello', got %s", string(content))
	}
	
	// Should be no more entries (duplicate was deduplicated)
	_, err = tr.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}