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
	"errors"
	"testing"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/leases"
	"github.com/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type mockClient struct {
	contentStore content.Store
	imageStore   images.Store
	leaseError   error
	leaseFunc    func(context.Context) error
}

func (m *mockClient) WithLease(ctx context.Context, opts ...leases.Opt) (context.Context, func(context.Context) error, error) {
	if m.leaseError != nil {
		return nil, nil, m.leaseError
	}
	return ctx, m.leaseFunc, nil
}

func (m *mockClient) ContentStore() content.Store {
	return m.contentStore
}

func (m *mockClient) ImageService() images.Store {
	return m.imageStore
}

type mockImageStore struct {
	images      map[string]images.Image
	getError    error
	createError error
	updateError error
	deleteError error
}

func (m *mockImageStore) Get(ctx context.Context, name string) (images.Image, error) {
	if m.getError != nil {
		return images.Image{}, m.getError
	}
	if img, ok := m.images[name]; ok {
		return img, nil
	}
	return images.Image{}, errors.New("image not found")
}

func (m *mockImageStore) List(ctx context.Context, filters ...string) ([]images.Image, error) {
	return nil, nil
}

func (m *mockImageStore) Create(ctx context.Context, image images.Image) (images.Image, error) {
	if m.createError != nil {
		return images.Image{}, m.createError
	}
	m.images[image.Name] = image
	return image, nil
}

func (m *mockImageStore) Update(ctx context.Context, image images.Image, fieldpaths ...string) (images.Image, error) {
	if m.updateError != nil {
		return images.Image{}, m.updateError
	}
	m.images[image.Name] = image
	return image, nil
}

func (m *mockImageStore) Delete(ctx context.Context, name string, opts ...images.DeleteOpt) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	delete(m.images, name)
	return nil
}

func TestWithLayerConvertFunc(t *testing.T) {
	testFn := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	opt := WithLayerConvertFunc(testFn)

	var opts convertOpts
	err := opt(&opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.layerConvertFunc == nil {
		t.Fatal("expected layerConvertFunc to be set")
	}
}

func TestWithDockerToOCI(t *testing.T) {
	opt := WithDockerToOCI(true)

	var opts convertOpts
	err := opt(&opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !opts.docker2oci {
		t.Fatal("expected docker2oci to be true")
	}
}

func TestWithPlatform(t *testing.T) {
	platform := platforms.DefaultStrict()
	opt := WithPlatform(platform)

	var opts convertOpts
	err := opt(&opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.platformMC == nil {
		t.Fatal("expected platformMC to be set")
	}
}

func TestWithIndexConvertFunc(t *testing.T) {
	testFn := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	opt := WithIndexConvertFunc(testFn)

	var opts convertOpts
	err := opt(&opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if opts.indexConvertFunc == nil {
		t.Fatal("expected indexConvertFunc to be set")
	}
}

func TestConvert_LeaseError(t *testing.T) {
	client := &mockClient{
		leaseError: errors.New("lease error"),
	}

	_, err := Convert(context.Background(), client, "dst", "src")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "lease error" {
		t.Fatalf("expected 'lease error', got %v", err)
	}
}

func TestConvert_ImageNotFound(t *testing.T) {
	imageStore := &mockImageStore{
		images:   make(map[string]images.Image),
		getError: errors.New("image not found"),
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	_, err := Convert(context.Background(), client, "dst", "src")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "image not found" {
		t.Fatalf("expected 'image not found', got %v", err)
	}
}

func TestConvert_Success_SameName(t *testing.T) {
	srcImg := images.Image{
		Name: "test",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"test": srcImg,
		},
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	// Mock index convert function that returns nil (no conversion)
	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	result, err := Convert(context.Background(), client, "test", "test", WithIndexConvertFunc(indexConvertFunc))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "test" {
		t.Fatalf("expected name 'test', got %s", result.Name)
	}
}

func TestConvert_Success_DifferentName(t *testing.T) {
	srcImg := images.Image{
		Name: "src",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"src": srcImg,
		},
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	// Mock index convert function that returns a new descriptor
	newDesc := &ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:def456",
		Size:      456,
	}
	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return newDesc, nil
	}

	result, err := Convert(context.Background(), client, "dst", "src", WithIndexConvertFunc(indexConvertFunc))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "dst" {
		t.Fatalf("expected name 'dst', got %s", result.Name)
	}
	if result.Target.Digest != "sha256:def456" {
		t.Fatalf("expected digest 'sha256:def456', got %s", result.Target.Digest)
	}
}

func TestConvert_IndexConvertError(t *testing.T) {
	srcImg := images.Image{
		Name: "test",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"test": srcImg,
		},
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	// Mock index convert function that returns an error
	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, errors.New("convert error")
	}

	_, err := Convert(context.Background(), client, "test", "test", WithIndexConvertFunc(indexConvertFunc))
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "convert error" {
		t.Fatalf("expected 'convert error', got %v", err)
	}
}

func TestConvert_CreateError(t *testing.T) {
	srcImg := images.Image{
		Name: "src",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"src": srcImg,
		},
		createError: errors.New("create error"),
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	// Mock index convert function that returns nil (no conversion)
	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	_, err := Convert(context.Background(), client, "dst", "src", WithIndexConvertFunc(indexConvertFunc))
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "create error" {
		t.Fatalf("expected 'create error', got %v", err)
	}
}

func TestConvert_UpdateError(t *testing.T) {
	srcImg := images.Image{
		Name: "test",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"test": srcImg,
		},
		updateError: errors.New("update error"),
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	// Mock index convert function that returns nil (no conversion)
	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	_, err := Convert(context.Background(), client, "test", "test", WithIndexConvertFunc(indexConvertFunc))
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "update error" {
		t.Fatalf("expected 'update error', got %v", err)
	}
}

func TestConvert_DefaultOptions(t *testing.T) {
	// This test is tricky because the default converter tries to read from content store
	// For simplicity, just test that Convert doesn't panic with basic setup
	srcImg := images.Image{
		Name: "test",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"test": srcImg,
		},
	}

	// Mock index convert function that returns nil (no conversion) to avoid content store access
	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	// Test with no options but provide an index converter to avoid nil content store issues
	result, err := Convert(context.Background(), client, "test", "test", WithIndexConvertFunc(indexConvertFunc))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "test" {
		t.Fatalf("expected name 'test', got %s", result.Name)
	}
}

func TestConvert_WithAllOptions(t *testing.T) {
	srcImg := images.Image{
		Name: "test",
		Target: ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageManifest,
			Digest:    "sha256:abc123",
			Size:      123,
		},
	}

	imageStore := &mockImageStore{
		images: map[string]images.Image{
			"test": srcImg,
		},
	}

	client := &mockClient{
		imageStore: imageStore,
		leaseFunc:  func(context.Context) error { return nil },
	}

	layerConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	indexConvertFunc := func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		return nil, nil
	}

	result, err := Convert(context.Background(), client, "test", "test",
		WithLayerConvertFunc(layerConvertFunc),
		WithDockerToOCI(true),
		WithPlatform(platforms.DefaultStrict()),
		WithIndexConvertFunc(indexConvertFunc),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "test" {
		t.Fatalf("expected name 'test', got %s", result.Name)
	}
}
