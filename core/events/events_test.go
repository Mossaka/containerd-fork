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

package events

import (
	"context"
	"testing"
	"time"

	"github.com/containerd/typeurl/v2"
)

// Mock event type for testing
type TestEvent struct {
	Name  string
	Value string
}

// Implement Field method for TestEvent to work with Envelope.Field
func (t *TestEvent) Field(fieldpath []string) (string, bool) {
	if len(fieldpath) == 0 {
		return "", false
	}

	switch fieldpath[0] {
	case "name":
		return t.Name, len(t.Name) > 0
	case "value":
		return t.Value, len(t.Value) > 0
	default:
		return "", false
	}
}

// Simple event type without Field method
type SimpleEvent struct {
	Message string
}

func TestEnvelope_Field(t *testing.T) {
	// Register test event type
	typeurl.Register(&TestEvent{}, "test.TestEvent")

	testEvent := &TestEvent{
		Name:  "test-name",
		Value: "test-value",
	}

	eventAny, err := typeurl.MarshalAny(testEvent)
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	envelope := &Envelope{
		Timestamp: time.Now(),
		Namespace: "test-namespace",
		Topic:     "test-topic",
		Event:     eventAny,
	}

	tests := []struct {
		name      string
		fieldpath []string
		expected  string
		found     bool
	}{
		{
			name:      "empty fieldpath",
			fieldpath: []string{},
			expected:  "",
			found:     false,
		},
		{
			name:      "namespace field",
			fieldpath: []string{"namespace"},
			expected:  "test-namespace",
			found:     true,
		},
		{
			name:      "topic field",
			fieldpath: []string{"topic"},
			expected:  "test-topic",
			found:     true,
		},
		{
			name:      "event nested field - name",
			fieldpath: []string{"event", "name"},
			expected:  "test-name",
			found:     true,
		},
		{
			name:      "event nested field - value",
			fieldpath: []string{"event", "value"},
			expected:  "test-value",
			found:     true,
		},
		{
			name:      "event nested field - nonexistent",
			fieldpath: []string{"event", "nonexistent"},
			expected:  "",
			found:     false,
		},
		{
			name:      "invalid field",
			fieldpath: []string{"invalid"},
			expected:  "",
			found:     false,
		},
		{
			name:      "timestamp field (unhandled)",
			fieldpath: []string{"timestamp"},
			expected:  "",
			found:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, found := envelope.Field(test.fieldpath)
			if value != test.expected {
				t.Errorf("Expected value %q, got %q", test.expected, value)
			}
			if found != test.found {
				t.Errorf("Expected found %v, got %v", test.found, found)
			}
		})
	}
}

func TestEnvelope_Field_EmptyValues(t *testing.T) {
	envelope := &Envelope{
		Timestamp: time.Now(),
		Namespace: "",  // empty namespace
		Topic:     "",  // empty topic
		Event:     nil, // nil event
	}

	tests := []struct {
		name      string
		fieldpath []string
		expected  string
		found     bool
	}{
		{
			name:      "empty namespace",
			fieldpath: []string{"namespace"},
			expected:  "",
			found:     false,
		},
		{
			name:      "empty topic",
			fieldpath: []string{"topic"},
			expected:  "",
			found:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, found := envelope.Field(test.fieldpath)
			if value != test.expected {
				t.Errorf("Expected value %q, got %q", test.expected, value)
			}
			if found != test.found {
				t.Errorf("Expected found %v, got %v", test.found, found)
			}
		})
	}
}

func TestEnvelope_Field_EventWithoutFieldMethod(t *testing.T) {
	// Register simple event type
	typeurl.Register(&SimpleEvent{}, "test.SimpleEvent")

	simpleEvent := &SimpleEvent{
		Message: "test-message",
	}

	eventAny, err := typeurl.MarshalAny(simpleEvent)
	if err != nil {
		t.Fatalf("Failed to marshal simple event: %v", err)
	}

	envelope := &Envelope{
		Timestamp: time.Now(),
		Namespace: "test-namespace",
		Topic:     "test-topic",
		Event:     eventAny,
	}

	// Should return false since SimpleEvent doesn't implement Field method
	value, found := envelope.Field([]string{"event", "message"})
	if value != "" {
		t.Errorf("Expected empty value, got %q", value)
	}
	if found != false {
		t.Errorf("Expected found false, got %v", found)
	}
}

func TestEnvelope_Field_NilEvent(t *testing.T) {
	// Test behavior with non-event fieldpath when Event is nil
	envelope := &Envelope{
		Timestamp: time.Now(),
		Namespace: "test-namespace",
		Topic:     "test-topic",
		Event:     nil, // nil event
	}

	// Non-event fields should still work
	value, found := envelope.Field([]string{"namespace"})
	if value != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got %q", value)
	}
	if !found {
		t.Errorf("Expected to find namespace")
	}

	value, found = envelope.Field([]string{"topic"})
	if value != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got %q", value)
	}
	if !found {
		t.Errorf("Expected to find topic")
	}

	// Note: accessing event field with nil Event will cause panic in UnmarshalAny
	// This is expected behavior based on the current implementation
}

