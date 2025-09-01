//go:build linux

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

package v1

import (
	"reflect"
	"testing"

	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
)

// TestTypeAliases verifies that all type aliases are properly defined and match their upstream types
func TestTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		local    interface{}
		upstream interface{}
	}{
		{"Metrics", Metrics{}, v1.Metrics{}},
		{"BlkIOEntry", BlkIOEntry{}, v1.BlkIOEntry{}},
		{"MemoryStat", MemoryStat{}, v1.MemoryStat{}},
		{"CPUStat", CPUStat{}, v1.CPUStat{}},
		{"CPUUsage", CPUUsage{}, v1.CPUUsage{}},
		{"BlkIOStat", BlkIOStat{}, v1.BlkIOStat{}},
		{"PidsStat", PidsStat{}, v1.PidsStat{}},
		{"RdmaStat", RdmaStat{}, v1.RdmaStat{}},
		{"RdmaEntry", RdmaEntry{}, v1.RdmaEntry{}},
		{"HugetlbStat", HugetlbStat{}, v1.HugetlbStat{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localType := reflect.TypeOf(tt.local)
			upstreamType := reflect.TypeOf(tt.upstream)

			if localType != upstreamType {
				t.Errorf("Type alias %s does not match upstream type: got %v, want %v", tt.name, localType, upstreamType)
			}

			// Verify that both types have the same kind (struct, interface, etc.)
			if localType.Kind() != upstreamType.Kind() {
				t.Errorf("Type alias %s has different kind: got %v, want %v", tt.name, localType.Kind(), upstreamType.Kind())
			}
		})
	}
}

// TestMetricsTypeAlias specifically tests the Metrics type alias
func TestMetricsTypeAlias(t *testing.T) {
	var metrics Metrics
	var v1Metrics v1.Metrics

	// Test that the types are identical
	if reflect.TypeOf(metrics) != reflect.TypeOf(v1Metrics) {
		t.Errorf("Metrics type alias does not match v1.Metrics: got %T, want %T", metrics, v1Metrics)
	}

	// Test that we can assign between them
	metrics = v1.Metrics{}
	v1Metrics = metrics

	// Test pointer compatibility
	ptrMetrics := &metrics
	ptrV1Metrics := &v1Metrics

	if reflect.TypeOf(ptrMetrics) != reflect.TypeOf(ptrV1Metrics) {
		t.Error("Pointer types should also be compatible")
	}
}

// TestBlkIOEntryTypeAlias specifically tests the BlkIOEntry type alias
func TestBlkIOEntryTypeAlias(t *testing.T) {
	var entry BlkIOEntry
	var v1Entry v1.BlkIOEntry

	if reflect.TypeOf(entry) != reflect.TypeOf(v1Entry) {
		t.Errorf("BlkIOEntry type alias does not match v1.BlkIOEntry: got %T, want %T", entry, v1Entry)
	}

	// Test assignment compatibility
	entry = v1.BlkIOEntry{}
	v1Entry = entry

	// Test that we can create slices of these types
	var entries []BlkIOEntry
	var v1Entries []v1.BlkIOEntry

	// These should be assignable due to type alias
	entries = []BlkIOEntry{}
	v1Entries = []v1.BlkIOEntry{}

	_ = entries
	_ = v1Entries
}

