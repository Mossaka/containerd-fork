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

package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	transfertypes "github.com/containerd/containerd/api/types/transfer"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/remotes"
	"github.com/containerd/containerd/v2/core/streaming"
	"github.com/containerd/typeurl/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestWithHeaders(t *testing.T) {
	headers := http.Header{
		"Authorization": []string{"Bearer token"},
		"User-Agent":    []string{"test-client/1.0"},
	}
	
	opt := WithHeaders(headers)
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithHeaders() error = %v, want nil", err)
	}
	
	if opts.headers == nil {
		t.Error("expected headers to be set")
	}
	
	if opts.headers.Get("Authorization") != "Bearer token" {
		t.Errorf("expected Authorization header = 'Bearer token', got %v", opts.headers.Get("Authorization"))
	}
}

func TestWithCredentials(t *testing.T) {
	creds := &mockCredentialHelper{}
	opt := WithCredentials(creds)
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithCredentials() error = %v, want nil", err)
	}
	
	if opts.creds != creds {
		t.Error("expected credentials to be set")
	}
}

func TestWithHostDir(t *testing.T) {
	hostDir := "/etc/containerd/certs.d"
	opt := WithHostDir(hostDir)
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithHostDir() error = %v, want nil", err)
	}
	
	if opts.hostDir != hostDir {
		t.Errorf("expected hostDir = %v, got %v", hostDir, opts.hostDir)
	}
}

func TestWithDefaultScheme(t *testing.T) {
	scheme := "http"
	opt := WithDefaultScheme(scheme)
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithDefaultScheme() error = %v, want nil", err)
	}
	
	if opts.defaultScheme != scheme {
		t.Errorf("expected defaultScheme = %v, got %v", scheme, opts.defaultScheme)
	}
}

func TestWithHTTPDebug(t *testing.T) {
	opt := WithHTTPDebug()
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithHTTPDebug() error = %v, want nil", err)
	}
	
	if !opts.httpDebug {
		t.Error("expected httpDebug to be true")
	}
}

func TestWithHTTPTrace(t *testing.T) {
	opt := WithHTTPTrace()
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithHTTPTrace() error = %v, want nil", err)
	}
	
	if !opts.httpTrace {
		t.Error("expected httpTrace to be true")
	}
}

func TestWithClientStream(t *testing.T) {
	writer := &mockWriteCloser{}
	opt := WithClientStream(writer)
	opts := &registryOpts{}
	
	err := opt(opts)
	if err != nil {
		t.Errorf("WithClientStream() error = %v, want nil", err)
	}
	
	if opts.localStream != writer {
		t.Error("expected localStream to be set")
	}
}

func TestNewOCIRegistry(t *testing.T) {
	ctx := context.Background()
	ref := "docker.io/library/hello-world:latest"
	
	registry, err := NewOCIRegistry(ctx, ref)
	if err != nil {
		t.Fatalf("NewOCIRegistry() error = %v, want nil", err)
	}
	
	if registry.reference != ref {
		t.Errorf("expected reference = %v, got %v", ref, registry.reference)
	}
	
	if registry.resolver == nil {
		t.Error("expected resolver to be initialized")
	}
}

func TestNewOCIRegistryWithOptions(t *testing.T) {
	ctx := context.Background()
	ref := "docker.io/library/hello-world:latest"
	headers := http.Header{"Authorization": []string{"Bearer token"}}
	creds := &mockCredentialHelper{}
	hostDir := "/etc/containerd/certs.d"
	
	registry, err := NewOCIRegistry(ctx, ref,
		WithHeaders(headers),
		WithCredentials(creds),
		WithHostDir(hostDir),
		WithDefaultScheme("http"),
		WithHTTPDebug(),
		WithHTTPTrace(),
	)
	if err != nil {
		t.Fatalf("NewOCIRegistry() with options error = %v, want nil", err)
	}
	
	if registry.reference != ref {
		t.Errorf("expected reference = %v, got %v", ref, registry.reference)
	}
	
	if registry.headers.Get("Authorization") != "Bearer token" {
		t.Error("expected headers to be set")
	}
	
	if registry.creds != creds {
		t.Error("expected credentials to be set")
	}
	
	if registry.hostDir != hostDir {
		t.Error("expected hostDir to be set")
	}
	
	if !registry.httpDebug {
		t.Error("expected httpDebug to be true")
	}
	
	if !registry.httpTrace {
		t.Error("expected httpTrace to be true")
	}
}

func TestOCIRegistry_String(t *testing.T) {
	registry := &OCIRegistry{reference: "docker.io/library/hello-world:latest"}
	expected := "OCI Registry (docker.io/library/hello-world:latest)"
	
	if registry.String() != expected {
		t.Errorf("expected String() = %v, got %v", expected, registry.String())
	}
}

func TestOCIRegistry_Image(t *testing.T) {
	ref := "docker.io/library/hello-world:latest"
	registry := &OCIRegistry{reference: ref}
	
	if registry.Image() != ref {
		t.Errorf("expected Image() = %v, got %v", ref, registry.Image())
	}
}

