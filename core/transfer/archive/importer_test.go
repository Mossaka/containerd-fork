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
	"context"
	"io"
	"strings"
	"testing"

	"github.com/containerd/typeurl/v2"

	transferapi "github.com/containerd/containerd/api/types/transfer"
)

func TestWithForceCompression(t *testing.T) {
	stream := strings.NewReader("test data")
	importer := NewImageImportStream(stream, "application/vnd.docker.image.manifest.v2+json")

	if importer.forceCompress {
		t.Fatal("expected forceCompress to be false by default")
	}

	WithForceCompression(importer)

	if !importer.forceCompress {
		t.Fatal("expected forceCompress to be true after applying option")
	}
}

func TestNewImageImportStream(t *testing.T) {
	stream := strings.NewReader("test data")
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	importer := NewImageImportStream(stream, mediaType, WithForceCompression)

	if importer.stream != stream {
		t.Fatal("stream not set correctly")
	}
	if importer.mediaType != mediaType {
		t.Fatal("mediaType not set correctly")
	}
	if !importer.forceCompress {
		t.Fatal("forceCompress not set by option")
	}
}

func TestNewImageImportStream_NoOptions(t *testing.T) {
	stream := strings.NewReader("test data")
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	importer := NewImageImportStream(stream, mediaType)

	if importer.stream != stream {
		t.Fatal("stream not set correctly")
	}
	if importer.mediaType != mediaType {
		t.Fatal("mediaType not set correctly")
	}
	if importer.forceCompress {
		t.Fatal("forceCompress should be false by default")
	}
}

func TestImageImportStream_ImportStream(t *testing.T) {
	stream := strings.NewReader("test data")
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	importer := NewImageImportStream(stream, mediaType)

	ctx := context.Background()
	importStream, importMediaType, err := importer.ImportStream(ctx)
	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	if importStream != stream {
		t.Fatal("imported stream does not match original")
	}
	if importMediaType != mediaType {
		t.Fatal("imported media type does not match original")
	}
}

func TestImageImportStream_Import(t *testing.T) {
	// Test with non-empty media type (no decompression)
	stream := strings.NewReader("test tar data")
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	importer := NewImageImportStream(stream, mediaType)

	ctx := context.Background()
	store := &mockContentStore{}

	// This will fail because our mock doesn't implement actual archive functionality
	// but we can verify the method is callable
	_, err := importer.Import(ctx, store)
	// We expect an error since we're using a mock content store and fake data
	if err == nil {
		t.Log("Import succeeded (unexpected with mock)")
	}

	// Test with force compression
	importer2 := NewImageImportStream(stream, mediaType, WithForceCompression)
	_, err = importer2.Import(ctx, store)
	// We expect an error since we're using a mock content store and fake data
	if err == nil {
		t.Log("Import with force compression succeeded (unexpected with mock)")
	}
}

func TestImageImportStream_Import_EmptyMediaType(t *testing.T) {
	// Test with empty media type (triggers decompression)
	stream := strings.NewReader("test data")
	mediaType := ""

	importer := NewImageImportStream(stream, mediaType)

	ctx := context.Background()
	store := &mockContentStore{}

	// This should fail during decompression since our test data isn't compressed
	_, err := importer.Import(ctx, store)
	if err == nil {
		t.Log("Import with empty media type succeeded (unexpected)")
	}
	// Error is expected due to decompression failure with invalid data
}

func TestImageImportStream_MarshalAny(t *testing.T) {
	stream := strings.NewReader("test data")
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	importer := NewImageImportStream(stream, mediaType, WithForceCompression)

	ctx := context.Background()
	sm := &mockStreamCreator{}

	anyType, err := importer.MarshalAny(ctx, sm)
	if err != nil {
		t.Fatalf("MarshalAny failed: %v", err)
	}

	// Verify the marshaled type contains expected data
	var s transferapi.ImageImportStream
	if err := typeurl.UnmarshalTo(anyType, &s); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if s.MediaType != mediaType {
		t.Fatalf("expected media type %s, got %s", mediaType, s.MediaType)
	}
	if !s.ForceCompress {
		t.Fatal("ForceCompress not marshaled correctly")
	}
	if s.Stream == "" {
		t.Fatal("Stream ID not set")
	}
}

