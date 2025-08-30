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

package httpdbg

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"
)

// mockRoundTripper is a mock http.RoundTripper for testing
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// mockAddr implements net.Addr for testing
type mockAddr struct {
	network string
	address string
}

func (m mockAddr) Network() string {
	return m.network
}

func (m mockAddr) String() string {
	return m.address
}

// mockConn implements net.Conn for testing DNS/connection tracing
type mockConn struct {
	remoteAddr net.Addr
}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr               { return nil }
func (m *mockConn) RemoteAddr() net.Addr              { return m.remoteAddr }
func (m *mockConn) SetDeadline(t time.Time) error     { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestDebugTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  *http.Response
		mockError     error
		expectedError bool
		writeError    bool
	}{
		{
			name: "successful request and response",
			mockResponse: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("response body")),
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name:          "transport returns error",
			mockResponse:  nil,
			mockError:     errors.New("transport error"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := &mockRoundTripper{
				response: tt.mockResponse,
				err:      tt.mockError,
			}

			var buf bytes.Buffer
			debugTrans := debugTransport{
				transport: mockTransport,
				writer:    &buf,
			}

			req, err := http.NewRequest("GET", "http://example.com/test", strings.NewReader("test body"))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := debugTrans.RoundTrip(req)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp != tt.mockResponse {
				t.Errorf("expected response %v, got %v", tt.mockResponse, resp)
			}

			// Check that request and response were written to buffer
			output := buf.String()
			if !strings.Contains(output, "GET /test HTTP/1.1") {
				t.Errorf("request not found in debug output: %s", output)
			}
			if tt.mockResponse != nil && !strings.Contains(output, "200 OK") {
				t.Errorf("response not found in debug output: %s", output)
			}
		})
	}
}

func TestDebugTransport_RoundTripWriteError(t *testing.T) {
	// Test write error during request dump
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("response body")),
		},
		err: nil,
	}

	// Writer that always returns an error
	errorWriter := &errorWriter{err: errors.New("write error")}

	debugTrans := debugTransport{
		transport: mockTransport,
		writer:    errorWriter,
	}

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = debugTrans.RoundTrip(req)
	if err == nil {
		t.Error("expected write error but got none")
	}
}

// errorWriter always returns an error on Write
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, e.err
}

func TestDumpRequests(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	client := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("test")),
			},
		},
	}

	DumpRequests(ctx, client, &buf)

	// Check that transport was wrapped
	debugTrans, ok := client.Transport.(debugTransport)
	if !ok {
		t.Error("client transport was not wrapped with debugTransport")
	}

	if debugTrans.writer != &buf {
		t.Error("debug transport writer not set correctly")
	}
}

func TestDumpRequestsNilWriter(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &mockRoundTripper{},
	}

	DumpRequests(ctx, client, nil)

	// Check that transport was wrapped
	_, ok := client.Transport.(debugTransport)
	if !ok {
		t.Error("client transport was not wrapped with debugTransport")
	}
}

func TestNewDebugClientTrace(t *testing.T) {
	ctx := context.Background()
	trace := NewDebugClientTrace(ctx)

	if trace == nil {
		t.Fatal("NewDebugClientTrace returned nil")
	}

	// Test that the trace has the expected callbacks
	if trace.DNSStart == nil {
		t.Error("DNSStart callback not set")
	}

	if trace.DNSDone == nil {
		t.Error("DNSDone callback not set")
	}

	if trace.GotConn == nil {
		t.Error("GotConn callback not set")
	}
}

func TestDebugClientTrace_DNSStart(t *testing.T) {
	ctx := context.Background()
	trace := NewDebugClientTrace(ctx)

	// This should not panic
	trace.DNSStart(httptrace.DNSStartInfo{
		Host: "example.com",
	})
}

