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

package nvidia

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/containerd/containerd/v2/core/containers"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestAllCaps(t *testing.T) {
	caps := AllCaps()
	expected := []Capability{
		Compute,
		Compat32,
		Graphics,
		Utility,
		Video,
		Display,
	}

	if len(caps) != len(expected) {
		t.Fatalf("expected %d capabilities, got %d", len(expected), len(caps))
	}

	for i, cap := range caps {
		if cap != expected[i] {
			t.Errorf("expected capability %s at index %d, got %s", expected[i], i, cap)
		}
	}
}

func TestCapabilityConstants(t *testing.T) {
	tests := []struct {
		cap      Capability
		expected string
	}{
		{Compute, "compute"},
		{Compat32, "compat32"},
		{Graphics, "graphics"},
		{Utility, "utility"},
		{Video, "video"},
		{Display, "display"},
	}

	for _, test := range tests {
		if string(test.cap) != test.expected {
			t.Errorf("expected capability %s to have value %s, got %s", test.expected, test.expected, string(test.cap))
		}
	}
}

func TestWithDevices(t *testing.T) {
	c := &config{}
	opt := WithDevices(0, 1, 2)

	err := opt(c)
	if err != nil {
		t.Fatalf("WithDevices failed: %v", err)
	}

	expected := []string{"0", "1", "2"}
	if !reflect.DeepEqual(c.Devices, expected) {
		t.Errorf("expected devices %v, got %v", expected, c.Devices)
	}
}

func TestWithDeviceUUIDs(t *testing.T) {
	c := &config{}
	uuids := []string{"GPU-12345", "GPU-67890"}
	opt := WithDeviceUUIDs(uuids...)

	err := opt(c)
	if err != nil {
		t.Fatalf("WithDeviceUUIDs failed: %v", err)
	}

	if !reflect.DeepEqual(c.Devices, uuids) {
		t.Errorf("expected devices %v, got %v", uuids, c.Devices)
	}
}

func TestWithAllDevices(t *testing.T) {
	c := &config{}

	err := WithAllDevices(c)
	if err != nil {
		t.Fatalf("WithAllDevices failed: %v", err)
	}

	expected := []string{"all"}
	if !reflect.DeepEqual(c.Devices, expected) {
		t.Errorf("expected devices %v, got %v", expected, c.Devices)
	}
}

func TestWithAllCapabilities(t *testing.T) {
	c := &config{}

	err := WithAllCapabilities(c)
	if err != nil {
		t.Fatalf("WithAllCapabilities failed: %v", err)
	}

	expected := AllCaps()
	if !reflect.DeepEqual(c.Capabilities, expected) {
		t.Errorf("expected capabilities %v, got %v", expected, c.Capabilities)
	}
}

func TestWithCapabilities(t *testing.T) {
	c := &config{}
	caps := []Capability{Compute, Graphics}
	opt := WithCapabilities(caps...)

	err := opt(c)
	if err != nil {
		t.Fatalf("WithCapabilities failed: %v", err)
	}

	if !reflect.DeepEqual(c.Capabilities, caps) {
		t.Errorf("expected capabilities %v, got %v", caps, c.Capabilities)
	}
}

func TestWithRequiredCUDAVersion(t *testing.T) {
	c := &config{}
	opt := WithRequiredCUDAVersion(11, 4)

	err := opt(c)
	if err != nil {
		t.Fatalf("WithRequiredCUDAVersion failed: %v", err)
	}

	expected := []string{"cuda>=11.4"}
	if !reflect.DeepEqual(c.Requirements, expected) {
		t.Errorf("expected requirements %v, got %v", expected, c.Requirements)
	}
}

func TestWithOCIHookPath(t *testing.T) {
	c := &config{}
	path := "/usr/bin/containerd"
	opt := WithOCIHookPath(path)

	err := opt(c)
	if err != nil {
		t.Fatalf("WithOCIHookPath failed: %v", err)
	}

	if c.OCIHookPath != path {
		t.Errorf("expected hook path %s, got %s", path, c.OCIHookPath)
	}
}

