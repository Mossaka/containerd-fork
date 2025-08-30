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

package shutdown

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWithShutdown(t *testing.T) {
	ctx := context.Background()
	shutdownCtx, service := WithShutdown(ctx)

	if shutdownCtx == nil {
		t.Fatal("WithShutdown returned nil context")
	}

	if service == nil {
		t.Fatal("WithShutdown returned nil service")
	}

	// Check that the service implements the Service interface
	var _ Service = service
}

func TestShutdownService_Basic(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	// Initially should not be done
	select {
	case <-service.Done():
		t.Error("service should not be done initially")
	default:
		// expected
	}

	// Error should be nil initially
	if err := service.Err(); err != nil {
		t.Errorf("initial error should be nil, got %v", err)
	}

	// Shutdown the service
	service.Shutdown()

	// Should be done after shutdown
	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Error("service should be done after shutdown")
	}

	// Error should be ErrShutdown after shutdown
	if err := service.Err(); err != ErrShutdown {
		t.Errorf("error after shutdown should be ErrShutdown, got %v", err)
	}
}

func TestShutdownService_MultipleShutdown(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	var callCount int32

	service.RegisterCallback(func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	// Call shutdown multiple times
	service.Shutdown()
	service.Shutdown()
	service.Shutdown()

	// Wait for completion
	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Callback should only be called once
	if count := atomic.LoadInt32(&callCount); count != 1 {
		t.Errorf("callback should be called once, but was called %d times", count)
	}
}

func TestShutdownService_RegisterCallback(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	var callOrder []int
	var mu sync.Mutex

	// Register multiple callbacks
	for i := 1; i <= 3; i++ {
		i := i // capture loop variable
		service.RegisterCallback(func(ctx context.Context) error {
			mu.Lock()
			callOrder = append(callOrder, i)
			mu.Unlock()
			return nil
		})
	}

	service.Shutdown()

	// Wait for completion
	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(callOrder) != 3 {
		t.Errorf("expected 3 callbacks to be called, got %d", len(callOrder))
	}

	// All callbacks should have been called (order may vary due to concurrency)
	expectedSum := 6 // 1 + 2 + 3
	actualSum := 0
	for _, v := range callOrder {
		actualSum += v
	}
	if actualSum != expectedSum {
		t.Errorf("expected sum %d, got %d", expectedSum, actualSum)
	}
}

func TestShutdownService_CallbackError(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	testError := errors.New("test error")

	// Register a callback that returns an error
	service.RegisterCallback(func(ctx context.Context) error {
		return testError
	})

	service.Shutdown()

	// Wait for completion
	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Error should be the callback error, not ErrShutdown
	if err := service.Err(); err != testError {
		t.Errorf("error should be the callback error %v, got %v", testError, err)
	}
}

func TestShutdownService_MultipleCallbackErrors(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	error1 := errors.New("error 1")
	error2 := errors.New("error 2")

	// Register callbacks that return different errors
	service.RegisterCallback(func(ctx context.Context) error {
		return error1
	})

	service.RegisterCallback(func(ctx context.Context) error {
		return error2
	})

	service.Shutdown()

	// Wait for completion
	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Should return one of the errors (implementation may return first error encountered)
	err := service.Err()
	if err != error1 && err != error2 {
		t.Errorf("error should be one of the callback errors, got %v", err)
	}
}

func TestShutdownService_CallbackTimeout(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	// Temporarily reduce timeout for faster test
	shutdownSvc := service.(*shutdownService)
	shutdownSvc.timeout = 50 * time.Millisecond

	// Register a callback that takes longer than timeout
	service.RegisterCallback(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	service.Shutdown()

	// Wait for completion
	select {
	case <-service.Done():
		// expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Should have a timeout error
	err := service.Err()
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestShutdownService_ConcurrentCallbackRegistration(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	const numGoroutines = 10
	const numCallbacks = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Register callbacks concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numCallbacks; j++ {
				service.RegisterCallback(func(ctx context.Context) error {
					return nil
				})
			}
		}()
	}

	wg.Wait()

	// Shutdown should handle all registered callbacks
	service.Shutdown()

	select {
	case <-service.Done():
		// expected
	case <-time.After(1 * time.Second):
		t.Fatal("service should be done after shutdown")
	}

	if err := service.Err(); err != ErrShutdown {
		t.Errorf("error should be ErrShutdown, got %v", err)
	}
}

func TestShutdownService_CallbackContext(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	var receivedCtx context.Context

	service.RegisterCallback(func(ctx context.Context) error {
		receivedCtx = ctx
		return nil
	})

	service.Shutdown()

	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Check that the callback received a context with timeout
	if receivedCtx == nil {
		t.Error("callback should receive a context")
	}

	if _, ok := receivedCtx.Deadline(); !ok {
		t.Error("callback context should have a deadline")
	}
}

func TestShutdownService_CallbackCancellation(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	// Temporarily reduce timeout for faster test
	shutdownSvc := service.(*shutdownService)
	shutdownSvc.timeout = 50 * time.Millisecond

	var callbackCtx context.Context

	service.RegisterCallback(func(ctx context.Context) error {
		callbackCtx = ctx
		// Wait for context cancellation
		<-ctx.Done()
		return ctx.Err()
	})

	service.Shutdown()

	select {
	case <-service.Done():
		// expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Callback context should be cancelled
	if callbackCtx == nil {
		t.Fatal("callback context should not be nil")
	}

	select {
	case <-callbackCtx.Done():
		// expected - context was cancelled
	default:
		t.Error("callback context should be cancelled")
	}
}

func TestShutdownService_NoCallbacks(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	// Shutdown without registering any callbacks
	service.Shutdown()

	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	if err := service.Err(); err != ErrShutdown {
		t.Errorf("error should be ErrShutdown, got %v", err)
	}
}

func TestShutdownService_RegisterAfterShutdown(t *testing.T) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	service.Shutdown()

	// Wait for shutdown to complete
	select {
	case <-service.Done():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("service should be done after shutdown")
	}

	// Register callback after shutdown - should not cause issues
	service.RegisterCallback(func(ctx context.Context) error {
		t.Error("callback should not be called after shutdown")
		return nil
	})

	// Give some time for any erroneous callback execution
	time.Sleep(50 * time.Millisecond)

	if err := service.Err(); err != ErrShutdown {
		t.Errorf("error should still be ErrShutdown, got %v", err)
	}
}

func TestErrShutdown(t *testing.T) {
	if ErrShutdown == nil {
		t.Error("ErrShutdown should not be nil")
	}

	if ErrShutdown.Error() != "shutdown" {
		t.Errorf("ErrShutdown.Error() should be 'shutdown', got '%s'", ErrShutdown.Error())
	}
}

func BenchmarkShutdownService_RegisterCallback(b *testing.B) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	callback := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.RegisterCallback(callback)
	}
}

func BenchmarkShutdownService_Shutdown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, service := WithShutdown(ctx)

		// Register a simple callback
		service.RegisterCallback(func(ctx context.Context) error {
			return nil
		})

		b.StartTimer()
		service.Shutdown()
		<-service.Done()
		b.StopTimer()
	}
}

func BenchmarkShutdownService_ConcurrentCallbacks(b *testing.B) {
	ctx := context.Background()
	_, service := WithShutdown(ctx)

	// Register many callbacks
	for i := 0; i < 100; i++ {
		service.RegisterCallback(func(ctx context.Context) error {
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, service := WithShutdown(ctx)

		// Register callbacks
		for j := 0; j < 100; j++ {
			service.RegisterCallback(func(ctx context.Context) error {
				return nil
			})
		}

		service.Shutdown()
		<-service.Done()
	}
}