func TestImageImportStream_UnmarshalAny(t *testing.T) {
	ctx := context.Background()
	sm := &mockStreamGetter{}

	// Create a transferapi.ImageImportStream and marshal it
	s := &transferapi.ImageImportStream{
		Stream:        "test-stream-id",
		MediaType:     "application/vnd.docker.image.manifest.v2+json",
		ForceCompress: true,
	}

	anyType, err := typeurl.MarshalAny(s)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Create an empty importer and unmarshal into it
	importer := &ImageImportStream{}
	err = importer.UnmarshalAny(ctx, sm, anyType)
	if err != nil {
		t.Fatalf("UnmarshalAny failed: %v", err)
	}

	// Verify the unmarshaled data
	if importer.mediaType != s.MediaType {
		t.Fatalf("expected media type %s, got %s", s.MediaType, importer.mediaType)
	}
	if !importer.forceCompress {
		t.Fatal("forceCompress not unmarshaled correctly")
	}
	if importer.stream == nil {
		t.Fatal("stream not set after unmarshal")
	}
}

func TestImageImportStream_UnmarshalAny_StreamError(t *testing.T) {
	ctx := context.Background()

	// Use mockStreamGetterWithError that returns errors
	sm := &mockStreamGetterWithError{}

	s := &transferapi.ImageImportStream{
		Stream:    "test-stream-id",
		MediaType: "application/vnd.docker.image.manifest.v2+json",
	}

	anyType, err := typeurl.MarshalAny(s)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	importer := &ImageImportStream{}
	err = importer.UnmarshalAny(ctx, sm, anyType)
	if err == nil {
		t.Fatal("expected error when stream getter fails, got nil")
	}
}

func TestImageImportStream_EdgeCases(t *testing.T) {
	// Test with nil reader (should not panic)
	var nilReader io.Reader
	importer := NewImageImportStream(nilReader, "test")
	if importer.stream != nilReader {
		t.Fatal("nil reader not handled correctly")
	}

	// Test with empty media type
	emptyReader := strings.NewReader("")
	importer2 := NewImageImportStream(emptyReader, "")
	if importer2.mediaType != "" {
		t.Fatal("empty media type not handled correctly")
	}

	// Test ImportStream with various inputs
	ctx := context.Background()

	// Empty reader
	stream, mediaType, err := importer2.ImportStream(ctx)
	if err != nil {
		t.Fatalf("ImportStream with empty reader failed: %v", err)
	}
	if stream != emptyReader {
		t.Fatal("stream mismatch with empty reader")
	}
	if mediaType != "" {
		t.Fatal("media type mismatch with empty string")
	}
}

// Test concurrent access to verify thread safety
func TestImageImportStream_ConcurrentAccess(t *testing.T) {
	stream := strings.NewReader("test data")
	mediaType := "application/vnd.docker.image.manifest.v2+json"

	importer := NewImageImportStream(stream, mediaType)
	ctx := context.Background()

	// Run multiple goroutines accessing the same importer
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test concurrent access to ImportStream
			s, mt, err := importer.ImportStream(ctx)
			if err != nil {
				t.Errorf("ImportStream failed: %v", err)
				return
			}
			if s != stream || mt != mediaType {
				t.Errorf("ImportStream returned wrong values")
				return
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test with various media types
func TestImageImportStream_MediaTypes(t *testing.T) {
	testCases := []struct {
		name      string
		mediaType string
	}{
		{"Docker manifest", "application/vnd.docker.distribution.manifest.v2+json"},
		{"OCI manifest", "application/vnd.oci.image.manifest.v1+json"},
		{"Docker index", "application/vnd.docker.distribution.manifest.list.v2+json"},
		{"OCI index", "application/vnd.oci.image.index.v1+json"},
		{"Empty", ""},
		{"Custom", "application/custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stream := strings.NewReader("test data")
			importer := NewImageImportStream(stream, tc.mediaType)

			if importer.mediaType != tc.mediaType {
				t.Fatalf("expected media type %s, got %s", tc.mediaType, importer.mediaType)
			}

			// Test that ImportStream preserves the media type
			ctx := context.Background()
			_, returnedMediaType, err := importer.ImportStream(ctx)
			if err != nil {
				t.Fatalf("ImportStream failed: %v", err)
			}
			if returnedMediaType != tc.mediaType {
				t.Fatalf("ImportStream returned wrong media type: expected %s, got %s", tc.mediaType, returnedMediaType)
			}
		})
	}
}
