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

package command

import (
	"strings"
	"testing"
)

func TestOciHook(t *testing.T) {
	if ociHook == nil {
		t.Fatal("ociHook is nil")
	}

	if ociHook.Name != "oci-hook" {
		t.Errorf("expected oci-hook command name 'oci-hook', got %q", ociHook.Name)
	}

	if !strings.Contains(ociHook.Usage, "OCI runtime hooks") {
		t.Errorf("expected usage to contain 'OCI runtime hooks', got %q", ociHook.Usage)
	}

	if ociHook.Action == nil {
		t.Error("expected oci-hook command to have an action")
	}
}

func TestHookSpecStruct(t *testing.T) {
	// Test that hookSpec struct has expected fields
	spec := &hookSpec{}
	if spec.Root != nil {
		t.Error("expected hookSpec.Root to be nil by default")
	}

	// This tests the struct definition exists and is accessible
	_ = hookSpec{
		Root: nil,
	}
}

func TestTemplateContextStruct(t *testing.T) {
	// Test that templateContext struct has expected fields
	ctx := &templateContext{}
	if ctx.state != nil {
		t.Error("expected templateContext.state to be nil by default")
	}
	if ctx.root != "" {
		t.Error("expected templateContext.root to be empty by default")
	}
	if ctx.funcs != nil {
		t.Error("expected templateContext.funcs to be nil by default")
	}
}
