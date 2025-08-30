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

package streaming

import (
	"context"
	"errors"
	"testing"

	"github.com/containerd/typeurl/v2"
)

// testAny is a simple implementation of typeurl.Any for testing
type testAny struct {
	typeURL string
	value   []byte
}

func (t *testAny) GetTypeUrl() string {
	return t.typeURL
}

func (t *testAny) GetValue() []byte {
	return t.value
}

func newTestAny(typeURL string, value []byte) typeurl.Any {
	return &testAny{typeURL: typeURL, value: value}
}

// mockStream implements the Stream interface for testing
type mockStream struct {
	sendFunc  func(typeurl.Any) error
	recvFunc  func() (typeurl.Any, error)
	closeFunc func() error
}

func (m *mockStream) Send(a typeurl.Any) error {
	if m.sendFunc != nil {
		return m.sendFunc(a)
	}
	return nil
}

func (m *mockStream) Recv() (typeurl.Any, error) {
	if m.recvFunc != nil {
		return m.recvFunc()
	}
	return newTestAny("", nil), nil
}

func (m *mockStream) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// mockStreamManager implements StreamManager interface for testing
type mockStreamManager struct {
	streams map[string]Stream
	getFunc func(context.Context, string) (Stream, error)
	regFunc func(context.Context, string, Stream) error
}

func (m *mockStreamManager) Get(ctx context.Context, id string) (Stream, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	if s, exists := m.streams[id]; exists {
		return s, nil
	}
	return nil, errors.New("stream not found")
}

func (m *mockStreamManager) Register(ctx context.Context, id string, stream Stream) error {
	if m.regFunc != nil {
		return m.regFunc(ctx, id, stream)
	}
	if m.streams == nil {
		m.streams = make(map[string]Stream)
	}
	m.streams[id] = stream
	return nil
}

// mockStreamGetter implements StreamGetter interface for testing
type mockStreamGetter struct {
	streams map[string]Stream
	getFunc func(context.Context, string) (Stream, error)
}

func (m *mockStreamGetter) Get(ctx context.Context, id string) (Stream, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	if s, exists := m.streams[id]; exists {
		return s, nil
	}
	return nil, errors.New("stream not found")
}

// mockStreamCreator implements StreamCreator interface for testing
type mockStreamCreator struct {
	createFunc func(context.Context, string) (Stream, error)
}

func (m *mockStreamCreator) Create(ctx context.Context, id string) (Stream, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, id)
	}
	return &mockStream{}, nil
}

