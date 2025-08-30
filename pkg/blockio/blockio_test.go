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

package blockio

import (
	"testing"
)

// Cross-platform benchmark tests to ensure performance
func BenchmarkIsEnabled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsEnabled()
	}
}

func BenchmarkSetConfigEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SetConfig("")
	}
}

func BenchmarkClassNameToLinuxOCI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ClassNameToLinuxOCI("default")
	}
}

func BenchmarkContainerClassFromAnnotations(b *testing.B) {
	containerAnnotations := map[string]string{
		"io.kubernetes.container.name": "test-container",
	}
	podAnnotations := map[string]string{
		"io.kubernetes.pod.name": "test-pod",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ContainerClassFromAnnotations("test-container", containerAnnotations, podAnnotations)
	}
}

// Basic functionality tests that work on all platforms
func TestBasicFunctionality(t *testing.T) {
	// Test IsEnabled doesn't panic
	enabled := IsEnabled()
	t.Logf("IsEnabled() returned: %v", enabled)
	
	// Test SetConfig with empty path doesn't panic
	err := SetConfig("")
	if err != nil {
		t.Logf("SetConfig(\"\") returned error: %v", err)
	}
	
	// Test ClassNameToLinuxOCI doesn't panic
	result, err := ClassNameToLinuxOCI("test-class")
	t.Logf("ClassNameToLinuxOCI returned result=%v, err=%v", result, err)
	
	// Test ContainerClassFromAnnotations doesn't panic
	className, err := ContainerClassFromAnnotations("test", nil, nil)
	t.Logf("ContainerClassFromAnnotations returned className=%q, err=%v", className, err)
}