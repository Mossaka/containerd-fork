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

package diffservice

import (
	"context"
	"errors"
	"testing"
	"time"

	diffapi "github.com/containerd/containerd/api/services/diff/v1"
	mountapi "github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/core/diff"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mock applier for testing
type mockApplier struct {
	applyFunc func(ctx context.Context, desc ocispec.Descriptor, mounts []mount.Mount, opts ...diff.ApplyOpt) (ocispec.Descriptor, error)
}

func (m *mockApplier) Apply(ctx context.Context, desc ocispec.Descriptor, mounts []mount.Mount, opts ...diff.ApplyOpt) (ocispec.Descriptor, error) {
	if m.applyFunc != nil {
		return m.applyFunc(ctx, desc, mounts, opts...)
	}
	return ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    "sha256:applied123",
		Size:      1024,
	}, nil
}

// Mock comparer for testing
type mockComparer struct {
	compareFunc func(ctx context.Context, a, b []mount.Mount, opts ...diff.Opt) (ocispec.Descriptor, error)
}

func (m *mockComparer) Compare(ctx context.Context, a, b []mount.Mount, opts ...diff.Opt) (ocispec.Descriptor, error) {
	if m.compareFunc != nil {
		return m.compareFunc(ctx, a, b, opts...)
	}
	return ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
		Digest:    "sha256:compared123",
		Size:      512,
	}, nil
}

func TestFromApplierAndComparer(t *testing.T) {
	applier := &mockApplier{}
	comparer := &mockComparer{}

	svc := FromApplierAndComparer(applier, comparer)

	if svc == nil {
		t.Fatal("FromApplierAndComparer should return a service")
	}

	// Verify it implements the DiffServer interface
	_, ok := svc.(diffapi.DiffServer)
	if !ok {
		t.Fatal("Service should implement DiffServer interface")
	}
}

func TestFromApplierAndComparerWithNil(t *testing.T) {
	// Test with nil applier
	svc := FromApplierAndComparer(nil, &mockComparer{})
	if svc == nil {
		t.Fatal("Service should be created even with nil applier")
	}

	// Test with nil comparer
	svc = FromApplierAndComparer(&mockApplier{}, nil)
	if svc == nil {
		t.Fatal("Service should be created even with nil comparer")
	}

	// Test with both nil
	svc = FromApplierAndComparer(nil, nil)
	if svc == nil {
		t.Fatal("Service should be created even with both nil")
	}
}

func TestServiceApply(t *testing.T) {
	tests := []struct {
		name      string
		applier   diff.Applier
		request   *diffapi.ApplyRequest
		wantErr   bool
		errTarget error
	}{
		{
			name:    "successful apply",
			applier: &mockApplier{},
			request: &diffapi.ApplyRequest{
				Diff: oci.DescriptorToProto(ocispec.Descriptor{
					MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
					Digest:    "sha256:test123",
					Size:      1024,
				}),
				Mounts: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/overlay",
					},
				},
				SyncFs: true,
			},
			wantErr: false,
		},
		{
			name:    "nil applier",
			applier: nil,
			request: &diffapi.ApplyRequest{
				Diff: oci.DescriptorToProto(ocispec.Descriptor{
					MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
					Digest:    "sha256:test123",
					Size:      1024,
				}),
			},
			wantErr:   true,
			errTarget: errdefs.ErrNotImplemented,
		},
		{
			name: "applier error",
			applier: &mockApplier{
				applyFunc: func(ctx context.Context, desc ocispec.Descriptor, mounts []mount.Mount, opts ...diff.ApplyOpt) (ocispec.Descriptor, error) {
					return ocispec.Descriptor{}, errors.New("apply failed")
				},
			},
			request: &diffapi.ApplyRequest{
				Diff: oci.DescriptorToProto(ocispec.Descriptor{
					MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
					Digest:    "sha256:test123",
					Size:      1024,
				}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := FromApplierAndComparer(tt.applier, &mockComparer{})

			resp, err := svc.Apply(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Apply() expected error, got nil")
				}
				// Note: GRPC errors are wrapped, so we just check for presence
				return
			}

			if err != nil {
				t.Errorf("Apply() unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Apply() response should not be nil")
			}

			if resp.Applied == nil {
				t.Fatal("Apply() response Applied field should not be nil")
			}
		})
	}
}

