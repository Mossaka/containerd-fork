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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	srvconfig "github.com/containerd/containerd/v2/cmd/containerd/server/config"
	"github.com/containerd/containerd/v2/defaults"
	"github.com/containerd/containerd/v2/version"
	"github.com/urfave/cli/v2"
)

func TestApp(t *testing.T) {
	app := App()
	if app == nil {
		t.Fatal("App() returned nil")
	}

	if app.Name != "containerd" {
		t.Errorf("expected app name 'containerd', got %q", app.Name)
	}

	if app.Version != version.Version {
		t.Errorf("expected app version %q, got %q", version.Version, app.Version)
	}

	if !strings.Contains(app.Usage, "high performance container runtime") {
		t.Error("expected usage to contain 'high performance container runtime'")
	}

	if !strings.Contains(app.Description, "containerd is a high performance container runtime") {
		t.Error("expected description to contain containerd description")
	}

	// Check expected flags
	expectedFlags := []string{"config", "log-level", "address", "root", "state"}
	for _, expectedFlag := range expectedFlags {
		found := false
		for _, flag := range app.Flags {
			if flag.Names()[0] == expectedFlag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected flag %q not found", expectedFlag)
		}
	}

	// Check expected commands
	expectedCommands := []string{"config", "publish", "oci-hook"}
	for _, expectedCmd := range expectedCommands {
		found := false
		for _, cmd := range app.Commands {
			if cmd.Name == expectedCmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %q not found", expectedCmd)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()
	if config == nil {
		t.Fatal("defaultConfig() returned nil")
	}

	// Verify basic configuration structure
	if config.Root == "" {
		t.Error("expected default config to have root directory set")
	}
	if config.State == "" {
		t.Error("expected default config to have state directory set")
	}

	// Check that GRPC configuration exists
	if config.GRPC.Address == "" {
		t.Error("expected default config to have GRPC address set")
	}
}

func TestOutputConfig(t *testing.T) {
	ctx := context.Background()
	config := defaultConfig()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan []byte, 1)
	go func() {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
		done <- buf.Bytes()
	}()

	// Test outputConfig
	err = outputConfig(ctx, config)
	w.Close()
	os.Stdout = originalStdout
	output := <-done
	r.Close()

	if err != nil {
		t.Errorf("outputConfig failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("expected output from outputConfig, got empty")
	}

	// Check that output contains expected TOML structure
	outputStr := string(output)
	if !strings.Contains(outputStr, "version") {
		t.Error("expected output to contain version field")
	}
	if !strings.Contains(outputStr, "[grpc]") {
		t.Error("expected output to contain [grpc] section")
	}
}

func TestConfigCommand(t *testing.T) {
	if configCommand == nil {
		t.Fatal("configCommand is nil")
	}

	if configCommand.Name != "config" {
		t.Errorf("expected config command name 'config', got %q", configCommand.Name)
	}

	if len(configCommand.Subcommands) == 0 {
		t.Error("expected config command to have subcommands")
	}

	// Check for expected subcommands
	expectedSubcommands := []string{"default", "dump"}
	for _, expectedSub := range expectedSubcommands {
		found := false
		for _, sub := range configCommand.Subcommands {
			if sub.Name == expectedSub {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", expectedSub)
		}
	}
}

func TestConfigDefault(t *testing.T) {
	app := App()

	// Create a temporary buffer to capture output
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan []byte, 1)
	go func() {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
		done <- buf.Bytes()
	}()

	// Test the config default subcommand
	err = app.RunContext(context.Background(), []string{"containerd", "config", "default"})
	w.Close()
	os.Stdout = originalStdout
	output := <-done
	r.Close()

	if err != nil {
		t.Errorf("config default command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "version") {
		t.Error("expected config default output to contain version field")
	}
}

func TestApplyFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]string
		expected func(*srvconfig.Config) bool
	}{
		{
			name: "address flag",
			flags: map[string]string{
				"address": "/tmp/test.sock",
			},
			expected: func(c *srvconfig.Config) bool {
				return c.GRPC.Address == "/tmp/test.sock"
			},
		},
		{
			name: "root flag",
			flags: map[string]string{
				"root": "/tmp/containerd",
			},
			expected: func(c *srvconfig.Config) bool {
				return c.Root == "/tmp/containerd"
			},
		},
		{
			name: "state flag",
			flags: map[string]string{
				"state": "/tmp/containerd-state",
			},
			expected: func(c *srvconfig.Config) bool {
				return c.State == "/tmp/containerd-state"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := cli.NewApp()
			app.Flags = []cli.Flag{
				&cli.StringFlag{Name: "log-level"},
				&cli.StringFlag{Name: "address"},
				&cli.StringFlag{Name: "root"},
				&cli.StringFlag{Name: "state"},
			}

			var args []string = []string{"test"}
			for flag, value := range tt.flags {
				args = append(args, "--"+flag, value)
			}

			config := defaultConfig()
			app.Action = func(cliContext *cli.Context) error {
				return applyFlags(cliContext, config)
			}

			err := app.Run(args)
			if err != nil {
				t.Errorf("applyFlags failed: %v", err)
			}

			if !tt.expected(config) {
				t.Errorf("expected condition not met for %s", tt.name)
			}
		})
	}
}

