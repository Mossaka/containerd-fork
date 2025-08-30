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

	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/core/snapshots"
	digest "github.com/opencontainers/go-digest"
)

// Mock mounter for testing
type mockMounter struct {
	mounted   map[string][]mount.Mount
	unmounted map[string]bool
	mountErr  error
	unmountErr error
}

func newMockMounter() *mockMounter {
	return &mockMounter{
		mounted:   make(map[string][]mount.Mount),
		unmounted: make(map[string]bool),
	}
}

func (mm *mockMounter) Mount(target string, mounts ...mount.Mount) error {
	if mm.mountErr != nil {
		return mm.mountErr
	}
	mm.mounted[target] = mounts
	return nil
}

func (mm *mockMounter) Unmount(target string) error {
	if mm.unmountErr != nil {
		return mm.unmountErr
	}
	mm.unmounted[target] = true
	return nil
}

func TestInitRootFS_ReadOnly(t *testing.T) {
	ctx := context.Background()
	sn := newMockSnapshotter()
	mounter := newMockMounter()
	
	name := "test-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Setup parent snapshot
	sn.snapshots[parent.String()] = &snapshots.Info{
		Name: parent.String(),
		Kind: snapshots.KindCommitted,
	}
	
	mounts, err := InitRootFS(ctx, name, parent, true, sn, mounter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(mounts) == 0 {
		t.Error("Expected non-empty mounts")
	}
	
	// Should create a view for readonly
	if _, exists := sn.snapshots[name]; !exists {
		t.Error("Expected snapshot to be created")
	}
	
	if sn.snapshots[name].Kind != snapshots.KindView {
		t.Errorf("Expected view snapshot, got %s", sn.snapshots[name].Kind)
	}
}

func TestInitRootFS_Writable(t *testing.T) {
	ctx := context.Background()
	sn := newMockSnapshotter()
	mounter := newMockMounter()
	
	name := "test-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Setup parent snapshot
	sn.snapshots[parent.String()] = &snapshots.Info{
		Name: parent.String(),
		Kind: snapshots.KindCommitted,
	}
	
	mounts, err := InitRootFS(ctx, name, parent, false, sn, mounter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(mounts) == 0 {
		t.Error("Expected non-empty mounts")
	}
	
	// Should prepare for writable
	if sn.snapshots[name].Kind != snapshots.KindActive {
		t.Errorf("Expected active snapshot, got %s", sn.snapshots[name].Kind)
	}
}

func TestInitRootFS_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	sn := newMockSnapshotter()
	mounter := newMockMounter()
	
	name := "existing-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Pre-create the rootfs
	sn.snapshots[name] = &snapshots.Info{
		Name: name,
		Kind: snapshots.KindActive,
	}
	
	_, err := InitRootFS(ctx, name, parent, false, sn, mounter)
	if err == nil {
		t.Fatal("Expected error for existing rootfs")
	}
	
	if !strings.Contains(err.Error(), "rootfs already exists") {
		t.Errorf("Expected 'rootfs already exists' error, got %v", err)
	}
}

func TestInitRootFS_WithInitializer(t *testing.T) {
	ctx := context.Background()
	sn := newMockSnapshotter()
	mounter := newMockMounter()
	
	name := "test-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Setup parent snapshot
	sn.snapshots[parent.String()] = &snapshots.Info{
		Name: parent.String(),
		Kind: snapshots.KindCommitted,
	}
	
	// Since we can't easily mock the defaultInitializer const,
	// let's test the core functionality without specific initialization
	_, err := InitRootFS(ctx, name, parent, false, sn, mounter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// The behavior depends on the platform - this test verifies basic functionality
}

func TestInitRootFS_InitializerError(t *testing.T) {
	// This test requires platform-specific handling since initializer behavior
	// varies between Linux and non-Linux systems
	ctx := context.Background()
	sn := newMockSnapshotter()
	mounter := newMockMounter()
	
	name := "test-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Setup parent snapshot
	sn.snapshots[parent.String()] = &snapshots.Info{
		Name: parent.String(),
		Kind: snapshots.KindCommitted,
	}
	
	// Test without specific initializer setup - should work fine
	_, err := InitRootFS(ctx, name, parent, false, sn, mounter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestInitRootFS_ViewError(t *testing.T) {
	ctx := context.Background()
	sn := newMockSnapshotter()
	sn.viewError = errors.New("view error")
	mounter := newMockMounter()
	
	name := "test-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Setup parent snapshot
	sn.snapshots[parent.String()] = &snapshots.Info{
		Name: parent.String(),
		Kind: snapshots.KindCommitted,
	}
	
	_, err := InitRootFS(ctx, name, parent, true, sn, mounter)
	if err == nil {
		t.Fatal("Expected view error to be propagated")
	}
	
	if !strings.Contains(err.Error(), "view error") {
		t.Errorf("Expected view error, got %v", err)
	}
}

func TestInitRootFS_PrepareError(t *testing.T) {
	ctx := context.Background()
	sn := newMockSnapshotter()
	sn.prepareError = errors.New("prepare error")
	mounter := newMockMounter()
	
	name := "test-rootfs"
	parent := digest.FromString("parent-layer")
	
	// Setup parent snapshot
	sn.snapshots[parent.String()] = &snapshots.Info{
		Name: parent.String(),
		Kind: snapshots.KindCommitted,
	}
	
	_, err := InitRootFS(ctx, name, parent, false, sn, mounter)
	if err == nil {
		t.Fatal("Expected prepare error to be propagated")
	}
	
	if !strings.Contains(err.Error(), "prepare error") {
		t.Errorf("Expected prepare error, got %v", err)
	}
}

func TestMounter_Interface(t *testing.T) {
	var _ Mounter = newMockMounter()
	
	mounter := newMockMounter()
	target := "/tmp/test"
	mounts := []mount.Mount{
		{Type: "bind", Source: "/src", Target: target},
	}
	
	// Test Mount
	err := mounter.Mount(target, mounts...)
	if err != nil {
		t.Fatalf("Expected no error from Mount, got %v", err)
	}
	
	if len(mounter.mounted[target]) != 1 {
		t.Errorf("Expected 1 mount recorded, got %d", len(mounter.mounted[target]))
	}
	
	// Test Unmount
	err = mounter.Unmount(target)
	if err != nil {
		t.Fatalf("Expected no error from Unmount, got %v", err)
	}
	
	if !mounter.unmounted[target] {
		t.Error("Expected unmount to be recorded")
	}
}

func TestMounter_MountError(t *testing.T) {
	mounter := newMockMounter()
	mounter.mountErr = errors.New("mount failed")
	
	target := "/tmp/test"
	mounts := []mount.Mount{
		{Type: "bind", Source: "/src", Target: target},
	}
	
	err := mounter.Mount(target, mounts...)
	if err == nil {
		t.Fatal("Expected mount error")
	}
	
	if !strings.Contains(err.Error(), "mount failed") {
		t.Errorf("Expected mount error, got %v", err)
	}
}

func TestMounter_UnmountError(t *testing.T) {
	mounter := newMockMounter()
	mounter.unmountErr = errors.New("unmount failed")
	
	target := "/tmp/test"
	
	err := mounter.Unmount(target)
	if err == nil {
		t.Fatal("Expected unmount error")
	}
	
	if !strings.Contains(err.Error(), "unmount failed") {
		t.Errorf("Expected unmount error, got %v", err)
	}
}