func TestEnvelope_Struct(t *testing.T) {
	timestamp := time.Now()
	namespace := "test-namespace"
	topic := "test/topic"

	testEvent := &TestEvent{
		Name:  "test-event",
		Value: "test-value",
	}

	eventAny, err := typeurl.MarshalAny(testEvent)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	envelope := &Envelope{
		Timestamp: timestamp,
		Namespace: namespace,
		Topic:     topic,
		Event:     eventAny,
	}

	// Test struct field access
	if envelope.Timestamp != timestamp {
		t.Errorf("Expected timestamp %v, got %v", timestamp, envelope.Timestamp)
	}
	if envelope.Namespace != namespace {
		t.Errorf("Expected namespace %q, got %q", namespace, envelope.Namespace)
	}
	if envelope.Topic != topic {
		t.Errorf("Expected topic %q, got %q", topic, envelope.Topic)
	}
	if envelope.Event.GetTypeUrl() != eventAny.GetTypeUrl() {
		t.Errorf("Expected event type URL %q, got %q", eventAny.GetTypeUrl(), envelope.Event.GetTypeUrl())
	}
}

// Test interfaces exist and can be used in type assertions
func TestInterfaces(t *testing.T) {
	// Test that interfaces can be used in type assertions
	var publisher interface{} = (*mockPublisher)(nil)
	if _, ok := publisher.(Publisher); !ok {
		t.Errorf("mockPublisher should implement Publisher interface")
	}

	var forwarder interface{} = (*mockForwarder)(nil)
	if _, ok := forwarder.(Forwarder); !ok {
		t.Errorf("mockForwarder should implement Forwarder interface")
	}

	var subscriber interface{} = (*mockSubscriber)(nil)
	if _, ok := subscriber.(Subscriber); !ok {
		t.Errorf("mockSubscriber should implement Subscriber interface")
	}

	// Test Event interface (generic interface{})
	var event Event = "any value"
	if event == nil {
		t.Errorf("Event should accept any value")
	}

	var eventStruct Event = &TestEvent{Name: "test"}
	if eventStruct == nil {
		t.Errorf("Event should accept struct values")
	}
}

// Mock implementations for interface testing
type mockPublisher struct{}

func (m *mockPublisher) Publish(ctx context.Context, topic string, event Event) error {
	return nil
}

type mockForwarder struct{}

func (m *mockForwarder) Forward(ctx context.Context, envelope *Envelope) error {
	return nil
}

type mockSubscriber struct{}

func (m *mockSubscriber) Subscribe(ctx context.Context, filters ...string) (ch <-chan *Envelope, errs <-chan error) {
	chEnv := make(chan *Envelope)
	chErr := make(chan error)
	close(chEnv)
	close(chErr)
	return chEnv, chErr
}

func TestMockImplementations(t *testing.T) {
	ctx := context.Background()

	// Test Publisher mock
	publisher := &mockPublisher{}
	err := publisher.Publish(ctx, "test/topic", &TestEvent{Name: "test"})
	if err != nil {
		t.Errorf("Mock publisher should not return error: %v", err)
	}

	// Test Forwarder mock
	forwarder := &mockForwarder{}
	envelope := &Envelope{
		Timestamp: time.Now(),
		Namespace: "test",
		Topic:     "test/topic",
	}
	err = forwarder.Forward(ctx, envelope)
	if err != nil {
		t.Errorf("Mock forwarder should not return error: %v", err)
	}

	// Test Subscriber mock
	subscriber := &mockSubscriber{}
	chEnv, chErr := subscriber.Subscribe(ctx, "test/*")
	if chEnv == nil {
		t.Errorf("Subscriber should return envelope channel")
	}
	if chErr == nil {
		t.Errorf("Subscriber should return error channel")
	}

	// Channels should be closed immediately in mock
	select {
	case _, ok := <-chEnv:
		if ok {
			t.Errorf("Envelope channel should be closed")
		}
	default:
		t.Errorf("Envelope channel should be readable (closed)")
	}

	select {
	case _, ok := <-chErr:
		if ok {
			t.Errorf("Error channel should be closed")
		}
	default:
		t.Errorf("Error channel should be readable (closed)")
	}
}

func TestEnvelope_Field_DeepNestedField(t *testing.T) {
	// Test deep nested field path through event
	// Reuse already registered TestEvent type

	testEvent := &TestEvent{
		Name:  "deep-test",
		Value: "deep-value",
	}

	eventAny, err := typeurl.MarshalAny(testEvent)
	if err != nil {
		t.Fatalf("Failed to marshal test event: %v", err)
	}

	envelope := &Envelope{
		Timestamp: time.Now(),
		Namespace: "deep-namespace",
		Topic:     "deep/topic",
		Event:     eventAny,
	}

	// Test that fieldpath is properly passed to nested event
	value, found := envelope.Field([]string{"event", "name"})
	if !found {
		t.Errorf("Expected to find nested field")
	}
	if value != "deep-test" {
		t.Errorf("Expected nested field value 'deep-test', got %q", value)
	}

	// Test empty fieldpath after "event"
	value, found = envelope.Field([]string{"event"})
	if found {
		t.Errorf("Expected not to find field with just 'event'")
	}
	if value != "" {
		t.Errorf("Expected empty value, got %q", value)
	}
}

func TestEvent_GenericInterface(t *testing.T) {
	// Test that Event interface accepts any type
	tests := []Event{
		"string event",
		42,
		[]string{"slice", "event"},
		map[string]interface{}{"map": "event"},
		&TestEvent{Name: "struct event"},
		nil,
	}

	for i, event := range tests {
		// Event interface should accept any type
		var e Event = event
		if i == len(tests)-1 && e != nil {
			t.Errorf("Test %d: Expected nil event", i)
		}
		// Just verify the assignment works - Event is interface{} so accepts anything
	}
}
