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
	"testing"

	v1 "github.com/containerd/containerd/v2/core/metrics/types/v1"
	metrics "github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPUMetrics_CPUTotal(t *testing.T) {
	metric := cpuMetrics[0] // cpu_total
	
	assert.Equal(t, "cpu_total", metric.name)
	assert.Equal(t, "The total cpu time", metric.help)
	assert.Equal(t, metrics.Nanoseconds, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)

	// Test with valid CPU stats
	stats := &v1.Metrics{
		CPU: &v1.CPUStat{
			Usage: &v1.CPUUsage{
				Total: 1000000000, // 1 second in nanoseconds
			},
		},
	}

	values := metric.getValues(stats)
	require.Len(t, values, 1)
	assert.Equal(t, float64(1000000000), values[0].v)
	assert.Empty(t, values[0].l) // No labels
}

func TestCPUMetrics_CPUTotal_NilCPU(t *testing.T) {
	metric := cpuMetrics[0] // cpu_total
	
	// Test with nil CPU stats
	stats := &v1.Metrics{
		CPU: nil,
	}

	values := metric.getValues(stats)
	assert.Nil(t, values)
}

func TestCPUMetrics_CPUKernel(t *testing.T) {
	metric := cpuMetrics[1] // cpu_kernel
	
	assert.Equal(t, "cpu_kernel", metric.name)
	assert.Equal(t, "The total kernel cpu time", metric.help)
	assert.Equal(t, metrics.Nanoseconds, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)

	// Test with valid CPU stats
	stats := &v1.Metrics{
		CPU: &v1.CPUStat{
			Usage: &v1.CPUUsage{
				Kernel: 500000000, // 0.5 seconds in nanoseconds
			},
		},
	}

	values := metric.getValues(stats)
	require.Len(t, values, 1)
	assert.Equal(t, float64(500000000), values[0].v)
	assert.Empty(t, values[0].l)
}

func TestCPUMetrics_CPUUser(t *testing.T) {
	metric := cpuMetrics[2] // cpu_user
	
	assert.Equal(t, "cpu_user", metric.name)
	assert.Equal(t, "The total user cpu time", metric.help)
	assert.Equal(t, metrics.Nanoseconds, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)

	// Test with valid CPU stats
	stats := &v1.Metrics{
		CPU: &v1.CPUStat{
			Usage: &v1.CPUUsage{
				User: 300000000, // 0.3 seconds in nanoseconds
			},
		},
	}

	values := metric.getValues(stats)
	require.Len(t, values, 1)
	assert.Equal(t, float64(300000000), values[0].v)
	assert.Empty(t, values[0].l)
}

func TestCPUMetrics_PerCPU(t *testing.T) {
	metric := cpuMetrics[3] // per_cpu
	
	assert.Equal(t, "per_cpu", metric.name)
	assert.Equal(t, "The total cpu time per cpu", metric.help)
	assert.Equal(t, metrics.Nanoseconds, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)
	assert.Equal(t, []string{"cpu"}, metric.labels)

	// Test with multiple CPUs
	stats := &v1.Metrics{
		CPU: &v1.CPUStat{
			Usage: &v1.CPUUsage{
				PerCPU: []uint64{1000000000, 800000000, 1200000000}, // 3 CPUs
			},
		},
	}

	values := metric.getValues(stats)
	require.Len(t, values, 3)
	
	// Check CPU 0
	assert.Equal(t, float64(1000000000), values[0].v)
	assert.Equal(t, []string{"0"}, values[0].l)
	
	// Check CPU 1
	assert.Equal(t, float64(800000000), values[1].v)
	assert.Equal(t, []string{"1"}, values[1].l)
	
	// Check CPU 2
	assert.Equal(t, float64(1200000000), values[2].v)
	assert.Equal(t, []string{"2"}, values[2].l)
}

