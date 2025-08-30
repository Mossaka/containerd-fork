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

package v2

import (
	"reflect"
	"testing"

	v2 "github.com/containerd/cgroups/v3/cgroup2/stats"
)

// TestTypeAliases verifies that all type aliases are properly defined and match their upstream types
func TestTypeAliases(t *testing.T) {
	tests := []struct {
		name     string
		local    interface{}
		upstream interface{}
	}{
		{"Metrics", Metrics{}, v2.Metrics{}},
		{"MemoryStat", MemoryStat{}, v2.MemoryStat{}},
		{"CPUStat", CPUStat{}, v2.CPUStat{}},
		{"PidsStat", PidsStat{}, v2.PidsStat{}},
		{"IOStat", IOStat{}, v2.IOStat{}},
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
	var v2Metrics v2.Metrics

	// Test that the types are identical
	if reflect.TypeOf(metrics) != reflect.TypeOf(v2Metrics) {
		t.Errorf("Metrics type alias does not match v2.Metrics: got %T, want %T", metrics, v2Metrics)
	}

	// Test that we can assign between them
	metrics = v2.Metrics{}
	v2Metrics = metrics

	// Test pointer compatibility
	ptrMetrics := &metrics
	ptrV2Metrics := &v2Metrics

	if reflect.TypeOf(ptrMetrics) != reflect.TypeOf(ptrV2Metrics) {
		t.Error("Pointer types should also be compatible")
	}
}

// TestMemoryStatTypeAlias specifically tests the MemoryStat type alias
func TestMemoryStatTypeAlias(t *testing.T) {
	var memoryStat MemoryStat
	var v2MemoryStat v2.MemoryStat

	if reflect.TypeOf(memoryStat) != reflect.TypeOf(v2MemoryStat) {
		t.Errorf("MemoryStat type alias does not match v2.MemoryStat: got %T, want %T", memoryStat, v2MemoryStat)
	}

	// Test assignment compatibility
	memoryStat = v2.MemoryStat{}
	v2MemoryStat = memoryStat

	// Test in a slice context
	memoryStats := []MemoryStat{{Usage: 1024}}
	v2MemoryStats := []v2.MemoryStat{{UsageLimit: 2048}}

	// Should be able to append between alias types
	memoryStats = append(memoryStats, v2MemoryStats...)

	if len(memoryStats) != 2 {
		t.Errorf("Expected 2 memory stats, got %d", len(memoryStats))
	}
	if memoryStats[0].Usage != 1024 {
		t.Error("First memory stat usage not preserved")
	}
	if memoryStats[1].UsageLimit != 2048 {
		t.Error("Second memory stat usage limit not preserved")
	}
}

// TestCPUStatTypeAlias specifically tests the CPUStat type alias
func TestCPUStatTypeAlias(t *testing.T) {
	var cpuStat CPUStat
	var v2CPUStat v2.CPUStat

	if reflect.TypeOf(cpuStat) != reflect.TypeOf(v2CPUStat) {
		t.Errorf("CPUStat type alias does not match v2.CPUStat: got %T, want %T", cpuStat, v2CPUStat)
	}

	// Test assignment compatibility
	cpuStat = v2.CPUStat{}
	v2CPUStat = cpuStat
}

// TestPidsStatTypeAlias specifically tests the PidsStat type alias
func TestPidsStatTypeAlias(t *testing.T) {
	var pidsStat PidsStat
	var v2PidsStat v2.PidsStat

	if reflect.TypeOf(pidsStat) != reflect.TypeOf(v2PidsStat) {
		t.Errorf("PidsStat type alias does not match v2.PidsStat: got %T, want %T", pidsStat, v2PidsStat)
	}

	// Test assignment compatibility with actual PidsStat values
	pidsStat = v2.PidsStat{Current: 10, Limit: 100}
	v2PidsStat = pidsStat

	if v2PidsStat.Current != 10 || v2PidsStat.Limit != 100 {
		t.Error("PidsStat assignment failed to preserve field values")
	}
}

// TestIOStatTypeAlias specifically tests the IOStat type alias
func TestIOStatTypeAlias(t *testing.T) {
	var ioStat IOStat
	var v2IOStat v2.IOStat

	if reflect.TypeOf(ioStat) != reflect.TypeOf(v2IOStat) {
		t.Errorf("IOStat type alias does not match v2.IOStat: got %T, want %T", ioStat, v2IOStat)
	}

	// Test assignment compatibility
	ioStat = v2.IOStat{}
	v2IOStat = ioStat
}

// TestTypeAliasZeroValues verifies zero value compatibility
func TestTypeAliasZeroValues(t *testing.T) {
	var (
		metrics    Metrics
		memoryStat MemoryStat
		cpuStat    CPUStat
		pidsStat   PidsStat
		ioStat     IOStat
	)

	var (
		v2Metrics    v2.Metrics
		v2MemoryStat v2.MemoryStat
		v2CPUStat    v2.CPUStat
		v2PidsStat   v2.PidsStat
		v2IOStat     v2.IOStat
	)

	// Zero values should be assignable
	metrics = v2Metrics
	memoryStat = v2MemoryStat
	cpuStat = v2CPUStat
	pidsStat = v2PidsStat
	ioStat = v2IOStat

	// And vice versa
	v2Metrics = metrics
	v2MemoryStat = memoryStat
	v2CPUStat = cpuStat
	v2PidsStat = pidsStat
	v2IOStat = ioStat

	// Suppress unused variable warnings
	_ = v2Metrics
}

// TestTypeAliasComparability tests that aliased types can be assigned and field values preserved
func TestTypeAliasComparability(t *testing.T) {
	// Test basic assignment and field access
	var (
		pids1   PidsStat
		v2Pids1 v2.PidsStat
	)

	// Test assignment with actual field values
	pids1 = PidsStat{Current: 10, Limit: 100}
	v2Pids1 = pids1

	// Verify field values are preserved
	if v2Pids1.Current != 10 || v2Pids1.Limit != 100 {
		t.Error("Field values not preserved in type alias assignment")
	}

	// Test reverse assignment
	v2Pids2 := v2.PidsStat{Current: 20, Limit: 200}
	pids2 := PidsStat(v2Pids2)

	if pids2.Current != 20 || pids2.Limit != 200 {
		t.Error("Field values not preserved in reverse assignment")
	}
}

// TestTypeAliasInMaps tests that type aliases work properly as map keys and values
func TestTypeAliasInMaps(t *testing.T) {
	// Test using aliased types as map values
	statsMap := map[string]MemoryStat{
		"container1": {Usage: 1024, UsageLimit: 2048},
		"container2": {Usage: 4096, UsageLimit: 8192},
	}

	v2StatsMap := map[string]v2.MemoryStat{
		"container3": {Usage: 16384, UsageLimit: 32768},
	}

	// Should be able to assign map entries between aliased and upstream types
	statsMap["container3"] = v2StatsMap["container3"]
	v2StatsMap["container1"] = statsMap["container1"]

	if len(statsMap) != 3 {
		t.Errorf("Expected 3 entries in statsMap, got %d", len(statsMap))
	}
	if len(v2StatsMap) != 2 {
		t.Errorf("Expected 2 entries in v2StatsMap, got %d", len(v2StatsMap))
	}

	// Verify the assignments worked
	if statsMap["container3"].Usage != 16384 || statsMap["container3"].UsageLimit != 32768 {
		t.Error("Map assignment from v2 type failed")
	}
	if v2StatsMap["container1"].Usage != 1024 || v2StatsMap["container1"].UsageLimit != 2048 {
		t.Error("Map assignment to v2 type failed")
	}
}

// TestTypeAliasInComplexStructures tests that type aliases work in complex nested structures
func TestTypeAliasInComplexStructures(t *testing.T) {
	// Create a complex Metrics structure using aliased types
	metrics := Metrics{
		Memory: &MemoryStat{
			Usage:      1048576, // 1MB
			UsageLimit: 2097152, // 2MB
		},
		CPU: &CPUStat{
			UsageUsec:  1000000, // 1 second
			UserUsec:   500000,  // 0.5 seconds
			SystemUsec: 500000,  // 0.5 seconds
		},
		Pids: &PidsStat{
			Current: 15,
			Limit:   150,
		},
	}

	// Should be able to convert to upstream v2.Metrics
	var v2Metrics v2.Metrics = metrics

	// Verify that nested structures were preserved
	if v2Metrics.Memory == nil {
		t.Error("Memory stat not preserved in type conversion")
	}
	if v2Metrics.CPU == nil {
		t.Error("CPU stat not preserved in type conversion")
	}
	if v2Metrics.Pids == nil {
		t.Error("Pids stat not preserved in type conversion")
	}

	// Verify specific nested field values
	if v2Metrics.Memory.Usage != 1048576 {
		t.Errorf("Memory usage not preserved: got %d, want %d", v2Metrics.Memory.Usage, 1048576)
	}
	if v2Metrics.Memory.UsageLimit != 2097152 {
		t.Errorf("Memory usage limit not preserved: got %d, want %d", v2Metrics.Memory.UsageLimit, 2097152)
	}
	if v2Metrics.CPU.UsageUsec != 1000000 {
		t.Errorf("CPU usage not preserved: got %d, want %d", v2Metrics.CPU.UsageUsec, 1000000)
	}
	if v2Metrics.Pids.Current != 15 {
		t.Errorf("Pids current not preserved: got %d, want %d", v2Metrics.Pids.Current, 15)
	}
}

// TestV2SpecificFeatures tests features specific to cgroups v2 that differ from v1
func TestV2SpecificFeatures(t *testing.T) {
	// Test v2-specific CPU stats with microsecond precision
	cpuStat := CPUStat{
		UsageUsec:  2000000, // 2 seconds in microseconds
		UserUsec:   1000000, // 1 second
		SystemUsec: 1000000, // 1 second
	}

	// Verify the v2 CPU statistics structure
	if cpuStat.UsageUsec != 2000000 {
		t.Errorf("Expected UsageUsec 2000000, got %d", cpuStat.UsageUsec)
	}
	if cpuStat.UserUsec != 1000000 {
		t.Errorf("Expected UserUsec 1000000, got %d", cpuStat.UserUsec)
	}
	if cpuStat.SystemUsec != 1000000 {
		t.Errorf("Expected SystemUsec 1000000, got %d", cpuStat.SystemUsec)
	}

	// Test that we can assign to upstream v2 types
	var v2CPUStat v2.CPUStat = cpuStat
	if v2CPUStat.UsageUsec != 2000000 {
		t.Error("Assignment to v2.CPUStat failed to preserve UsageUsec")
	}
}