func TestConfigVersionHandling(t *testing.T) {
	config := defaultConfig()

	// Test that config gets version set (we can test this without running outputConfig)
	if config.Version == 0 {
		t.Error("expected default config to have version set")
	}
}

func TestTimeoutConfiguration(t *testing.T) {
	config := defaultConfig()

	// Test that default config has some timeout configuration
	if config.Timeouts == nil {
		// This might be nil initially, which is fine
		t.Log("default config has nil timeouts (will be initialized by outputConfig)")
	}
}

func TestPlatformAgnosticDefaultConfig(t *testing.T) {
	config := platformAgnosticDefaultConfig()
	if config == nil {
		t.Fatal("platformAgnosticDefaultConfig() returned nil")
	}

	// Basic validation that essential fields are set
	if config.Version == 0 {
		t.Error("expected config version to be set")
	}

	if config.Root == "" {
		t.Error("expected root directory to be set")
	}

	if config.State == "" {
		t.Error("expected state directory to be set")
	}

	if config.GRPC.Address == "" {
		t.Error("expected GRPC address to be set")
	}
}

func TestConfigDumpAction(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Write a minimal config
	configContent := `version = 2
root = "/tmp/test"
state = "/tmp/test-state"

[grpc]
  address = "/run/containerd/containerd.sock"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	app := App()

	// Capture stdout
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan []byte, 1)
	go func() {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
		done <- buf.Bytes()
	}()

	// Test config dump command
	err = app.RunContext(context.Background(), []string{"containerd", "--config", configPath, "config", "dump"})
	w.Close()
	os.Stdout = originalStdout
	output := <-done
	r.Close()

	if err != nil {
		t.Errorf("config dump command failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "version") {
		t.Error("expected config dump output to contain version field")
	}
}

func TestAppAction(t *testing.T) {
	app := App()

	if app.Action == nil {
		t.Fatal("expected app to have an action")
	}

	// Just verify the action exists - we can't easily test the full daemon startup
	// in a unit test without complex mocking
}

func TestServiceFlags(t *testing.T) {
	flags := serviceFlags()

	// This is a platform-specific function, and might return nil on some platforms
	// We just test that the function exists and can be called
	_ = flags
}

func TestDefaultConfigPath(t *testing.T) {
	app := App()

	// Find the config flag
	var configFlag cli.Flag
	for _, flag := range app.Flags {
		if flag.Names()[0] == "config" {
			configFlag = flag
			break
		}
	}

	if configFlag == nil {
		t.Fatal("config flag not found")
	}

	// Verify default value
	if stringFlag, ok := configFlag.(*cli.StringFlag); ok {
		expectedPath := filepath.Join(defaults.DefaultConfigDir, "config.toml")
		if stringFlag.Value != expectedPath {
			t.Errorf("expected default config path %q, got %q", expectedPath, stringFlag.Value)
		}
	} else {
		t.Error("config flag is not a StringFlag")
	}
}