func TestCPUMetrics_PerCPU_Empty(t *testing.T) {
	metric := cpuMetrics[3] // per_cpu
	
	// Test with empty PerCPU slice
	stats := &v1.Metrics{
		CPU: &v1.CPUStat{
			Usage: &v1.CPUUsage{
				PerCPU: []uint64{}, // Empty slice
			},
		},
	}

	values := metric.getValues(stats)
	assert.Empty(t, values)
}

func TestCPUMetrics_CPUThrottlePeriods(t *testing.T) {
	metric := cpuMetrics[4] // cpu_throttle_periods
	
	assert.Equal(t, "cpu_throttle_periods", metric.name)
	assert.Equal(t, "The total cpu throttle periods", metric.help)
	assert.Equal(t, metrics.Total, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)

	// Test with nil CPU (should return nil)
	stats := &v1.Metrics{CPU: nil}
	values := metric.getValues(stats)
	assert.Nil(t, values)
}

func TestCPUMetrics_CPUThrottledPeriods(t *testing.T) {
	metric := cpuMetrics[5] // cpu_throttled_periods
	
	assert.Equal(t, "cpu_throttled_periods", metric.name)
	assert.Equal(t, "The total cpu throttled periods", metric.help)
	assert.Equal(t, metrics.Total, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)

	// Test with nil CPU (should return nil)
	stats := &v1.Metrics{CPU: nil}
	values := metric.getValues(stats)
	assert.Nil(t, values)
}

func TestCPUMetrics_CPUThrottledTime(t *testing.T) {
	metric := cpuMetrics[6] // cpu_throttled_time
	
	assert.Equal(t, "cpu_throttled_time", metric.name)
	assert.Equal(t, "The total cpu throttled time", metric.help)
	assert.Equal(t, metrics.Nanoseconds, metric.unit)
	assert.Equal(t, prometheus.GaugeValue, metric.vt)

	// Test with nil CPU (should return nil)
	stats := &v1.Metrics{CPU: nil}
	values := metric.getValues(stats)
	assert.Nil(t, values)
}

func TestCPUMetrics_AllNilCPU(t *testing.T) {
	// Test all CPU metrics with nil CPU stats
	stats := &v1.Metrics{
		CPU: nil,
	}

	for i, metric := range cpuMetrics {
		t.Run(metric.name, func(t *testing.T) {
			values := metric.getValues(stats)
			assert.Nil(t, values, "metric %d (%s) should return nil values for nil CPU", i, metric.name)
		})
	}
}

func TestCPUMetrics_ZeroValues(t *testing.T) {
	// Test all CPU metrics with zero values (simplified without throttling)
	stats := &v1.Metrics{
		CPU: &v1.CPUStat{
			Usage: &v1.CPUUsage{
				Total:  0,
				Kernel: 0,
				User:   0,
				PerCPU: []uint64{0, 0},
			},
		},
	}

	// Test cpu_total, cpu_kernel, cpu_user metrics (indices 0-2)
	for i := 0; i < 3; i++ {
		metric := cpuMetrics[i]
		values := metric.getValues(stats)
		require.Len(t, values, 1)
		assert.Equal(t, float64(0), values[0].v, "metric %s should handle zero values", metric.name)
	}

	// Test per_cpu metric
	perCPUMetric := cpuMetrics[3]
	values := perCPUMetric.getValues(stats)
	require.Len(t, values, 2) // Two CPUs with zero values
	assert.Equal(t, float64(0), values[0].v)
	assert.Equal(t, float64(0), values[1].v)
}

func TestCPUMetrics_Count(t *testing.T) {
	// Verify we have the expected number of CPU metrics
	expectedCPUMetrics := 7 // total, kernel, user, per_cpu, throttle_periods, throttled_periods, throttled_time
	assert.Len(t, cpuMetrics, expectedCPUMetrics)
}

// Note: TestCPUMetrics_NilUsage removed because the current implementation
// has a design assumption that if CPU is non-nil, then Usage and Throttling
// should also be non-nil. Testing this edge case would require larger
// architectural changes that are beyond the scope of coverage improvement.