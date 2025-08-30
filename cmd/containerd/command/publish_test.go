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
	"testing"

	"github.com/urfave/cli/v2"
)

func TestPublishCommand(t *testing.T) {
	if publishCommand == nil {
		t.Fatal("publishCommand is nil")
	}

	if publishCommand.Name != "publish" {
		t.Errorf("expected publish command name 'publish', got %q", publishCommand.Name)
	}

	if publishCommand.Usage != "Binary to publish events to containerd" {
		t.Errorf("expected specific usage string, got %q", publishCommand.Usage)
	}

	// Check expected flags
	expectedFlags := []string{"namespace", "topic"}
	for _, expectedFlag := range expectedFlags {
		found := false
		for _, flag := range publishCommand.Flags {
			if flag.Names()[0] == expectedFlag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected flag %q not found", expectedFlag)
		}
	}

	// Verify action is set
	if publishCommand.Action == nil {
		t.Error("expected publish command to have an action")
	}
}

func TestPublishCommandFlags(t *testing.T) {
	tests := []struct {
		flagName     string
		expectedType string
	}{
		{"namespace", "StringFlag"},
		{"topic", "StringFlag"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			found := false
			for _, flag := range publishCommand.Flags {
				if flag.Names()[0] == tt.flagName {
					found = true
					// Basic type checking - we know these should be string flags
					if _, ok := flag.(*cli.StringFlag); !ok && tt.expectedType == "StringFlag" {
						t.Errorf("expected %s to be StringFlag", tt.flagName)
					}
					break
				}
			}
			if !found {
				t.Errorf("flag %s not found", tt.flagName)
			}
		})
	}
}