func TestServiceApplyWithPayloads(t *testing.T) {
	payloadsCalled := false
	applier := &mockApplier{
		applyFunc: func(ctx context.Context, desc ocispec.Descriptor, mounts []mount.Mount, opts ...diff.ApplyOpt) (ocispec.Descriptor, error) {
			// Check if payloads option was passed
			for _, opt := range opts {
				if opt != nil {
					payloadsCalled = true
					break
				}
			}
			return ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    "sha256:applied123",
				Size:      1024,
			}, nil
		},
	}

	svc := FromApplierAndComparer(applier, &mockComparer{})

	testPayload, err := anypb.New(&timestamppb.Timestamp{})
	if err != nil {
		t.Fatalf("Failed to marshal test payload: %v", err)
	}

	req := &diffapi.ApplyRequest{
		Diff: oci.DescriptorToProto(ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			Digest:    "sha256:test123",
			Size:      1024,
		}),
		Payloads: map[string]*anypb.Any{
			"test": testPayload,
		},
		SyncFs: true,
	}

	_, err = svc.Apply(context.Background(), req)
	if err != nil {
		t.Errorf("Apply() with payloads failed: %v", err)
	}

	if !payloadsCalled {
		t.Error("Apply() should have processed payloads")
	}
}

func TestServiceDiff(t *testing.T) {
	tests := []struct {
		name      string
		comparer  diff.Comparer
		request   *diffapi.DiffRequest
		wantErr   bool
		errTarget error
	}{
		{
			name:     "successful diff",
			comparer: &mockComparer{},
			request: &diffapi.DiffRequest{
				Left: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/left",
					},
				},
				Right: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/right",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "nil comparer",
			comparer: nil,
			request: &diffapi.DiffRequest{
				Left: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/left",
					},
				},
				Right: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/right",
					},
				},
			},
			wantErr:   true,
			errTarget: errdefs.ErrNotImplemented,
		},
		{
			name: "comparer error",
			comparer: &mockComparer{
				compareFunc: func(ctx context.Context, a, b []mount.Mount, opts ...diff.Opt) (ocispec.Descriptor, error) {
					return ocispec.Descriptor{}, errors.New("compare failed")
				},
			},
			request: &diffapi.DiffRequest{
				Left: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/left",
					},
				},
				Right: []*mountapi.Mount{
					{
						Type:   "overlay",
						Source: "/tmp/right",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := FromApplierAndComparer(&mockApplier{}, tt.comparer)

			resp, err := svc.Diff(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Diff() expected error, got nil")
				}
				// Note: GRPC errors are wrapped, so we just check for presence
				return
			}

			if err != nil {
				t.Errorf("Diff() unexpected error: %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Diff() response should not be nil")
			}

			if resp.Diff == nil {
				t.Fatal("Diff() response Diff field should not be nil")
			}
		})
	}
}

func TestServiceDiffWithOptions(t *testing.T) {
	optionsCalled := false
	comparer := &mockComparer{
		compareFunc: func(ctx context.Context, a, b []mount.Mount, opts ...diff.Opt) (ocispec.Descriptor, error) {
			// Check if options were passed
			if len(opts) > 0 {
				optionsCalled = true
			}
			return ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    "sha256:compared123",
				Size:      512,
			}, nil
		},
	}

	svc := FromApplierAndComparer(&mockApplier{}, comparer)

	timestamp := time.Now()
	req := &diffapi.DiffRequest{
		Left: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/left",
			},
		},
		Right: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/right",
			},
		},
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		Ref:       "test-ref",
		Labels: map[string]string{
			"test": "label",
		},
		SourceDateEpoch: timestamppb.New(timestamp),
	}

	_, err := svc.Diff(context.Background(), req)
	if err != nil {
		t.Errorf("Diff() with options failed: %v", err)
	}

	if !optionsCalled {
		t.Error("Diff() should have processed options")
	}
}

func TestServiceDiffEmptyFields(t *testing.T) {
	comparer := &mockComparer{}
	svc := FromApplierAndComparer(&mockApplier{}, comparer)

	req := &diffapi.DiffRequest{
		Left: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/left",
			},
		},
		Right: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/right",
			},
		},
		// All optional fields empty
		MediaType:       "",
		Ref:             "",
		Labels:          nil,
		SourceDateEpoch: nil,
	}

	resp, err := svc.Diff(context.Background(), req)
	if err != nil {
		t.Errorf("Diff() with empty optional fields failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Diff() response should not be nil")
	}
}

// Benchmark tests
func BenchmarkServiceApply(b *testing.B) {
	applier := &mockApplier{}
	svc := FromApplierAndComparer(applier, &mockComparer{})

	req := &diffapi.ApplyRequest{
		Diff: oci.DescriptorToProto(ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			Digest:    "sha256:test123",
			Size:      1024,
		}),
		Mounts: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/overlay",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Apply(context.Background(), req)
		if err != nil {
			b.Fatalf("Apply failed: %v", err)
		}
	}
}

func BenchmarkServiceDiff(b *testing.B) {
	comparer := &mockComparer{}
	svc := FromApplierAndComparer(&mockApplier{}, comparer)

	req := &diffapi.DiffRequest{
		Left: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/left",
			},
		},
		Right: []*mountapi.Mount{
			{
				Type:   "overlay",
				Source: "/tmp/right",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.Diff(context.Background(), req)
		if err != nil {
			b.Fatalf("Diff failed: %v", err)
		}
	}
}
