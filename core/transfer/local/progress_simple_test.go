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

package local

import (
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestNewProgressTracker_Simple(t *testing.T) {
	root := "test-root"
	state := "downloading"
	
	pt := NewProgressTracker(root, state)
	
	if pt.root != root {
		t.Fatalf("Expected root %s, got %s", root, pt.root)
	}
	
	if pt.transferState != state {
		t.Fatalf("Expected transferState %s, got %s", state, pt.transferState)
	}
	
	if pt.added == nil {
		t.Fatal("Expected added channel to be initialized")
	}
	
	if pt.waitC == nil {
		t.Fatal("Expected waitC channel to be initialized")
	}
	
	if pt.parents == nil {
		t.Fatal("Expected parents map to be initialized")
	}
}

func TestProgressTracker_Add_Simple(t *testing.T) {
	pt := NewProgressTracker("test", "downloading")
	
	desc := ocispec.Descriptor{
		Digest:    digest.FromString("test-content"),
		Size:      123,
		MediaType: "application/vnd.docker.distribution.manifest.v2+json",
	}
	
	// Add should not block or panic
	pt.Add(desc)
	
	// Adding the same descriptor again should not cause issues
	pt.Add(desc)
}

func TestProgressTracker_AddChildren_Simple(t *testing.T) {
	pt := NewProgressTracker("test", "downloading")
	
	parentDesc := ocispec.Descriptor{
		Digest:    digest.FromString("parent-content"),
		Size:      456,
		MediaType: "application/vnd.docker.distribution.manifest.v2+json",
	}
	
	children := []ocispec.Descriptor{
		{
			Digest:    digest.FromString("child1"),
			Size:      100,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		},
		{
			Digest:    digest.FromString("child2"),
			Size:      200,
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
		},
	}
	
	// AddChildren should not panic
	pt.AddChildren(parentDesc, children)
	
	// Verify children were stored
	pt.parentL.Lock()
	storedChildren, exists := pt.parents[parentDesc.Digest]
	pt.parentL.Unlock()
	
	if !exists {
		t.Fatal("Expected parent-child relationship to be stored")
	}
	
	if len(storedChildren) != len(children) {
		t.Fatalf("Expected %d children, got %d", len(children), len(storedChildren))
	}
	
	for i, child := range storedChildren {
		if child.Digest != children[i].Digest {
			t.Fatalf("Expected child %d digest %s, got %s", i, children[i].Digest, child.Digest)
		}
	}
}

func TestProgressTracker_MarkExists_Simple(t *testing.T) {
	pt := NewProgressTracker("test", "downloading")
	
	desc := ocispec.Descriptor{
		Digest:    digest.FromString("existing-content"),
		Size:      789,
		MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
	}
	
	// MarkExists should not block or panic
	pt.MarkExists(desc)
}

func TestProgressTracker_Wait_Simple(t *testing.T) {
	pt := NewProgressTracker("test", "downloading")
	
	// Start a goroutine to close waitC after a delay
	go func() {
		time.Sleep(time.Millisecond * 10)
		close(pt.waitC)
	}()
	
	// Wait should block until waitC is closed
	start := time.Now()
	pt.Wait()
	elapsed := time.Since(start)
	
	if elapsed < time.Millisecond*5 {
		t.Fatal("Wait returned too quickly, expected to block")
	}
}

func TestJobState_Constants_Simple(t *testing.T) {
	// Test that job state constants have expected values
	if jobAdded != 0 {
		t.Fatalf("Expected jobAdded to be 0, got %d", jobAdded)
	}
	
	if jobInProgress != 1 {
		t.Fatalf("Expected jobInProgress to be 1, got %d", jobInProgress)
	}
	
	if jobComplete != 2 {
		t.Fatalf("Expected jobComplete to be 2, got %d", jobComplete)
	}
}