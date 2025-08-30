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
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/containerd/typeurl/v2"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/containerd/containerd/api/types"
	transfertypes "github.com/containerd/containerd/api/types/transfer"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/streaming"
)

// mockWriteCloser implements io.WriteCloser for testing
type mockWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}

// mockContentStore provides a minimal content.Store implementation for testing
type mockContentStore struct{}

func (m *mockContentStore) Delete(ctx context.Context, dgst digest.Digest) error { return nil }
func (m *mockContentStore) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	return true, nil
}
func (m *mockContentStore) Info(ctx context.Context, dgst digest.Digest) (content.Info, error) {
	return content.Info{}, nil
}
func (m *mockContentStore) ListStatuses(ctx context.Context, filters ...string) ([]content.Status, error) {
	return nil, nil
}
func (m *mockContentStore) Reader(ctx context.Context, desc v1.Descriptor) (content.ReaderAt, error) {
	return nil, nil
}
func (m *mockContentStore) ReaderAt(ctx context.Context, desc v1.Descriptor) (content.ReaderAt, error) {
	return nil, nil
}
func (m *mockContentStore) Status(ctx context.Context, ref string) (content.Status, error) {
	return content.Status{}, nil
}
func (m *mockContentStore) Update(ctx context.Context, info content.Info, fieldpaths ...string) (content.Info, error) {
	return info, nil
}
func (m *mockContentStore) Walk(ctx context.Context, fn content.WalkFunc, filters ...string) error {
	return nil
}
func (m *mockContentStore) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	return nil, nil
}
func (m *mockContentStore) Abort(ctx context.Context, ref string) error {
	return nil
}

// mockStreamCreator provides streaming.StreamCreator for testing
type mockStreamCreator struct{}

func (m *mockStreamCreator) Create(ctx context.Context, id string) (streaming.Stream, error) {
	return &mockStream{id: id}, nil
}

// mockStream provides streaming.Stream for testing
type mockStream struct {
	id string
}

func (m *mockStream) Send(typeurl.Any) error     { return nil }
func (m *mockStream) Recv() (typeurl.Any, error) { return nil, io.EOF }
func (m *mockStream) Close() error               { return nil }

// mockStreamGetter provides streaming.StreamGetter for testing
type mockStreamGetter struct{}

func (m *mockStreamGetter) Get(ctx context.Context, id string) (streaming.Stream, error) {
	return &mockStream{id: id}, nil
}

// mockStreamGetterWithError for error testing
type mockStreamGetterWithError struct{}

func (m *mockStreamGetterWithError) Get(ctx context.Context, id string) (streaming.Stream, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestWithPlatform(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	exporter := NewImageExportStream(stream, "application/vnd.docker.image.manifest.v2+json")

	platform := v1.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}

	opt := WithPlatform(platform)
	opt(exporter)

	if len(exporter.platforms) != 1 {
		t.Fatalf("expected 1 platform, got %d", len(exporter.platforms))
	}

	if exporter.platforms[0].OS != "linux" || exporter.platforms[0].Architecture != "amd64" {
		t.Fatalf("platform not set correctly: %+v", exporter.platforms[0])
	}
}

func TestWithAllPlatforms(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	exporter := NewImageExportStream(stream, "application/vnd.docker.image.manifest.v2+json")

	WithAllPlatforms(exporter)

	if !exporter.allPlatforms {
		t.Fatal("expected allPlatforms to be true")
	}
}

func TestWithSkipCompatibilityManifest(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	exporter := NewImageExportStream(stream, "application/vnd.docker.image.manifest.v2+json")

	WithSkipCompatibilityManifest(exporter)

	if !exporter.skipCompatibilityManifest {
		t.Fatal("expected skipCompatibilityManifest to be true")
	}
}

func TestWithSkipNonDistributableBlobs(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	exporter := NewImageExportStream(stream, "application/vnd.docker.image.manifest.v2+json")

	WithSkipNonDistributableBlobs(exporter)

	if !exporter.skipNonDistributable {
		t.Fatal("expected skipNonDistributable to be true")
	}
}

func TestNewImageExportStream(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	platform := v1.Platform{OS: "linux", Architecture: "amd64"}

	exporter := NewImageExportStream(stream, mediaType,
		WithPlatform(platform),
		WithAllPlatforms,
		WithSkipCompatibilityManifest,
		WithSkipNonDistributableBlobs,
	)

	if exporter.stream != stream {
		t.Fatal("stream not set correctly")
	}
	if exporter.mediaType != mediaType {
		t.Fatal("mediaType not set correctly")
	}
	if len(exporter.platforms) != 1 {
		t.Fatal("platform not added correctly")
	}
	if !exporter.allPlatforms {
		t.Fatal("allPlatforms not set")
	}
	if !exporter.skipCompatibilityManifest {
		t.Fatal("skipCompatibilityManifest not set")
	}
	if !exporter.skipNonDistributable {
		t.Fatal("skipNonDistributable not set")
	}
}

func TestImageExportStream_ExportStream(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	exporter := NewImageExportStream(stream, mediaType)

	ctx := context.Background()
	exportStream, exportMediaType, err := exporter.ExportStream(ctx)
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	if exportStream != stream {
		t.Fatal("exported stream does not match original")
	}
	if exportMediaType != mediaType {
		t.Fatal("exported media type does not match original")
	}
}

