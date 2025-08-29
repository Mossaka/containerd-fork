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

package snapshots

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestKindConstants(t *testing.T) {
	tests := []struct {
		kind     Kind
		expected string
	}{
		{KindUnknown, "Unknown"},
		{KindView, "View"},
		{KindActive, "Active"},
		{KindCommitted, "Committed"},
	}

	for _, test := range tests {
		if test.kind.String() != test.expected {
			t.Errorf("Kind.String() = %q, expected %q", test.kind.String(), test.expected)
		}
	}
}

func TestParseKind(t *testing.T) {
	tests := []struct {
		input    string
		expected Kind
	}{
		{"view", KindView},
		{"VIEW", KindView},
		{"View", KindView},
		{"active", KindActive},
		{"ACTIVE", KindActive},
		{"Active", KindActive},
		{"committed", KindCommitted},
		{"COMMITTED", KindCommitted},
		{"Committed", KindCommitted},
		{"unknown", KindUnknown},
		{"invalid", KindUnknown},
		{"", KindUnknown},
	}

	for _, test := range tests {
		result := ParseKind(test.input)
		if result != test.expected {
			t.Errorf("ParseKind(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestKindJSONMarshal(t *testing.T) {
	tests := []struct {
		kind     Kind
		expected string
	}{
		{KindView, `"View"`},
		{KindActive, `"Active"`},
		{KindCommitted, `"Committed"`},
		{KindUnknown, `"Unknown"`},
	}

	for _, test := range tests {
		data, err := json.Marshal(test.kind)
		if err != nil {
			t.Errorf("json.Marshal(%v) error: %v", test.kind, err)
			continue
		}
		if string(data) != test.expected {
			t.Errorf("json.Marshal(%v) = %s, expected %s", test.kind, string(data), test.expected)
		}
	}
}

func TestKindJSONUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected Kind
	}{
		{`"View"`, KindView},
		{`"view"`, KindView},
		{`"Active"`, KindActive},
		{`"active"`, KindActive},
		{`"Committed"`, KindCommitted},
		{`"committed"`, KindCommitted},
		{`"Unknown"`, KindUnknown},
		{`"invalid"`, KindUnknown},
	}

	for _, test := range tests {
		var kind Kind
		err := json.Unmarshal([]byte(test.input), &kind)
		if err != nil {
			t.Errorf("json.Unmarshal(%s) error: %v", test.input, err)
			continue
		}
		if kind != test.expected {
			t.Errorf("json.Unmarshal(%s) = %v, expected %v", test.input, kind, test.expected)
		}
	}
}

func TestKindJSONUnmarshalInvalidJSON(t *testing.T) {
	var kind Kind
	err := json.Unmarshal([]byte(`invalid json`), &kind)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestKindJSONRoundTrip(t *testing.T) {
	kinds := []Kind{KindView, KindActive, KindCommitted, KindUnknown}

	for _, originalKind := range kinds {
		// Marshal
		data, err := json.Marshal(originalKind)
		if err != nil {
			t.Errorf("Marshal error for %v: %v", originalKind, err)
			continue
		}

		// Unmarshal
		var unmarshaledKind Kind
		err = json.Unmarshal(data, &unmarshaledKind)
		if err != nil {
			t.Errorf("Unmarshal error for %v: %v", originalKind, err)
			continue
		}

		if originalKind != unmarshaledKind {
			t.Errorf("Round trip failed: %v != %v", originalKind, unmarshaledKind)
		}
	}
}

func TestUsageAdd(t *testing.T) {
	tests := []struct {
		name     string
		base     Usage
		add      Usage
		expected Usage
	}{
		{
			name:     "add zero usage",
			base:     Usage{Size: 100, Inodes: 10},
			add:      Usage{Size: 0, Inodes: 0},
			expected: Usage{Size: 100, Inodes: 10},
		},
		{
			name:     "add positive usage",
			base:     Usage{Size: 100, Inodes: 10},
			add:      Usage{Size: 50, Inodes: 5},
			expected: Usage{Size: 150, Inodes: 15},
		},
		{
			name:     "add to empty usage",
			base:     Usage{Size: 0, Inodes: 0},
			add:      Usage{Size: 200, Inodes: 20},
			expected: Usage{Size: 200, Inodes: 20},
		},
		{
			name:     "add large values",
			base:     Usage{Size: 1000000000, Inodes: 1000000},
			add:      Usage{Size: 500000000, Inodes: 500000},
			expected: Usage{Size: 1500000000, Inodes: 1500000},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a copy to avoid modifying the test case
			usage := test.base
			usage.Add(test.add)

			if usage.Size != test.expected.Size {
				t.Errorf("Size: got %d, expected %d", usage.Size, test.expected.Size)
			}
			if usage.Inodes != test.expected.Inodes {
				t.Errorf("Inodes: got %d, expected %d", usage.Inodes, test.expected.Inodes)
			}
		})
	}
}

func TestWithLabels(t *testing.T) {
	tests := []struct {
		name           string
		initialInfo    Info
		labelsToAdd    map[string]string
		expectedLabels map[string]string
	}{
		{
			name:           "add to nil labels",
			initialInfo:    Info{Name: "test"},
			labelsToAdd:    map[string]string{"key1": "value1"},
			expectedLabels: map[string]string{"key1": "value1"},
		},
		{
			name:        "add to existing labels",
			initialInfo: Info{Name: "test", Labels: map[string]string{"existing": "value"}},
			labelsToAdd: map[string]string{"key1": "value1", "key2": "value2"},
			expectedLabels: map[string]string{
				"existing": "value",
				"key1":     "value1",
				"key2":     "value2",
			},
		},
		{
			name:           "overwrite existing label",
			initialInfo:    Info{Name: "test", Labels: map[string]string{"key1": "oldvalue"}},
			labelsToAdd:    map[string]string{"key1": "newvalue"},
			expectedLabels: map[string]string{"key1": "newvalue"},
		},
		{
			name:           "empty labels to add",
			initialInfo:    Info{Name: "test", Labels: map[string]string{"existing": "value"}},
			labelsToAdd:    map[string]string{},
			expectedLabels: map[string]string{"existing": "value"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opt := WithLabels(test.labelsToAdd)
			info := test.initialInfo // copy

			err := opt(&info)
			if err != nil {
				t.Errorf("WithLabels option returned error: %v", err)
				return
			}

			if !reflect.DeepEqual(info.Labels, test.expectedLabels) {
				t.Errorf("Labels = %v, expected %v", info.Labels, test.expectedLabels)
			}
		})
	}
}

func TestFilterInheritedLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil labels",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty labels",
			input:    map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "only inherited labels",
			input: map[string]string{
				"containerd.io/snapshot/test":    "value1",
				"containerd.io/snapshot.ref":     "value2",
				"containerd.io/snapshot/another": "value3",
			},
			expected: map[string]string{
				"containerd.io/snapshot/test":    "value1",
				"containerd.io/snapshot.ref":     "value2",
				"containerd.io/snapshot/another": "value3",
			},
		},
		{
			name: "mixed labels",
			input: map[string]string{
				"containerd.io/snapshot/test": "value1",
				"containerd.io/snapshot.ref":  "value2",
				"custom.label":                "value3",
				"another.custom":              "value4",
			},
			expected: map[string]string{
				"containerd.io/snapshot/test": "value1",
				"containerd.io/snapshot.ref":  "value2",
			},
		},
		{
			name: "only non-inherited labels",
			input: map[string]string{
				"custom.label":   "value1",
				"another.custom": "value2",
			},
			expected: map[string]string{},
		},
		{
			name: "edge case with similar prefixes",
			input: map[string]string{
				"containerd.io/snapshot/test": "value1",
				"containerd.io/snapshot.ref":  "value2",
				"containerd.io/snapshot":      "value3", // no trailing slash
				"containerd.io/snapshotother": "value4", // different prefix
			},
			expected: map[string]string{
				"containerd.io/snapshot/test": "value1",
				"containerd.io/snapshot.ref":  "value2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := FilterInheritedLabels(test.input)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("FilterInheritedLabels() = %v, expected %v", result, test.expected)
			}
		})
	}
}

