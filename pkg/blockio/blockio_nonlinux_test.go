//go:build !linux

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

func TestIsEnabledNonLinux(t *testing.T) {
	// On non-Linux platforms, IsEnabled should always return false
	if IsEnabled() {
		t.Error("IsEnabled() should always return false on non-Linux platforms")
	}
}

func TestSetConfigNonLinux(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{
			name: "empty path",
			path: "",
		},
		{
			name: "valid path",
			path: "/some/config/path.conf",
		},
		{
			name: "invalid path",
			path: "/nonexistent/path.conf",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// SetConfig should always return nil on non-Linux platforms
			err := SetConfig(tc.path)
			if err != nil {
				t.Errorf("SetConfig(%q) should return nil on non-Linux platforms, got: %v", tc.path, err)
			}

			// IsEnabled should still be false after SetConfig
			if IsEnabled() {
				t.Error("IsEnabled() should remain false after SetConfig on non-Linux platforms")
			}
		})
	}
}

func TestClassNameToLinuxOCINonLinux(t *testing.T) {
	testCases := []struct {
		name      string
		className string
	}{
		{
			name:      "empty class name",
			className: "",
		},
		{
			name:      "valid class name",
			className: "default",
		},
		{
			name:      "invalid class name",
			className: "nonexistent-class",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ClassNameToLinuxOCI(tc.className)
			
			// On non-Linux platforms, should always return nil, nil
			if result != nil || err != nil {
				t.Errorf("ClassNameToLinuxOCI(%q) should return nil, nil on non-Linux platforms, got result=%v, err=%v", 
					tc.className, result, err)
			}
		})
	}
}

func TestContainerClassFromAnnotationsNonLinux(t *testing.T) {
	testCases := []struct {
		name                  string
		containerName         string
		containerAnnotations  map[string]string
		podAnnotations        map[string]string
	}{
		{
			name:          "empty inputs",
			containerName: "",
			containerAnnotations: nil,
			podAnnotations: nil,
		},
		{
			name:          "with container name",
			containerName: "test-container",
			containerAnnotations: map[string]string{},
			podAnnotations: map[string]string{},
		},
		{
			name:          "with annotations",
			containerName: "test-container",
			containerAnnotations: map[string]string{
				"io.kubernetes.container.name": "test-container",
				"custom.annotation": "value",
			},
			podAnnotations: map[string]string{
				"io.kubernetes.pod.name": "test-pod",
				"blockio.class": "high-priority",
			},
		},
		{
			name:          "with blockio annotations",
			containerName: "test-container",
			containerAnnotations: map[string]string{
				"intel.com/blockio": "performance",
			},
			podAnnotations: map[string]string{
				"intel.com/blockio": "default",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			className, err := ContainerClassFromAnnotations(
				tc.containerName,
				tc.containerAnnotations,
				tc.podAnnotations,
			)
			
			// On non-Linux platforms, should always return empty string, nil
			if className != "" || err != nil {
				t.Errorf("ContainerClassFromAnnotations should return empty string, nil on non-Linux platforms, got className=%q, err=%v", 
					className, err)
			}
		})
	}
}

func TestContainerClassFromAnnotationsNilMapsNonLinux(t *testing.T) {
	// Test with nil annotation maps to ensure no panic on non-Linux
	className, err := ContainerClassFromAnnotations("test-container", nil, nil)
	
	// Should return empty string and nil error on non-Linux
	if className != "" || err != nil {
		t.Errorf("ContainerClassFromAnnotations with nil maps should return empty string, nil on non-Linux, got className=%q, err=%v", 
			className, err)
	}
}

// Test edge cases with special characters in annotations on non-Linux
func TestContainerClassFromAnnotationsSpecialCharsNonLinux(t *testing.T) {
	testCases := []struct {
		name                 string
		containerName        string
		containerAnnotations map[string]string
		podAnnotations       map[string]string
	}{
		{
			name:          "unicode characters",
			containerName: "test-container-ü",
			containerAnnotations: map[string]string{
				"annotation/with/slashes": "value",
				"annotation.with.dots": "value",
			},
			podAnnotations: map[string]string{
				"pod-annotation_with_underscores": "value",
			},
		},
		{
			name:          "empty values",
			containerName: "test",
			containerAnnotations: map[string]string{
				"empty-value": "",
			},
			podAnnotations: map[string]string{
				"another-empty": "",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic with special characters and return expected values
			className, err := ContainerClassFromAnnotations(
				tc.containerName,
				tc.containerAnnotations,
				tc.podAnnotations,
			)
			
			// On non-Linux, should always return empty string, nil
			if className != "" || err != nil {
				t.Errorf("ContainerClassFromAnnotations with special chars should return empty string, nil on non-Linux, got className=%q, err=%v", 
					className, err)
			}
		})
	}
}

// Benchmark tests for non-Linux platforms
func BenchmarkIsEnabledNonLinux(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsEnabled()
	}
}

func BenchmarkSetConfigEmptyNonLinux(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SetConfig("")
	}
}

func BenchmarkClassNameToLinuxOCINonLinux(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ClassNameToLinuxOCI("default")
	}
}

func BenchmarkContainerClassFromAnnotationsNonLinux(b *testing.B) {
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