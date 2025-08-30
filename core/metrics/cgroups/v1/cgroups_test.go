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
	"testing"

	"github.com/containerd/containerd/v2/core/events"
	"github.com/docker/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPublisher struct {
	published []interface{}
}

func (m *mockPublisher) Publish(ctx context.Context, topic string, event events.Event) error {
	m.published = append(m.published, event)
	return nil
}

func TestNewTaskMonitor(t *testing.T) {
	ctx := context.Background()
	publisher := &mockPublisher{}
	ns := metrics.NewNamespace("test", "", nil)

	monitor, err := NewTaskMonitor(ctx, publisher, ns)
	require.NoError(t, err)
	require.NotNil(t, monitor)

	cgroupMonitor, ok := monitor.(*cgroupsMonitor)
	require.True(t, ok)
	assert.Equal(t, ctx, cgroupMonitor.context)
	assert.Equal(t, publisher, cgroupMonitor.publisher)
	assert.NotNil(t, cgroupMonitor.collector)
	assert.NotNil(t, cgroupMonitor.oom)
}

func TestNewTaskMonitorWithNilNamespace(t *testing.T) {
	ctx := context.Background()
	publisher := &mockPublisher{}

	monitor, err := NewTaskMonitor(ctx, publisher, nil)
	require.NoError(t, err)
	require.NotNil(t, monitor)
}

func TestTaskID(t *testing.T) {
	id := taskID("container1", "namespace1")
	expected := "container1-namespace1"
	assert.Equal(t, expected, id)
}