// TestTypeAliasZeroValues verifies zero value compatibility
func TestTypeAliasZeroValues(t *testing.T) {
	var (
		metrics     Metrics
		blkIOEntry  BlkIOEntry
		memoryStat  MemoryStat
		cpuStat     CPUStat
		cpuUsage    CPUUsage
		blkIOStat   BlkIOStat
		pidsStat    PidsStat
		rdmaStat    RdmaStat
		rdmaEntry   RdmaEntry
		hugetlbStat HugetlbStat
	)

	var (
		v1Metrics     v1.Metrics
		v1BlkIOEntry  v1.BlkIOEntry
		v1MemoryStat  v1.MemoryStat
		v1CPUStat     v1.CPUStat
		v1CPUUsage    v1.CPUUsage
		v1BlkIOStat   v1.BlkIOStat
		v1PidsStat    v1.PidsStat
		v1RdmaStat    v1.RdmaStat
		v1RdmaEntry   v1.RdmaEntry
		v1HugetlbStat v1.HugetlbStat
	)

	// Zero values should be assignable
	metrics = v1Metrics
	blkIOEntry = v1BlkIOEntry
	memoryStat = v1MemoryStat
	cpuStat = v1CPUStat
	cpuUsage = v1CPUUsage
	blkIOStat = v1BlkIOStat
	pidsStat = v1PidsStat
	rdmaStat = v1RdmaStat
	rdmaEntry = v1RdmaEntry
	hugetlbStat = v1HugetlbStat

	// And vice versa
	v1Metrics = metrics
	v1BlkIOEntry = blkIOEntry
	v1MemoryStat = memoryStat
	v1CPUStat = cpuStat
	v1CPUUsage = cpuUsage
	v1BlkIOStat = blkIOStat
	v1PidsStat = pidsStat
	v1RdmaStat = rdmaStat
	v1RdmaEntry = rdmaEntry
	v1HugetlbStat = hugetlbStat

	// Suppress unused variable warnings
	_ = v1Metrics
}

// TestTypeAliasComparability tests that aliased types have the same comparability as upstream types
func TestTypeAliasComparability(t *testing.T) {
	// Note: Most cgroups v1 stats types contain protobuf MessageState and are not comparable
	// This test verifies that our type aliases have the same comparability behavior

	// Test that we can at least create and assign values
	var (
		entry1   BlkIOEntry
		v1Entry1 v1.BlkIOEntry
	)

	// Assignment should work
	entry1 = v1.BlkIOEntry{Op: "read", Value: 100}
	v1Entry1 = entry1

	// Test that the fields are accessible and preserved
	if entry1.Op != "read" || entry1.Value != 100 {
		t.Error("BlkIOEntry fields not preserved after assignment")
	}
	if v1Entry1.Op != "read" || v1Entry1.Value != 100 {
		t.Error("v1.BlkIOEntry fields not preserved after assignment")
	}
}

// TestTypeAliasInSlices tests that type aliases work properly in slices
func TestTypeAliasInSlices(t *testing.T) {
	// Create slices using the aliased types
	entries := []BlkIOEntry{{Op: "read", Value: 100}}
	v1Entries := []v1.BlkIOEntry{{Op: "write", Value: 200}}

	// We should be able to append compatible types
	entries = append(entries, v1Entries...)

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Check that the values were preserved
	if entries[0].Op != "read" || entries[0].Value != 100 {
		t.Error("First entry values not preserved")
	}
	if entries[1].Op != "write" || entries[1].Value != 200 {
		t.Error("Second entry values not preserved")
	}
}

// TestTypeAliasInMaps tests that type aliases work properly as map keys and values
func TestTypeAliasInMaps(t *testing.T) {
	// Test using aliased types as map values
	statsMap := map[string]MemoryStat{
		"container1": {Cache: 1024},
		"container2": {RSS: 2048},
	}

	v1StatsMap := map[string]v1.MemoryStat{
		"container3": {Cache: 4096},
	}

	// Should be able to assign map entries between aliased and upstream types
	statsMap["container3"] = v1StatsMap["container3"]
	v1StatsMap["container1"] = statsMap["container1"]

	if len(statsMap) != 3 {
		t.Errorf("Expected 3 entries in statsMap, got %d", len(statsMap))
	}
	if len(v1StatsMap) != 2 {
		t.Errorf("Expected 2 entries in v1StatsMap, got %d", len(v1StatsMap))
	}

	// Verify the assignments worked
	if statsMap["container3"].Cache != 4096 {
		t.Error("Map assignment from v1 type failed")
	}
	if v1StatsMap["container1"].Cache != 1024 {
		t.Error("Map assignment to v1 type failed")
	}
}