func TestStreamInterface(t *testing.T) {
	t.Run("Stream Send/Recv/Close", func(t *testing.T) {
		var sentData typeurl.Any
		recvData := newTestAny("test", []byte("data"))

		stream := &mockStream{
			sendFunc: func(a typeurl.Any) error {
				sentData = a
				return nil
			},
			recvFunc: func() (typeurl.Any, error) {
				return recvData, nil
			},
			closeFunc: func() error {
				return nil
			},
		}

		// Test Send
		testData := newTestAny("test", []byte("test"))
		if err := stream.Send(testData); err != nil {
			t.Errorf("Send failed: %v", err)
		}
		if sentData.GetTypeUrl() != testData.GetTypeUrl() {
			t.Errorf("Expected sent data type %v, got %v", testData.GetTypeUrl(), sentData.GetTypeUrl())
		}

		// Test Recv
		received, err := stream.Recv()
		if err != nil {
			t.Errorf("Recv failed: %v", err)
		}
		if received.GetTypeUrl() != recvData.GetTypeUrl() {
			t.Errorf("Expected received data type %v, got %v", recvData.GetTypeUrl(), received.GetTypeUrl())
		}

		// Test Close
		if err := stream.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	t.Run("Stream Send Error", func(t *testing.T) {
		expectedErr := errors.New("send error")
		stream := &mockStream{
			sendFunc: func(typeurl.Any) error {
				return expectedErr
			},
		}

		err := stream.Send(newTestAny("", nil))
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("Stream Recv Error", func(t *testing.T) {
		expectedErr := errors.New("recv error")
		stream := &mockStream{
			recvFunc: func() (typeurl.Any, error) {
				return nil, expectedErr
			},
		}

		_, err := stream.Recv()
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("Stream Close Error", func(t *testing.T) {
		expectedErr := errors.New("close error")
		stream := &mockStream{
			closeFunc: func() error {
				return expectedErr
			},
		}

		err := stream.Close()
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestStreamManagerInterface(t *testing.T) {
	ctx := context.Background()

	t.Run("StreamManager Register and Get", func(t *testing.T) {
		manager := &mockStreamManager{
			streams: make(map[string]Stream),
		}

		stream := &mockStream{}
		id := "test-stream-1"

		// Test Register
		if err := manager.Register(ctx, id, stream); err != nil {
			t.Errorf("Register failed: %v", err)
		}

		// Test Get
		retrieved, err := manager.Get(ctx, id)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if retrieved != stream {
			t.Errorf("Expected retrieved stream %v, got %v", stream, retrieved)
		}
	})

	t.Run("StreamManager Get Non-existent Stream", func(t *testing.T) {
		manager := &mockStreamManager{
			streams: make(map[string]Stream),
		}

		_, err := manager.Get(ctx, "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent stream")
		}
	})

	t.Run("StreamManager Register Error", func(t *testing.T) {
		expectedErr := errors.New("register error")
		manager := &mockStreamManager{
			regFunc: func(context.Context, string, Stream) error {
				return expectedErr
			},
		}

		err := manager.Register(ctx, "test", &mockStream{})
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("StreamManager Get Error", func(t *testing.T) {
		expectedErr := errors.New("get error")
		manager := &mockStreamManager{
			getFunc: func(context.Context, string) (Stream, error) {
				return nil, expectedErr
			},
		}

		_, err := manager.Get(ctx, "test")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestStreamGetterInterface(t *testing.T) {
	ctx := context.Background()

	t.Run("StreamGetter Get Success", func(t *testing.T) {
		stream := &mockStream{}
		getter := &mockStreamGetter{
			streams: map[string]Stream{
				"test-stream": stream,
			},
		}

		retrieved, err := getter.Get(ctx, "test-stream")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if retrieved != stream {
			t.Errorf("Expected retrieved stream %v, got %v", stream, retrieved)
		}
	})

	t.Run("StreamGetter Get Non-existent Stream", func(t *testing.T) {
		getter := &mockStreamGetter{
			streams: make(map[string]Stream),
		}

		_, err := getter.Get(ctx, "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent stream")
		}
	})

	t.Run("StreamGetter Get Error", func(t *testing.T) {
		expectedErr := errors.New("get error")
		getter := &mockStreamGetter{
			getFunc: func(context.Context, string) (Stream, error) {
				return nil, expectedErr
			},
		}

		_, err := getter.Get(ctx, "test")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestStreamCreatorInterface(t *testing.T) {
	ctx := context.Background()

	t.Run("StreamCreator Create Success", func(t *testing.T) {
		creator := &mockStreamCreator{}

		stream, err := creator.Create(ctx, "test-stream")
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
		if stream == nil {
			t.Error("Expected non-nil stream")
		}
	})

	t.Run("StreamCreator Create Error", func(t *testing.T) {
		expectedErr := errors.New("create error")
		creator := &mockStreamCreator{
			createFunc: func(context.Context, string) (Stream, error) {
				return nil, expectedErr
			},
		}

		_, err := creator.Create(ctx, "test")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestIntegrationScenarios(t *testing.T) {
	ctx := context.Background()

	t.Run("Complete Stream Lifecycle", func(t *testing.T) {
		manager := &mockStreamManager{
			streams: make(map[string]Stream),
		}

		// Create and register a stream
		stream := &mockStream{
			sendFunc: func(a typeurl.Any) error {
				return nil
			},
			recvFunc: func() (typeurl.Any, error) {
				return newTestAny("response", []byte("ok")), nil
			},
		}

		streamID := "lifecycle-test"
		if err := manager.Register(ctx, streamID, stream); err != nil {
			t.Errorf("Register failed: %v", err)
		}

		// Retrieve and use the stream
		retrieved, err := manager.Get(ctx, streamID)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}

		// Send data
		if err := retrieved.Send(newTestAny("request", []byte("test"))); err != nil {
			t.Errorf("Send failed: %v", err)
		}

		// Receive data
		response, err := retrieved.Recv()
		if err != nil {
			t.Errorf("Recv failed: %v", err)
		}
		if response.GetTypeUrl() != "response" {
			t.Errorf("Expected response TypeUrl 'response', got '%s'", response.GetTypeUrl())
		}

		// Close stream
		if err := retrieved.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	t.Run("StreamCreator and StreamManager Integration", func(t *testing.T) {
		var createdStream Stream
		creator := &mockStreamCreator{
			createFunc: func(ctx context.Context, id string) (Stream, error) {
				createdStream = &mockStream{}
				return createdStream, nil
			},
		}

		manager := &mockStreamManager{
			streams: make(map[string]Stream),
		}

		// Create stream via creator
		streamID := "integration-test"
		stream, err := creator.Create(ctx, streamID)
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}

		// Register in manager
		if err := manager.Register(ctx, streamID, stream); err != nil {
			t.Errorf("Register failed: %v", err)
		}

		// Verify retrieval
		retrieved, err := manager.Get(ctx, streamID)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if retrieved != createdStream {
			t.Error("Retrieved stream does not match created stream")
		}
	})
}

// Benchmark tests for streaming interface performance
func BenchmarkStreamSend(b *testing.B) {
	stream := &mockStream{
		sendFunc: func(typeurl.Any) error { return nil },
	}
	data := newTestAny("bench", []byte("benchmark data"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Send(data)
	}
}

func BenchmarkStreamRecv(b *testing.B) {
	stream := &mockStream{
		recvFunc: func() (typeurl.Any, error) {
			return newTestAny("bench", []byte("benchmark data")), nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Recv()
	}
}

func BenchmarkStreamManagerRegister(b *testing.B) {
	ctx := context.Background()
	manager := &mockStreamManager{streams: make(map[string]Stream)}
	stream := &mockStream{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Register(ctx, "bench-stream", stream)
	}
}

func BenchmarkStreamManagerGet(b *testing.B) {
	ctx := context.Background()
	manager := &mockStreamManager{
		streams: map[string]Stream{
			"bench-stream": &mockStream{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Get(ctx, "bench-stream")
	}
}