func TestDebugClientTrace_DNSDone(t *testing.T) {
	ctx := context.Background()
	trace := NewDebugClientTrace(ctx)

	// Test successful DNS resolution
	addrs := []net.IPAddr{
		{IP: net.ParseIP("127.0.0.1")},
	}

	trace.DNSDone(httptrace.DNSDoneInfo{
		Addrs:     addrs,
		Err:       nil,
		Coalesced: false,
	})

	// Test DNS resolution with error
	trace.DNSDone(httptrace.DNSDoneInfo{
		Addrs:     nil,
		Err:       errors.New("DNS error"),
		Coalesced: false,
	})
}

func TestDebugClientTrace_GotConn(t *testing.T) {
	ctx := context.Background()
	trace := NewDebugClientTrace(ctx)

	// Test with connection that has RemoteAddr
	mockConnWithAddr := &mockConn{
		remoteAddr: &mockAddr{
			network: "tcp",
			address: "127.0.0.1:80",
		},
	}

	trace.GotConn(httptrace.GotConnInfo{
		Conn:   mockConnWithAddr,
		Reused: false,
	})

	// Test with connection that has nil RemoteAddr
	mockConnNilAddr := &mockConn{
		remoteAddr: nil,
	}

	trace.GotConn(httptrace.GotConnInfo{
		Conn:   mockConnNilAddr,
		Reused: true,
	})
}

func TestTraceTransport_RoundTrip(t *testing.T) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("test")),
		},
	}

	ctx := context.Background()
	tracer := NewDebugClientTrace(ctx)

	traceTrans := traceTransport{
		tracer:    tracer,
		transport: mockTransport,
	}

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := traceTrans.RoundTrip(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp != mockTransport.response {
		t.Errorf("expected response %v, got %v", mockTransport.response, resp)
	}
}

func TestDumpTraces(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{
		Transport: &mockRoundTripper{},
	}

	DumpTraces(ctx, client)

	// Check that transport was wrapped
	traceTrans, ok := client.Transport.(traceTransport)
	if !ok {
		t.Error("client transport was not wrapped with traceTransport")
	}

	if traceTrans.tracer == nil {
		t.Error("trace transport tracer not set")
	}
}

func TestWithClientTrace(t *testing.T) {
	ctx := context.Background()
	newCtx := WithClientTrace(ctx)

	if newCtx == ctx {
		t.Error("WithClientTrace returned the same context")
	}

	// Check that the new context has a client trace
	trace := httptrace.ContextClientTrace(newCtx)
	if trace == nil {
		t.Error("context does not contain client trace")
	}

	if trace.DNSStart == nil || trace.DNSDone == nil || trace.GotConn == nil {
		t.Error("client trace does not have expected callbacks")
	}
}

func TestDebugTransportWithNilTransport(t *testing.T) {
	var buf bytes.Buffer
	debugTrans := debugTransport{
		transport: nil,
		writer:    &buf,
	}

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// This should panic or return an error
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil transport but got none")
		}
	}()

	_, _ = debugTrans.RoundTrip(req)
}

func TestDebugTransportResponseDumpError(t *testing.T) {
	// Test that response dumping errors are handled
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       &errorReader{err: errors.New("read error")},
		},
	}

	var buf bytes.Buffer
	debugTrans := debugTransport{
		transport: mockTransport,
		writer:    &buf,
	}

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = debugTrans.RoundTrip(req)
	if err == nil {
		t.Error("expected error from DumpResponse but got none")
	}

	if !strings.Contains(err.Error(), "failed to dump response") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// errorReader always returns an error on Read
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReader) Close() error {
	return nil
}

func BenchmarkDebugTransportRoundTrip(b *testing.B) {
	mockTransport := &mockRoundTripper{
		response: &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("test response")),
		},
	}

	var buf bytes.Buffer
	debugTrans := debugTransport{
		transport: mockTransport,
		writer:    &buf,
	}

	req, err := http.NewRequest("GET", "http://example.com/test", strings.NewReader("test body"))
	if err != nil {
		b.Fatalf("failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset response body for each iteration
		mockTransport.response.Body = io.NopCloser(strings.NewReader("test response"))
		_, err := debugTrans.RoundTrip(req)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkNewDebugClientTrace(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		trace := NewDebugClientTrace(ctx)
		_ = trace
	}
}