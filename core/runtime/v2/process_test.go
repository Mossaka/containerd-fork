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

package v2

import (
	"testing"

	tasktypes "github.com/containerd/containerd/api/types/task"
	"github.com/containerd/containerd/v2/core/runtime"
)

func TestStatusFromProto(t *testing.T) {
	tests := []struct {
		name     string
		input    tasktypes.Status
		expected runtime.Status
	}{
		{
			name:     "CREATED status",
			input:    tasktypes.Status_CREATED,
			expected: runtime.CreatedStatus,
		},
		{
			name:     "RUNNING status",
			input:    tasktypes.Status_RUNNING,
			expected: runtime.RunningStatus,
		},
		{
			name:     "STOPPED status",
			input:    tasktypes.Status_STOPPED,
			expected: runtime.StoppedStatus,
		},
		{
			name:     "PAUSED status",
			input:    tasktypes.Status_PAUSED,
			expected: runtime.PausedStatus,
		},
		{
			name:     "PAUSING status",
			input:    tasktypes.Status_PAUSING,
			expected: runtime.PausingStatus,
		},
		{
			name:     "Unknown status (default case)",
			input:    tasktypes.Status(999), // Invalid status value
			expected: runtime.Status(0),     // Default zero status
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusFromProto(tt.input)
			if result != tt.expected {
				t.Errorf("statusFromProto(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcess_ID(t *testing.T) {
	testID := "test-process-id"
	p := &process{
		id: testID,
	}

	if p.ID() != testID {
		t.Errorf("expected ID %q, got %q", testID, p.ID())
	}
}

func TestProcess_ID_Empty(t *testing.T) {
	p := &process{
		id: "",
	}

	if p.ID() != "" {
		t.Errorf("expected empty ID, got %q", p.ID())
	}
}