func TestOCIRegistry_Resolve(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{resolver: &mockResolver{}}
	
	name, desc, err := registry.Resolve(ctx)
	if err != nil {
		t.Errorf("Resolve() error = %v, want nil", err)
	}
	
	if name != "test-name" {
		t.Errorf("expected name = 'test-name', got %v", name)
	}
	
	if desc.MediaType != "application/vnd.oci.image.manifest.v1+json" {
		t.Errorf("expected MediaType = 'application/vnd.oci.image.manifest.v1+json', got %v", desc.MediaType)
	}
}

func TestOCIRegistry_Fetcher(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{resolver: &mockResolver{}}
	
	fetcher, err := registry.Fetcher(ctx, "test-ref")
	if err != nil {
		t.Errorf("Fetcher() error = %v, want nil", err)
	}
	
	if fetcher == nil {
		t.Error("expected fetcher to be non-nil")
	}
}

func TestOCIRegistry_Pusher(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{
		reference: "docker.io/library/hello-world:latest",
		resolver:  &mockResolver{},
	}
	
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:abc123",
		Size:      1234,
	}
	
	pusher, err := registry.Pusher(ctx, desc)
	if err != nil {
		t.Errorf("Pusher() error = %v, want nil", err)
	}
	
	if pusher == nil {
		t.Error("expected pusher to be non-nil")
	}
}

func TestOCIRegistry_PusherWithDigest(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{
		reference: "docker.io/library/hello-world@sha256:existing",
		resolver:  &mockResolver{},
	}
	
	desc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:abc123",
		Size:      1234,
	}
	
	pusher, err := registry.Pusher(ctx, desc)
	if err != nil {
		t.Errorf("Pusher() error = %v, want nil", err)
	}
	
	if pusher == nil {
		t.Error("expected pusher to be non-nil")
	}
}

func TestOCIRegistry_MarshalAny(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{
		reference: "docker.io/library/hello-world:latest",
		headers:   http.Header{"Authorization": []string{"Bearer token"}},
		hostDir:   "/etc/containerd/certs.d",
	}
	
	streamCreator := &mockStreamCreator{}
	
	any, err := registry.MarshalAny(ctx, streamCreator)
	if err != nil {
		t.Errorf("MarshalAny() error = %v, want nil", err)
	}
	
	if any == nil {
		t.Error("expected any to be non-nil")
	}
}

func TestOCIRegistry_MarshalAnyWithCredentials(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{
		reference: "docker.io/library/hello-world:latest",
		creds:     &mockCredentialHelper{},
	}
	
	streamCreator := &mockStreamCreator{}
	
	any, err := registry.MarshalAny(ctx, streamCreator)
	if err != nil {
		t.Errorf("MarshalAny() with credentials error = %v, want nil", err)
	}
	
	if any == nil {
		t.Error("expected any to be non-nil")
	}
}

func TestOCIRegistry_MarshalAnyWithHTTPDebug(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{
		reference:   "docker.io/library/hello-world:latest",
		httpDebug:   true,
		httpTrace:   true,
		localStream: &mockWriteCloser{},
	}
	
	streamCreator := &mockStreamCreator{}
	
	any, err := registry.MarshalAny(ctx, streamCreator)
	if err != nil {
		t.Errorf("MarshalAny() with HTTP debug error = %v, want nil", err)
	}
	
	if any == nil {
		t.Error("expected any to be non-nil")
	}
}

func TestOCIRegistry_UnmarshalAny(t *testing.T) {
	ctx := context.Background()
	registry := &OCIRegistry{}
	
	resolver := &transfertypes.RegistryResolver{
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
		HostDir:       "/etc/containerd/certs.d",
		DefaultScheme: "http",
	}
	
	ociRegistry := &transfertypes.OCIRegistry{
		Reference: "docker.io/library/hello-world:latest",
		Resolver:  resolver,
	}
	
	any, err := typeurl.MarshalAny(ociRegistry)
	if err != nil {
		t.Fatalf("MarshalAny() setup error = %v", err)
	}
	
	streamGetter := &mockStreamGetter{}
	
	err = registry.UnmarshalAny(ctx, streamGetter, any)
	if err != nil {
		t.Errorf("UnmarshalAny() error = %v, want nil", err)
	}
	
	if registry.reference != "docker.io/library/hello-world:latest" {
		t.Errorf("expected reference to be set")
	}
	
	if registry.headers == nil || registry.headers.Get("Authorization") != "Bearer token" {
		t.Error("expected headers to be set")
	}
}