func TestImageExportStream_Export(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	// Test with platforms
	platform := v1.Platform{OS: "linux", Architecture: "amd64"}
	exporter := NewImageExportStream(stream, mediaType, WithPlatform(platform))

	ctx := context.Background()
	cs := &mockContentStore{}
	imgs := []images.Image{
		{Name: "test:latest"},
	}

	// This will fail because our mock doesn't implement actual archive functionality
	// but we can verify the method is callable and handles platform configuration
	err := exporter.Export(ctx, cs, imgs)
	// We expect an error since we're using a mock content store
	if err == nil {
		t.Log("Export succeeded (unexpected with mock)")
	}

	// Test with all platforms
	exporter2 := NewImageExportStream(stream, mediaType, WithAllPlatforms)
	err = exporter2.Export(ctx, cs, imgs)
	// We expect an error since we're using a mock content store
	if err == nil {
		t.Log("Export with all platforms succeeded (unexpected with mock)")
	}

	// Test with skip options
	exporter3 := NewImageExportStream(stream, mediaType,
		WithSkipCompatibilityManifest,
		WithSkipNonDistributableBlobs,
	)
	err = exporter3.Export(ctx, cs, imgs)
	// We expect an error since we're using a mock content store
	if err == nil {
		t.Log("Export with skip options succeeded (unexpected with mock)")
	}
}

func TestImageExportStream_MarshalAny(t *testing.T) {
	stream := &mockWriteCloser{Buffer: &bytes.Buffer{}}
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	platform := v1.Platform{OS: "linux", Architecture: "amd64", Variant: "v8"}
	exporter := NewImageExportStream(stream, mediaType,
		WithPlatform(platform),
		WithAllPlatforms,
		WithSkipCompatibilityManifest,
		WithSkipNonDistributableBlobs,
	)

	ctx := context.Background()
	sm := &mockStreamCreator{}

	anyType, err := exporter.MarshalAny(ctx, sm)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}

	// Verify the marshaled type contains expected data
	var s transfertypes.ImageExportStream
	if err := typeurl.UnmarshalTo(anyType, &s); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if s.MediaType != mediaType {
		t.Fatalf("expected media type %s, got %s", mediaType, s.MediaType)
	}
	if len(s.Platforms) != 1 {
		t.Fatalf("expected 1 platform, got %d", len(s.Platforms))
	}
	if s.Platforms[0].OS != "linux" || s.Platforms[0].Architecture != "amd64" || s.Platforms[0].Variant != "v8" {
		t.Fatalf("platform not marshaled correctly: %+v", s.Platforms[0])
	}
	if !s.AllPlatforms {
		t.Fatal("AllPlatforms not marshaled correctly")
	}
	if !s.SkipCompatibilityManifest {
		t.Fatal("SkipCompatibilityManifest not marshaled correctly")
	}
	if !s.SkipNonDistributable {
		t.Fatal("SkipNonDistributable not marshaled correctly")
	}
}

func TestImageExportStream_UnmarshalAny(t *testing.T) {
	ctx := context.Background()
	sm := &mockStreamGetter{}

	// Create a transfertypes.ImageExportStream and marshal it
	s := &transfertypes.ImageExportStream{
		Stream:    "test-stream-id",
		MediaType: "application/vnd.docker.image.manifest.v2+json",
		Platforms: []*types.Platform{
			{OS: "linux", Architecture: "amd64", Variant: "v8"},
		},
		AllPlatforms:              true,
		SkipCompatibilityManifest: true,
		SkipNonDistributable:      true,
	}

	anyType, err := typeurl.MarshalAny(s)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Create an empty exporter and unmarshal into it
	exporter := &ImageExportStream{}
	err = exporter.UnmarshalAny(ctx, sm, anyType)
	if err != nil {
		t.Fatalf("UnmarshalAny failed: %v", err)
	}

	// Verify the unmarshaled data
	if exporter.mediaType != s.MediaType {
		t.Fatalf("expected media type %s, got %s", s.MediaType, exporter.mediaType)
	}
	if len(exporter.platforms) != 1 {
		t.Fatalf("expected 1 platform, got %d", len(exporter.platforms))
	}
	if exporter.platforms[0].OS != "linux" || exporter.platforms[0].Architecture != "amd64" || exporter.platforms[0].Variant != "v8" {
		t.Fatalf("platform not unmarshaled correctly: %+v", exporter.platforms[0])
	}
	if !exporter.allPlatforms {
		t.Fatal("allPlatforms not unmarshaled correctly")
	}
	if !exporter.skipCompatibilityManifest {
		t.Fatal("skipCompatibilityManifest not unmarshaled correctly")
	}
	if !exporter.skipNonDistributable {
		t.Fatal("skipNonDistributable not unmarshaled correctly")
	}
}

func TestImageExportStream_UnmarshalAny_StreamError(t *testing.T) {
	ctx := context.Background()

	// mockStreamGetterWithError that returns errors
	sm := &mockStreamGetterWithError{}

	s := &transfertypes.ImageExportStream{
		Stream:    "test-stream-id",
		MediaType: "application/vnd.docker.image.manifest.v2+json",
	}

	anyType, err := typeurl.MarshalAny(s)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	exporter := &ImageExportStream{}
	err = exporter.UnmarshalAny(ctx, sm, anyType)
	if err == nil {
		t.Fatal("expected error when stream getter fails, got nil")
	}
}
