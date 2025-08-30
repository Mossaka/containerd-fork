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
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/containerd/containerd/v2/core/metrics/cgroups/common"
	v2 "github.com/containerd/containerd/v2/core/metrics/types/v2"
	"github.com/containerd/containerd/v2/pkg/protobuf/types"
	"github.com/containerd/typeurl/v2"
	"github.com/docker/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStatableTask implements common.Statable for testing
type mockStatableTask struct {
	id        string
	namespace string
	stats     *v2.Metrics
}

func (m *mockStatableTask) ID() string {
	return m.id
}

func (m *mockStatableTask) Namespace() string {
	return m.namespace
}

func (m *mockStatableTask) Stats(ctx context.Context) (*types.Any, error) {
	any, err := typeurl.MarshalAny(m.stats)
	if err != nil {
		return nil, err
	}
	return &types.Any{
		TypeUrl: any.GetTypeUrl(),
		Value:   any.GetValue(),
	}, nil
}

// TestNewCollector tests the collector creation
func TestNewCollector(t *testing.T) {
	t.Run("WithNamespace", func(t *testing.T) {
		ns := metrics.NewNamespace("containerd", "test", nil)
		collector := NewCollector(ns)

		assert.NotNil(t, collector)
		assert.Equal(t, ns, collector.ns)
		assert.NotNil(t, collector.tasks)
		assert.NotEmpty(t, collector.metrics)
	})

	t.Run("WithNilNamespace", func(t *testing.T) {
		collector := NewCollector(nil)

		assert.NotNil(t, collector)
		assert.Nil(t, collector.ns)
		assert.Nil(t, collector.tasks)   // tasks map is nil when namespace is nil
		assert.Nil(t, collector.metrics) // metrics slice is nil when namespace is nil
	})
}

// TestCollectorAdd tests adding tasks to the collector
func TestCollectorAdd(t *testing.T) {
	ns := metrics.NewNamespace("containerd", "test", nil)
	collector := NewCollector(ns)

	stats := &v2.Metrics{
		Pids: &v2.PidsStat{
			Current: 5,
			Limit:   100,
		},
		CPU: &v2.CPUStat{
			UsageUsec:     1000000,
			UserUsec:      600000,
			SystemUsec:    400000,
			NrPeriods:     10,
			NrThrottled:   2,
			ThrottledUsec: 50000,
		},
		Memory: &v2.MemoryStat{
			Usage:      1024 * 1024,
			UsageLimit: 100 * 1024 * 1024,
			SwapUsage:  512 * 1024,
			SwapLimit:  10 * 1024 * 1024,
		},
	}

	task := &mockStatableTask{
		id:        "test-container",
		namespace: "test-namespace",
		stats:     stats,
	}

	t.Run("AddTask", func(t *testing.T) {
		err := collector.Add(task, nil)
		require.NoError(t, err)

		collector.mu.RLock()
		taskID := taskID(task.id, task.namespace)
		entry, exists := collector.tasks[taskID]
		collector.mu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, task, entry.task)
		assert.Nil(t, entry.ns)
	})

	t.Run("AddTaskWithLabels", func(t *testing.T) {
		labels := map[string]string{
			"service": "nginx",
			"version": "1.18",
		}

		task2 := &mockStatableTask{
			id:        "test-container-2",
			namespace: "test-namespace",
			stats:     stats,
		}

		err := collector.Add(task2, labels)
		require.NoError(t, err)

		collector.mu.RLock()
		taskID := taskID(task2.id, task2.namespace)
		entry, exists := collector.tasks[taskID]
		collector.mu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, task2, entry.task)
		assert.NotNil(t, entry.ns) // Should have custom namespace with labels
	})

	t.Run("AddDuplicateTask", func(t *testing.T) {
		// Adding the same task should be idempotent
		originalCount := len(collector.tasks)

		err := collector.Add(task, nil)
		require.NoError(t, err)

		collector.mu.RLock()
		newCount := len(collector.tasks)
		collector.mu.RUnlock()

		assert.Equal(t, originalCount, newCount)
	})
}

// TestCollectorAddNilNamespace tests adding tasks when collector has nil namespace
func TestCollectorAddNilNamespace(t *testing.T) {
	collector := NewCollector(nil)

	task := &mockStatableTask{
		id:        "test-container",
		namespace: "test-namespace",
		stats:     &v2.Metrics{},
	}

	err := collector.Add(task, nil)
	require.NoError(t, err)

	// Should not add tasks when namespace is nil
	collector.mu.RLock()
	taskCount := len(collector.tasks)
	collector.mu.RUnlock()

	assert.Equal(t, 0, taskCount)
}

// TestCollectorDescribe tests Prometheus describe functionality
func TestCollectorDescribe(t *testing.T) {
	ns := metrics.NewNamespace("containerd", "test", nil)
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

	// Should have multiple metric descriptions from pids, cpu, memory, io
	assert.NotEmpty(t, descriptions)
}

// TestTaskID tests the taskID helper function
func TestTaskID(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		namespace  string
		expectedID string
	}{
		{
			name:       "NormalTaskID",
			taskID:     "container1",
			namespace:  "default",
			expectedID: "container1-default",
		},
		{
			name:       "EmptyTaskID",
			taskID:     "",
			namespace:  "default",
			expectedID: "-default",
		},
		{
			name:       "EmptyNamespace",
			taskID:     "container1",
			namespace:  "",
			expectedID: "container1-",
		},
		{
			name:       "BothEmpty",
			taskID:     "",
			namespace:  "",
			expectedID: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := taskID(tt.taskID, tt.namespace)
			assert.Equal(t, tt.expectedID, result)
		})
	}
}

// TestCollectorConcurrentAccess tests thread safety of the collector
func TestCollectorConcurrentAccess(t *testing.T) {
	ns := metrics.NewNamespace("containerd", "test", nil)
	collector := NewCollector(ns)

	const numGoroutines = 10
	const tasksPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent Add operations
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < tasksPerGoroutine; j++ {
				task := &mockStatableTask{
					id:        fmt.Sprintf("task-%d-%d", routineID, j),
					namespace: "test-ns",
					stats:     &v2.Metrics{Pids: &v2.PidsStat{Current: 1}},
				}

				err := collector.Add(task, nil)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all tasks were added
	collector.mu.RLock()
	taskCount := len(collector.tasks)
	collector.mu.RUnlock()

	assert.Equal(t, numGoroutines*tasksPerGoroutine, taskCount)
}

// TestCollectorRemove tests removing tasks from the collector
func TestCollectorRemove(t *testing.T) {
	ns := metrics.NewNamespace("containerd", "test", nil)
	collector := NewCollector(ns)

	// Add a task first
	task := &mockStatableTask{
		id:        "test-container",
		namespace: "test-namespace",
		stats:     &v2.Metrics{Pids: &v2.PidsStat{Current: 1}},
	}

	err := collector.Add(task, nil)
	require.NoError(t, err)

	// Verify task was added
	collector.mu.RLock()
	originalCount := len(collector.tasks)
	collector.mu.RUnlock()
	assert.Equal(t, 1, originalCount)

	// Remove the task using the Remove method
	taskToRemove := common.Statable(task)
	collector.Remove(taskToRemove)

	// Verify task was removed
	collector.mu.RLock()
	finalCount := len(collector.tasks)
	collector.mu.RUnlock()
	assert.Equal(t, 0, finalCount)
}