func TestInfoStructure(t *testing.T) {
	now := time.Now()

	info := Info{
		Kind:    KindActive,
		Name:    "test-snapshot",
		Parent:  "parent-snapshot",
		Labels:  map[string]string{"test": "value"},
		Created: now,
		Updated: now,
	}

	// Test field assignments
	if info.Kind != KindActive {
		t.Errorf("Kind = %v, expected %v", info.Kind, KindActive)
	}
	if info.Name != "test-snapshot" {
		t.Errorf("Name = %q, expected %q", info.Name, "test-snapshot")
	}
	if info.Parent != "parent-snapshot" {
		t.Errorf("Parent = %q, expected %q", info.Parent, "parent-snapshot")
	}
	if len(info.Labels) != 1 || info.Labels["test"] != "value" {
		t.Errorf("Labels = %v, expected map with test:value", info.Labels)
	}
	if !info.Created.Equal(now) {
		t.Errorf("Created = %v, expected %v", info.Created, now)
	}
	if !info.Updated.Equal(now) {
		t.Errorf("Updated = %v, expected %v", info.Updated, now)
	}
}

func TestInfoJSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second) // Truncate for JSON round-trip

	info := Info{
		Kind:    KindCommitted,
		Name:    "test-snapshot",
		Parent:  "parent-snapshot",
		Labels:  map[string]string{"label1": "value1", "label2": "value2"},
		Created: now,
		Updated: now,
	}

	// Marshal to JSON
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal(info) error: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaledInfo Info
	err = json.Unmarshal(data, &unmarshaledInfo)
	if err != nil {
		t.Fatalf("json.Unmarshal(data) error: %v", err)
	}

	// Compare fields
	if unmarshaledInfo.Kind != info.Kind {
		t.Errorf("Kind: got %v, expected %v", unmarshaledInfo.Kind, info.Kind)
	}
	if unmarshaledInfo.Name != info.Name {
		t.Errorf("Name: got %q, expected %q", unmarshaledInfo.Name, info.Name)
	}
	if unmarshaledInfo.Parent != info.Parent {
		t.Errorf("Parent: got %q, expected %q", unmarshaledInfo.Parent, info.Parent)
	}
	if !reflect.DeepEqual(unmarshaledInfo.Labels, info.Labels) {
		t.Errorf("Labels: got %v, expected %v", unmarshaledInfo.Labels, info.Labels)
	}
	if !unmarshaledInfo.Created.Equal(info.Created) {
		t.Errorf("Created: got %v, expected %v", unmarshaledInfo.Created, info.Created)
	}
	if !unmarshaledInfo.Updated.Equal(info.Updated) {
		t.Errorf("Updated: got %v, expected %v", unmarshaledInfo.Updated, info.Updated)
	}
}

func TestConstants(t *testing.T) {
	// Test constant values
	if UnpackKeyPrefix != "extract" {
		t.Errorf("UnpackKeyPrefix = %q, expected %q", UnpackKeyPrefix, "extract")
	}
	if UnpackKeyFormat != "extract-%s %s" {
		t.Errorf("UnpackKeyFormat = %q, expected %q", UnpackKeyFormat, "extract-%s %s")
	}
	if LabelSnapshotUIDMapping != "containerd.io/snapshot/uidmapping" {
		t.Errorf("LabelSnapshotUIDMapping = %q, expected %q", LabelSnapshotUIDMapping, "containerd.io/snapshot/uidmapping")
	}
	if LabelSnapshotGIDMapping != "containerd.io/snapshot/gidmapping" {
		t.Errorf("LabelSnapshotGIDMapping = %q, expected %q", LabelSnapshotGIDMapping, "containerd.io/snapshot/gidmapping")
	}
}
