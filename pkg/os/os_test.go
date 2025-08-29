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

package os

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRealOS_MkdirAll(t *testing.T) {
	realOS := RealOS{}

	// Test with temporary directory
	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testPath := filepath.Join(tmpDir, "nested", "dir", "structure")

	// Test MkdirAll
	err = realOS.MkdirAll(testPath, 0755)
	if err != nil {
		t.Errorf("MkdirAll failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(testPath)
	if err != nil {
		t.Errorf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Path is not a directory")
	}
}

func TestRealOS_RemoveAll(t *testing.T) {
	realOS := RealOS{}

	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	testPath := filepath.Join(tmpDir, "nested", "dir")
	err = os.MkdirAll(testPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	// Test RemoveAll
	err = realOS.RemoveAll(tmpDir)
	if err != nil {
		t.Errorf("RemoveAll failed: %v", err)
	}

	// Verify directory was removed
	_, err = os.Stat(tmpDir)
	if !os.IsNotExist(err) {
		t.Errorf("Directory should have been removed")
	}
}

func TestRealOS_Stat(t *testing.T) {
	realOS := RealOS{}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Test Stat
	info, err := realOS.Stat(tmpFile.Name())
	if err != nil {
		t.Errorf("Stat failed: %v", err)
	}
	if info.IsDir() {
		t.Errorf("File should not be directory")
	}

	// Test Stat on non-existent file
	_, err = realOS.Stat("non-existent-file")
	if !os.IsNotExist(err) {
		t.Errorf("Expected file not found error")
	}
}

func TestRealOS_FollowSymlinkInScope(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlinks behave differently on Windows")
	}

	realOS := RealOS{}

	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create target file
	targetFile := filepath.Join(tmpDir, "target.txt")
	err = os.WriteFile(targetFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create symlink
	linkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink("target.txt", linkFile)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test FollowSymlinkInScope
	result, err := realOS.FollowSymlinkInScope(linkFile, tmpDir)
	if err != nil {
		t.Errorf("FollowSymlinkInScope failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "target.txt")
	if result != expectedPath {
		t.Errorf("Expected %s, got %s", expectedPath, result)
	}
}

func TestRealOS_CopyFile(t *testing.T) {
	realOS := RealOS{}

	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	testData := "test data content"
	err = os.WriteFile(srcFile, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test CopyFile
	destFile := filepath.Join(tmpDir, "dest.txt")
	err = realOS.CopyFile(srcFile, destFile, 0644)
	if err != nil {
		t.Errorf("CopyFile failed: %v", err)
	}

	// Verify file was copied
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Errorf("Failed to read destination file: %v", err)
	}
	if string(data) != testData {
		t.Errorf("Expected %q, got %q", testData, string(data))
	}

	// Test CopyFile with non-existent source
	err = realOS.CopyFile("non-existent-file", destFile, 0644)
	if err == nil {
		t.Errorf("Expected error for non-existent source file")
	}
}

func TestRealOS_WriteFile(t *testing.T) {
	realOS := RealOS{}

	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test content")

	// Test WriteFile
	err = realOS.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Errorf("WriteFile failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	if string(data) != string(testData) {
		t.Errorf("Expected %q, got %q", testData, data)
	}
}

func TestRealOS_Hostname(t *testing.T) {
	realOS := RealOS{}

	// Test Hostname
	hostname, err := realOS.Hostname()
	if err != nil {
		t.Errorf("Hostname failed: %v", err)
	}
	if hostname == "" {
		t.Errorf("Hostname should not be empty")
	}
}

func TestRealOS_ResolveSymbolicLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink behavior differs on Windows")
	}

	realOS := RealOS{}

	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with regular file (no symlink)
	regularFile := filepath.Join(tmpDir, "regular.txt")
	err = os.WriteFile(regularFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	resolved, err := realOS.ResolveSymbolicLink(regularFile)
	if err != nil {
		t.Errorf("ResolveSymbolicLink failed for regular file: %v", err)
	}
	if resolved != regularFile {
		t.Errorf("Expected %s, got %s", regularFile, resolved)
	}

	// Test with symlink
	targetFile := filepath.Join(tmpDir, "target.txt")
	err = os.WriteFile(targetFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	linkFile := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink("target.txt", linkFile)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	resolved, err = realOS.ResolveSymbolicLink(linkFile)
	if err != nil {
		t.Errorf("ResolveSymbolicLink failed for symlink: %v", err)
	}
	if resolved != targetFile {
		t.Errorf("Expected %s, got %s", targetFile, resolved)
	}

	// Test with non-existent file
	_, err = realOS.ResolveSymbolicLink("non-existent-file")
	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}
}

func TestRealOS_MountOperations(t *testing.T) {
	realOS := RealOS{}

	// These are platform-specific operations that require root privileges
	// We'll test the function calls but expect permission errors in most cases

	// Test Mount (should fail with permission error in most cases)
	err := realOS.Mount("/dev/null", "/tmp/test-mount", "tmpfs", 0, "")
	if err == nil {
		// If it succeeded, we need to unmount
		realOS.Unmount("/tmp/test-mount")
	}
	// We don't fail the test here since this requires root privileges

	// Test LookupMount with a known path
	_, err = realOS.LookupMount("/")
	// This might work or might fail depending on the system
	// We just verify it doesn't panic
}

func TestOS_Interface(t *testing.T) {
	// Verify RealOS implements OS interface
	var _ OS = RealOS{}
}

func TestOS_InterfaceMethods(t *testing.T) {
	realOS := RealOS{}

	// Verify all interface methods are callable (test method signatures)
	tmpDir, err := os.MkdirTemp("", "containerd-os-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")

	// Test all methods exist and can be called
	_ = realOS.MkdirAll(tmpDir, 0755)
	_ = realOS.RemoveAll("/non-existent")
	_, _ = realOS.Stat(testFile)
	_, _ = realOS.ResolveSymbolicLink(testFile)
	_, _ = realOS.FollowSymlinkInScope(testFile, tmpDir)
	_ = realOS.CopyFile("/dev/null", testFile, 0644)
	_ = realOS.WriteFile(testFile, []byte("test"), 0644)
	_, _ = realOS.Hostname()
	_ = realOS.Mount("", "", "", 0, "")
	_ = realOS.Unmount("")
	_, _ = realOS.LookupMount("/")
}
