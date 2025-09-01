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

package rootfs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/containerd/containerd/v2/core/diff"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/core/snapshots"
	"github.com/containerd/errdefs"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Mock comparer for testing
type mockComparer struct {
	compareResult ocispec.Descriptor
	compareError  error
	compareCalls  []compareCall
}

type compareCall struct {
	lower []mount.Mount
	upper []mount.Mount
	opts  []diff.Opt
}

func (mc *mockComparer) Compare(ctx context.Context, lower, upper []mount.Mount, opts ...diff.Opt) (ocispec.Descriptor, error) {
	mc.compareCalls = append(mc.compareCalls, compareCall{
		lower: lower,
		upper: upper,
		opts:  opts,
	})
	
	if mc.compareError != nil {
		return ocispec.Descriptor{}, mc.compareError
	}
	
	if mc.compareResult.Digest != "" {
		return mc.compareResult, nil
	}
	
	// Default result
	return ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar",
		Digest:    digest.FromString("test-diff"),
		Size:      100,
	}, nil
}

// Extended mock snapshotter for diff tests
type diffMockSnapshotter struct {
	*mockSnapshotter
	parentViewError bool
	upperViewError  bool
}

func newDiffMockSnapshotter() *diffMockSnapshotter {
	return &diffMockSnapshotter{
		mockSnapshotter: newMockSnapshotter(),
	}
}

func (dms *diffMockSnapshotter) View(ctx context.Context, key, parent string, opts ...snapshots.Opt) ([]mount.Mount, error) {
	if dms.parentViewError && strings.Contains(key, "parent-view") {
		return nil, errors.New("parent view error")
	}
	if dms.upperViewError && strings.Contains(key, "view") && !strings.Contains(key, "parent-view") {
		return nil, errors.New("upper view error")
	}
	return dms.mockSnapshotter.View(ctx, key, parent, opts...)
}

func TestCreateDiff_ActiveSnapshot(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	comparer := &mockComparer{}
	
	snapshotID := "test-snapshot"
	parentID := "parent-snapshot"
	
	// Setup active snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindActive,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	desc, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if desc.MediaType != "application/vnd.oci.image.layer.v1.tar" {
		t.Errorf("Expected tar media type, got %s", desc.MediaType)
	}
	
	if len(comparer.compareCalls) != 1 {
		t.Fatalf("Expected 1 compare call, got %d", len(comparer.compareCalls))
	}
}

func TestCreateDiff_CommittedSnapshot(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	comparer := &mockComparer{}
	
	snapshotID := "committed-snapshot"
	parentID := "parent-snapshot"
	
	// Setup committed snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindCommitted,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	desc, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if desc.Digest == "" {
		t.Error("Expected non-empty digest")
	}
	
	if len(comparer.compareCalls) != 1 {
		t.Fatalf("Expected 1 compare call, got %d", len(comparer.compareCalls))
	}
}

func TestCreateDiff_SnapshotNotFound(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	comparer := &mockComparer{}
	
	snapshotID := "nonexistent-snapshot"
	
	_, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err == nil {
		t.Fatal("Expected error for nonexistent snapshot")
	}
	
	if !errdefs.IsNotFound(err) {
		t.Errorf("Expected NotFound error, got %v", err)
	}
}

func TestCreateDiff_ParentViewError(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	sn.parentViewError = true
	comparer := &mockComparer{}
	
	snapshotID := "test-snapshot"
	parentID := "parent-snapshot"
	
	// Setup snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindActive,
	}
	
	_, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err == nil {
		t.Fatal("Expected error for parent view failure")
	}
	
	if !strings.Contains(err.Error(), "parent view error") {
		t.Errorf("Expected parent view error, got %v", err)
	}
}

func TestCreateDiff_UpperMountsError_Active(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	sn.mountsError = errors.New("mounts error")
	comparer := &mockComparer{}
	
	snapshotID := "test-snapshot"
	parentID := "parent-snapshot"
	
	// Setup active snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindActive,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	_, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err == nil {
		t.Fatal("Expected error for mounts failure")
	}
	
	if !strings.Contains(err.Error(), "mounts error") {
		t.Errorf("Expected mounts error, got %v", err)
	}
}

func TestCreateDiff_UpperViewError_Committed(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	sn.upperViewError = true
	comparer := &mockComparer{}
	
	snapshotID := "committed-snapshot"
	parentID := "parent-snapshot"
	
	// Setup committed snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindCommitted,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	_, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err == nil {
		t.Fatal("Expected error for upper view failure")
	}
	
	if !strings.Contains(err.Error(), "upper view error") {
		t.Errorf("Expected upper view error, got %v", err)
	}
}

func TestCreateDiff_CompareError(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	comparer := &mockComparer{
		compareError: errors.New("compare failed"),
	}
	
	snapshotID := "test-snapshot"
	parentID := "parent-snapshot"
	
	// Setup snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindActive,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	_, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err == nil {
		t.Fatal("Expected compare error to be propagated")
	}
	
	if !strings.Contains(err.Error(), "compare failed") {
		t.Errorf("Expected compare error, got %v", err)
	}
}

func TestCreateDiff_WithDiffOptions(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	comparer := &mockComparer{}
	
	snapshotID := "test-snapshot"
	parentID := "parent-snapshot"
	
	// Setup snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindActive,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	opts := []diff.Opt{}
	
	_, err := CreateDiff(ctx, snapshotID, sn, comparer, opts...)
	if err != nil {
		t.Fatalf("Expected no error with options, got %v", err)
	}
	
	if len(comparer.compareCalls) != 1 {
		t.Fatalf("Expected 1 compare call, got %d", len(comparer.compareCalls))
	}
	
	if len(comparer.compareCalls[0].opts) != 0 {
		t.Errorf("Expected 0 diff options, got %d", len(comparer.compareCalls[0].opts))
	}
}

func TestCreateDiff_CustomResult(t *testing.T) {
	ctx := context.Background()
	sn := newDiffMockSnapshotter()
	
	expectedDigest := digest.FromString("custom-result")
	comparer := &mockComparer{
		compareResult: ocispec.Descriptor{
			MediaType: "application/vnd.custom.layer",
			Digest:    expectedDigest,
			Size:      200,
		},
	}
	
	snapshotID := "test-snapshot"
	parentID := "parent-snapshot"
	
	// Setup snapshot
	sn.snapshots[snapshotID] = &snapshots.Info{
		Name:   snapshotID,
		Parent: parentID,
		Kind:   snapshots.KindActive,
	}
	sn.snapshots[parentID] = &snapshots.Info{
		Name: parentID,
		Kind: snapshots.KindCommitted,
	}
	
	desc, err := CreateDiff(ctx, snapshotID, sn, comparer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if desc.MediaType != "application/vnd.custom.layer" {
		t.Errorf("Expected custom media type, got %s", desc.MediaType)
	}
	
	if desc.Digest != expectedDigest {
		t.Errorf("Expected digest %s, got %s", expectedDigest, desc.Digest)
	}
	
	if desc.Size != 200 {
		t.Errorf("Expected size 200, got %d", desc.Size)
	}
}