func TestCredentials(t *testing.T) {
	creds := Credentials{
		Host:     "docker.io",
		Username: "testuser",
		Secret:   "testpass",
		Header:   "Bearer token",
	}
	
	if creds.Host != "docker.io" {
		t.Errorf("expected Host = 'docker.io', got %v", creds.Host)
	}
	
	if creds.Username != "testuser" {
		t.Errorf("expected Username = 'testuser', got %v", creds.Username)
	}
	
	if creds.Secret != "testpass" {
		t.Errorf("expected Secret = 'testpass', got %v", creds.Secret)
	}
	
	if creds.Header != "Bearer token" {
		t.Errorf("expected Header = 'Bearer token', got %v", creds.Header)
	}
}

func TestCredCallback_GetCredentials(t *testing.T) {
	ctx := context.Background()
	stream := &mockStream{}
	
	// Set up expected response
	authResp := &transfertypes.AuthResponse{
		AuthType: transfertypes.AuthType_CREDENTIALS,
		Username: "testuser",
		Secret:   "testpass",
	}
	respAny, _ := typeurl.MarshalAny(authResp)
	stream.recvResponse = respAny
	
	cc := &credCallback{stream: stream}
	
	creds, err := cc.GetCredentials(ctx, "docker.io/library/hello-world:latest", "docker.io")
	if err != nil {
		t.Errorf("GetCredentials() error = %v, want nil", err)
	}
	
	if creds.Host != "docker.io" {
		t.Errorf("expected Host = 'docker.io', got %v", creds.Host)
	}
	
	if creds.Username != "testuser" {
		t.Errorf("expected Username = 'testuser', got %v", creds.Username)
	}
	
	if creds.Secret != "testpass" {
		t.Errorf("expected Secret = 'testpass', got %v", creds.Secret)
	}
}

func TestCredCallback_GetCredentialsHeader(t *testing.T) {
	ctx := context.Background()
	stream := &mockStream{}
	
	// Set up expected response for header auth
	authResp := &transfertypes.AuthResponse{
		AuthType: transfertypes.AuthType_HEADER,
		Secret:   "Bearer token123",
	}
	respAny, _ := typeurl.MarshalAny(authResp)
	stream.recvResponse = respAny
	
	cc := &credCallback{stream: stream}
	
	creds, err := cc.GetCredentials(ctx, "docker.io/library/hello-world:latest", "docker.io")
	if err != nil {
		t.Errorf("GetCredentials() error = %v, want nil", err)
	}
	
	if creds.Header != "Bearer token123" {
		t.Errorf("expected Header = 'Bearer token123', got %v", creds.Header)
	}
}

func TestCredCallback_GetCredentialsRefresh(t *testing.T) {
	ctx := context.Background()
	stream := &mockStream{}
	
	// Set up expected response for refresh token
	authResp := &transfertypes.AuthResponse{
		AuthType: transfertypes.AuthType_REFRESH,
		Secret:   "refresh_token_abc",
	}
	respAny, _ := typeurl.MarshalAny(authResp)
	stream.recvResponse = respAny
	
	cc := &credCallback{stream: stream}
	
	creds, err := cc.GetCredentials(ctx, "docker.io/library/hello-world:latest", "docker.io")
	if err != nil {
		t.Errorf("GetCredentials() error = %v, want nil", err)
	}
	
	if creds.Secret != "refresh_token_abc" {
		t.Errorf("expected Secret = 'refresh_token_abc', got %v", creds.Secret)
	}
}

// Mock implementations for testing

type mockCredentialHelper struct{}

func (m *mockCredentialHelper) GetCredentials(ctx context.Context, ref, host string) (Credentials, error) {
	return Credentials{
		Host:     host,
		Username: "testuser",
		Secret:   "testpass",
	}, nil
}

type mockWriteCloser struct {
	closed bool
	mu     sync.Mutex
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockWriteCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

type mockResolver struct{}

func (m *mockResolver) Resolve(ctx context.Context, ref string) (name string, desc ocispec.Descriptor, err error) {
	return "test-name", ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    "sha256:abc123",
		Size:      1234,
	}, nil
}

func (m *mockResolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	return &mockFetcher{}, nil
}

func (m *mockResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	return &mockPusher{}, nil
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context, desc ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, fmt.Errorf("mock fetcher")
}

type mockPusher struct{}

func (m *mockPusher) Push(ctx context.Context, desc ocispec.Descriptor) (content.Writer, error) {
	return nil, fmt.Errorf("mock pusher")
}

type mockStreamCreator struct{}

func (m *mockStreamCreator) Create(ctx context.Context, id string) (streaming.Stream, error) {
	return &mockStream{}, nil
}

type mockStreamGetter struct{}

func (m *mockStreamGetter) Get(ctx context.Context, id string) (streaming.Stream, error) {
	return &mockStream{}, nil
}

type mockStream struct {
	mu           sync.Mutex
	sentMessages []typeurl.Any
	recvResponse typeurl.Any
}

func (m *mockStream) Send(any typeurl.Any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, any)
	return nil
}

func (m *mockStream) Recv() (typeurl.Any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recvResponse != nil {
		return m.recvResponse, nil
	}
	return nil, fmt.Errorf("no response set")
}

func (m *mockStream) Close() error {
	return nil
}