func TestWithLookupOCIHookPath(t *testing.T) {
	// Test with a binary that should exist on the system
	c := &config{}
	opt := WithLookupOCIHookPath("sh")

	err := opt(c)
	if err != nil {
		t.Fatalf("WithLookupOCIHookPath failed: %v", err)
	}

	if c.OCIHookPath == "" {
		t.Error("expected hook path to be set")
	}

	if !strings.Contains(c.OCIHookPath, "sh") {
		t.Errorf("expected hook path to contain 'sh', got %s", c.OCIHookPath)
	}
}

func TestWithLookupOCIHookPath_NotFound(t *testing.T) {
	c := &config{}
	opt := WithLookupOCIHookPath("nonexistent-binary-12345")

	err := opt(c)
	if err == nil {
		t.Fatal("expected WithLookupOCIHookPath to fail with nonexistent binary")
	}
}

func TestWithNoCgroups(t *testing.T) {
	c := &config{}

	err := WithNoCgroups(c)
	if err != nil {
		t.Fatalf("WithNoCgroups failed: %v", err)
	}

	if !c.NoCgroups {
		t.Error("expected NoCgroups to be true")
	}
}

func TestConfigArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   *config
		contains []string
	}{
		{
			name:   "empty config",
			config: &config{},
			contains: []string{
				"configure",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with devices",
			config: &config{
				Devices: []string{"0", "1"},
			},
			contains: []string{
				"configure",
				"--device=0,1",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with capabilities",
			config: &config{
				Capabilities: []Capability{Compute, Graphics},
			},
			contains: []string{
				"configure",
				"--compute",
				"--graphics",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with load kmods",
			config: &config{
				LoadKmods: true,
			},
			contains: []string{
				"--load-kmods",
				"configure",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with ldcache",
			config: &config{
				LDCache: "/tmp/ldcache",
			},
			contains: []string{
				"--ldcache=/tmp/ldcache",
				"configure",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with ldconfig",
			config: &config{
				LDConfig: "/usr/bin/ldconfig",
			},
			contains: []string{
				"configure",
				"--ldconfig=/usr/bin/ldconfig",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with requirements",
			config: &config{
				Requirements: []string{"cuda>=11.4", "arch=x86_64"},
			},
			contains: []string{
				"configure",
				"--require=cuda>=11.4",
				"--require=arch=x86_64",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "with no cgroups",
			config: &config{
				NoCgroups: true,
			},
			contains: []string{
				"configure",
				"--no-cgroups",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
		{
			name: "complex config",
			config: &config{
				Devices:      []string{"0", "1"},
				Capabilities: []Capability{Compute, Video},
				LoadKmods:    true,
				LDCache:      "/tmp/ldcache",
				LDConfig:     "/usr/bin/ldconfig",
				Requirements: []string{"cuda>=11.4"},
				NoCgroups:    true,
			},
			contains: []string{
				"--load-kmods",
				"--ldcache=/tmp/ldcache",
				"configure",
				"--device=0,1",
				"--compute",
				"--video",
				"--ldconfig=/usr/bin/ldconfig",
				"--require=cuda>=11.4",
				"--no-cgroups",
				"--pid={{pid}}",
				"{{rootfs}}",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.config.args()

			for _, expected := range test.contains {
				found := false
				for _, arg := range args {
					if arg == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected arg %s to be present in %v", expected, args)
				}
			}
		})
	}
}

// TestWithGPUs tests the main WithGPUs function with mocked environment
func TestWithGPUs(t *testing.T) {
	// Create a temporary directory and mock binaries for testing
	tmpDir := t.TempDir()

	// Create mock containerd binary
	containerdPath := filepath.Join(tmpDir, "containerd")
	err := os.WriteFile(containerdPath, []byte("#!/bin/sh\necho containerd"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create mock nvidia-container-cli binary
	nvidiaPath := filepath.Join(tmpDir, NvidiaCLI)
	err = os.WriteFile(nvidiaPath, []byte("#!/bin/sh\necho nvidia-cli"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily modify PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	tests := []struct {
		name string
		opts []Opts
	}{
		{
			name: "basic gpu support",
			opts: []Opts{WithDevices(0)},
		},
		{
			name: "all devices",
			opts: []Opts{WithAllDevices},
		},
		{
			name: "with capabilities",
			opts: []Opts{WithCapabilities(Compute, Graphics)},
		},
		{
			name: "complex setup",
			opts: []Opts{
				WithDevices(0, 1),
				WithCapabilities(Compute, Video),
				WithRequiredCUDAVersion(11, 4),
				WithOCIHookPath(containerdPath),
				WithNoCgroups,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := &specs.Spec{}
			container := &containers.Container{}

			opt := WithGPUs(test.opts...)
			err := opt(context.Background(), nil, container, spec)
			if err != nil {
				t.Fatalf("WithGPUs failed: %v", err)
			}

			// Verify hooks were added
			if spec.Hooks == nil {
				t.Fatal("expected hooks to be set")
			}

			if len(spec.Hooks.CreateRuntime) == 0 {
				t.Fatal("expected CreateRuntime hooks to be set")
			}

			hook := spec.Hooks.CreateRuntime[0]
			if hook.Path == "" {
				t.Error("expected hook path to be set")
			}

			// Verify containerd and nvidia-container-cli are in args
			foundContainerd := false
			foundNvidia := false
			for _, arg := range hook.Args {
				if strings.Contains(arg, "containerd") {
					foundContainerd = true
				}
				if strings.Contains(arg, NvidiaCLI) {
					foundNvidia = true
				}
			}

			if !foundContainerd {
				t.Error("expected containerd to be in hook args")
			}
			if !foundNvidia {
				t.Error("expected nvidia-container-cli to be in hook args")
			}
		})
	}
}

func TestWithGPUs_MissingNvidiaCLI(t *testing.T) {
	// Create a temporary directory without nvidia-container-cli
	tmpDir := t.TempDir()

	// Create mock containerd binary
	containerdPath := filepath.Join(tmpDir, "containerd")
	err := os.WriteFile(containerdPath, []byte("#!/bin/sh\necho containerd"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily modify PATH to only include tmpDir (no nvidia-container-cli)
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir)
	defer os.Setenv("PATH", oldPath)

	spec := &specs.Spec{}
	container := &containers.Container{}

	opt := WithGPUs()
	err = opt(context.Background(), nil, container, spec)
	if err == nil {
		t.Fatal("expected WithGPUs to fail when nvidia-container-cli is not found")
	}
}

func TestWithGPUs_MissingContainerd(t *testing.T) {
	// Create a temporary directory with nvidia-container-cli but no containerd
	tmpDir := t.TempDir()

	// Create mock nvidia-container-cli binary
	nvidiaPath := filepath.Join(tmpDir, NvidiaCLI)
	err := os.WriteFile(nvidiaPath, []byte("#!/bin/sh\necho nvidia-cli"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily modify PATH to only include tmpDir (no containerd)
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir)
	defer os.Setenv("PATH", oldPath)

	spec := &specs.Spec{}
	container := &containers.Container{}

	opt := WithGPUs()
	err = opt(context.Background(), nil, container, spec)
	if err == nil {
		t.Fatal("expected WithGPUs to fail when containerd is not found")
	}
}

func TestWithGPUs_ExplicitHookPath(t *testing.T) {
	// Create a temporary directory and mock nvidia-container-cli
	tmpDir := t.TempDir()

	// Create mock nvidia-container-cli binary
	nvidiaPath := filepath.Join(tmpDir, NvidiaCLI)
	err := os.WriteFile(nvidiaPath, []byte("#!/bin/sh\necho nvidia-cli"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Temporarily modify PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	explicitPath := "/custom/containerd/path"
	spec := &specs.Spec{}
	container := &containers.Container{}

	opt := WithGPUs(WithOCIHookPath(explicitPath))
	err = opt(context.Background(), nil, container, spec)
	if err != nil {
		t.Fatalf("WithGPUs failed: %v", err)
	}

	// Verify the explicit path was used
	if spec.Hooks == nil || len(spec.Hooks.CreateRuntime) == 0 {
		t.Fatal("expected hooks to be set")
	}

	hook := spec.Hooks.CreateRuntime[0]
	if hook.Path != explicitPath {
		t.Errorf("expected hook path %s, got %s", explicitPath, hook.Path)
	}
}
