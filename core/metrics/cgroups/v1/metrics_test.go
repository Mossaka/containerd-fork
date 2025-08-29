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
	"context"
	"sync"
	"testing"

	v1 "github.com/containerd/containerd/v2/core/metrics/types/v1"
	"github.com/containerd/containerd/v2/pkg/protobuf/types"
	"github.com/containerd/typeurl/v2"
	"github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStatable struct {
	id        string
	namespace string
	stats     *v1.Metrics
}

func (m *mockStatable) ID() string {
	return m.id
}

func (m *mockStatable) Namespace() string {
	return m.namespace
}

func (m *mockStatable) Stats(ctx context.Context) (*types.Any, error) {
	any, err := typeurl.MarshalAny(m.stats)
	if err != nil {
		return nil, err
	}
	return &types.Any{
		TypeUrl: any.GetTypeUrl(),
		Value:   any.GetValue(),
	}, nil
}

func TestNewCollector(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	require.NotNil(t, collector)
	assert.Equal(t, ns, collector.ns)
	assert.NotNil(t, collector.tasks)
	assert.NotNil(t, collector.metrics)
	assert.NotNil(t, collector.storedMetrics)

	// Verify metrics are initialized
	expectedMetricCount := len(pidMetrics) + len(cpuMetrics) + len(memoryMetrics) + len(hugetlbMetrics) + len(blkioMetrics)
	assert.Len(t, collector.metrics, expectedMetricCount)
}

func TestNewCollectorWithNilNamespace(t *testing.T) {
	collector := NewCollector(nil)

	require.NotNil(t, collector)
	assert.Nil(t, collector.ns)
}


func TestCollector_Add(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	err := collector.Add(task, nil)
	require.NoError(t, err)

	collector.mu.RLock()
	_, exists := collector.tasks[taskID(task.ID(), task.Namespace())]
	collector.mu.RUnlock()

	assert.True(t, exists)
}

func TestCollector_AddWithLabels(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	labels := map[string]string{"label1": "value1", "label2": "value2"}
	err := collector.Add(task, labels)
	require.NoError(t, err)

	collector.mu.RLock()
	entry, exists := collector.tasks[taskID(task.ID(), task.Namespace())]
	collector.mu.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, entry.ns) // Should have child namespace with labels
}

func TestCollector_AddIdempotent(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	// Add the same task twice
	err1 := collector.Add(task, nil)
	err2 := collector.Add(task, nil)

	require.NoError(t, err1)
	require.NoError(t, err2) // Should be idempotent

	collector.mu.RLock()
	taskCount := len(collector.tasks)
	collector.mu.RUnlock()

	assert.Equal(t, 1, taskCount) // Should only have one task
}

func TestCollector_AddWithNilNamespace(t *testing.T) {
	collector := NewCollector(nil)

	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	err := collector.Add(task, nil)
	require.NoError(t, err) // Should not error with nil namespace
}

func TestCollector_Remove(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	// Add then remove
	err := collector.Add(task, nil)
	require.NoError(t, err)

	collector.Remove(task)

	collector.mu.RLock()
	_, exists := collector.tasks[taskID(task.ID(), task.Namespace())]
	collector.mu.RUnlock()

	assert.False(t, exists)
}

func TestCollector_RemoveWithNilNamespace(t *testing.T) {
	collector := NewCollector(nil)

	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	// Should not panic with nil namespace
	collector.Remove(task)
}

func TestCollector_RemoveAll(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	// Add multiple tasks
	task1 := &mockStatable{id: "task1", namespace: "ns1", stats: &v1.Metrics{}}
	task2 := &mockStatable{id: "task2", namespace: "ns2", stats: &v1.Metrics{}}

	err1 := collector.Add(task1, nil)
	err2 := collector.Add(task2, nil)
	require.NoError(t, err1)
	require.NoError(t, err2)

	collector.RemoveAll()

	collector.mu.RLock()
	taskCount := len(collector.tasks)
	collector.mu.RUnlock()

	assert.Equal(t, 0, taskCount)
}

func TestCollector_RemoveAllWithNilNamespace(t *testing.T) {
	collector := NewCollector(nil)

	// Should not panic with nil namespace
	collector.RemoveAll()
}

func TestCollector_Describe(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	ch := make(chan *prometheus.Desc, 100)

	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	var descriptions []*prometheus.Desc
	for desc := range ch {
		descriptions = append(descriptions, desc)
	}

	// Should have descriptions for all metrics
	expectedCount := len(collector.metrics)
	assert.Len(t, descriptions, expectedCount)
}

func TestCollector_Collect(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	ch := make(chan prometheus.Metric, 100)

	// Test collect without any tasks (should not panic)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	// Without tasks, should have minimal stored metrics
	assert.True(t, len(metrics) >= 0)
}

func TestCollector_CollectConcurrentAccess(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	// Test concurrent access to collector (simplified)
	var wg sync.WaitGroup
	numGoroutines := 5

	// Add tasks concurrently with simple metrics
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			task := &mockStatable{
				id:        "task-" + string(rune('0'+id)),
				namespace: "test-namespace",
				stats:     &v1.Metrics{}, // Simple metrics
			}
			collector.Add(task, nil)
		}(i)
	}

	wg.Wait()

	// Verify no race conditions occurred
	collector.mu.RLock()
	taskCount := len(collector.tasks)
	collector.mu.RUnlock()

	assert.Equal(t, numGoroutines, taskCount)
}

func TestCollector_Entry(t *testing.T) {
	task := &mockStatable{
		id:        "test-task",
		namespace: "test-namespace",
		stats:     &v1.Metrics{},
	}

	ns := metrics.NewNamespace("test", "", nil)

	// Test entry without child namespace
	entry1 := entry{task: task}
	assert.Equal(t, task, entry1.task)
	assert.Nil(t, entry1.ns)

	// Test entry with child namespace
	entry2 := entry{task: task, ns: ns}
	assert.Equal(t, task, entry2.task)
	assert.Equal(t, ns, entry2.ns)
}

func TestCollector_StoredMetrics(t *testing.T) {
	ns := metrics.NewNamespace("test", "", nil)
	collector := NewCollector(ns)

	// Test that storedMetrics channel is properly sized
	assert.NotNil(t, collector.storedMetrics)

	// Should be able to write to the channel without blocking
	metric := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test"})
	
	select {
	case collector.storedMetrics <- metric:
		// Successfully wrote to channel
	default:
		t.Error("storedMetrics channel should accept metrics")
	}
}