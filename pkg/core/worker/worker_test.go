package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// mockReadinessWaiter is a mock implementation of health.ReadinessWaiter
type mockReadinessWaiter struct {
	readyChan        chan struct{}
	trafficReadyChan chan struct{}
}

func newMockReadinessWaiter() *mockReadinessWaiter {
	return &mockReadinessWaiter{
		readyChan:        make(chan struct{}),
		trafficReadyChan: make(chan struct{}),
	}
}

func (m *mockReadinessWaiter) WaitReady(ctx context.Context) error {
	select {
	case <-m.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *mockReadinessWaiter) WaitForTrafficReady(ctx context.Context) error {
	select {
	case <-m.trafficReadyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *mockReadinessWaiter) MarkReady() {
	select {
	case <-m.readyChan:
	default:
		close(m.readyChan)
	}
}

func (m *mockReadinessWaiter) MarkTrafficReady() {
	select {
	case <-m.trafficReadyChan:
	default:
		close(m.trafficReadyChan)
	}
}

// mockShutdowner is a mock implementation of fx.Shutdowner
type mockShutdowner struct {
	shutdownCalled atomic.Bool
	exitCode       int
	mu             sync.Mutex
}

func (m *mockShutdowner) Shutdown(opts ...fx.ShutdownOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shutdownCalled.Store(true)
	// Note: fx.ShutdownOption is opaque, we can't easily extract exit code
	// For testing purposes, we just mark that shutdown was called
	m.exitCode = 1
	return nil
}

func (m *mockShutdowner) WasShutdownCalled() bool {
	return m.shutdownCalled.Load()
}

func (m *mockShutdowner) GetExitCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitCode
}

// Test Options
func TestOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := Options{}
		assert.False(t, opts.WaitForTrafficReady)
		assert.False(t, opts.WaitReady)
		assert.False(t, opts.ShutdownOnError)
	})

	t.Run("WithTrafficReady", func(t *testing.T) {
		opts := Options{}
		WithTrafficReady()(&opts)
		assert.True(t, opts.WaitForTrafficReady)
	})

	t.Run("WithReady", func(t *testing.T) {
		opts := Options{}
		WithReady()(&opts)
		assert.True(t, opts.WaitReady)
	})

	t.Run("WithShutdown", func(t *testing.T) {
		opts := Options{}
		WithShutdown()(&opts)
		assert.True(t, opts.ShutdownOnError)
	})

	t.Run("multiple options", func(t *testing.T) {
		opts := Options{}
		WithTrafficReady()(&opts)
		WithReady()(&opts)
		WithShutdown()(&opts)

		assert.True(t, opts.WaitForTrafficReady)
		assert.True(t, opts.WaitReady)
		assert.True(t, opts.ShutdownOnError)
	})
}

// Test baseWorker Start
func TestBaseWorker_Start(t *testing.T) {
	t.Run("starts worker and runs function", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()

		executed := make(chan struct{})
		runFunc := func(ctx context.Context) error {
			close(executed)
			<-ctx.Done()
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{},
		}

		w.Start()

		select {
		case <-executed:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("run function was not executed")
		}

		w.Stop(context.Background())
	})

	t.Run("creates cancellable context", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()

		ctxReceived := make(chan context.Context, 1)
		runFunc := func(ctx context.Context) error {
			ctxReceived <- ctx
			<-ctx.Done()
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{},
		}

		w.Start()

		var ctx context.Context
		select {
		case ctx = <-ctxReceived:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("did not receive context")
		}

		assert.NotNil(t, ctx)
		assert.Nil(t, ctx.Err())

		w.Stop(context.Background())

		// Context should be cancelled after stop
		assert.NotNil(t, ctx.Err())
	})
}

// Test baseWorker Stop
func TestBaseWorker_Stop(t *testing.T) {
	t.Run("stops worker gracefully", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()

		stopped := make(chan struct{})
		runFunc := func(ctx context.Context) error {
			<-ctx.Done()
			close(stopped)
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{},
		}

		w.Start()
		time.Sleep(50 * time.Millisecond) // Let worker start

		w.Stop(context.Background())

		select {
		case <-stopped:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("worker did not stop")
		}
	})

	t.Run("respects context timeout", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()

		// Worker that ignores cancellation
		runFunc := func(ctx context.Context) error {
			time.Sleep(5 * time.Second) // Long running task
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{},
		}

		w.Start()
		time.Sleep(50 * time.Millisecond)

		// Stop with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		w.Stop(ctx)
		elapsed := time.Since(start)

		// Should return quickly due to timeout
		assert.Less(t, elapsed, 200*time.Millisecond)
	})

	t.Run("handles nil cancelFunc", func(t *testing.T) {
		logger := zap.NewNop()

		w := &baseWorker{
			name:       "test-worker",
			log:        logger,
			cancelFunc: nil,
		}

		// Should not panic
		assert.NotPanics(t, func() {
			w.Stop(context.Background())
		})
	})
}

// Test WaitReady option
func TestBaseWorker_WaitReady(t *testing.T) {
	t.Run("waits for readiness before running", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()

		var order []string
		var mu sync.Mutex

		runFunc := func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "run")
			mu.Unlock()
			<-ctx.Done()
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{WaitReady: true},
		}

		w.Start()

		// Give some time for worker to potentially run (it shouldn't)
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Empty(t, order, "run function should not have executed yet")
		mu.Unlock()

		// Mark ready
		readiness.MarkReady()

		// Wait for run to be called
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Contains(t, order, "run")
		mu.Unlock()

		w.Stop(context.Background())
	})

	t.Run("stops if cancelled while waiting for readiness", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()

		executed := atomic.Bool{}
		runFunc := func(ctx context.Context) error {
			executed.Store(true)
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{WaitReady: true},
		}

		w.Start()
		time.Sleep(50 * time.Millisecond)

		w.Stop(context.Background())

		assert.False(t, executed.Load(), "run function should not have executed")
	})
}

// Test WaitForTrafficReady option
func TestBaseWorker_WaitForTrafficReady(t *testing.T) {
	t.Run("waits for traffic readiness before running", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()

		executed := atomic.Bool{}
		runFunc := func(ctx context.Context) error {
			executed.Store(true)
			<-ctx.Done()
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{WaitForTrafficReady: true},
		}

		w.Start()
		time.Sleep(50 * time.Millisecond)

		assert.False(t, executed.Load(), "run function should not have executed yet")

		readiness.MarkTrafficReady()
		time.Sleep(50 * time.Millisecond)

		assert.True(t, executed.Load(), "run function should have executed")

		w.Stop(context.Background())
	})

	t.Run("stops if cancelled while waiting for traffic readiness", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()

		executed := atomic.Bool{}
		runFunc := func(ctx context.Context) error {
			executed.Store(true)
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{WaitForTrafficReady: true},
		}

		w.Start()
		time.Sleep(50 * time.Millisecond)

		w.Stop(context.Background())

		assert.False(t, executed.Load(), "run function should not have executed")
	})
}

// Test ShutdownOnError option
func TestBaseWorker_ShutdownOnError(t *testing.T) {
	t.Run("triggers shutdown on error", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()
		shutdowner := &mockShutdowner{}

		runFunc := func(ctx context.Context) error {
			return errors.New("fatal error")
		}

		w := &baseWorker{
			name:       "test-worker",
			log:        logger,
			runFunc:    runFunc,
			shutdowner: shutdowner,
			readiness:  readiness,
			options:    Options{ShutdownOnError: true},
		}

		w.Start()
		time.Sleep(100 * time.Millisecond)

		assert.True(t, shutdowner.WasShutdownCalled())
		assert.Equal(t, 1, shutdowner.GetExitCode())
	})

	t.Run("does not trigger shutdown when no error", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()
		shutdowner := &mockShutdowner{}

		runFunc := func(ctx context.Context) error {
			return nil
		}

		w := &baseWorker{
			name:       "test-worker",
			log:        logger,
			runFunc:    runFunc,
			shutdowner: shutdowner,
			readiness:  readiness,
			options:    Options{ShutdownOnError: true},
		}

		w.Start()
		time.Sleep(100 * time.Millisecond)

		assert.False(t, shutdowner.WasShutdownCalled())
	})

	t.Run("logs error but does not shutdown when ShutdownOnError is false", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()
		shutdowner := &mockShutdowner{}

		runFunc := func(ctx context.Context) error {
			return errors.New("non-fatal error")
		}

		w := &baseWorker{
			name:       "test-worker",
			log:        logger,
			runFunc:    runFunc,
			shutdowner: shutdowner,
			readiness:  readiness,
			options:    Options{ShutdownOnError: false},
		}

		w.Start()
		time.Sleep(100 * time.Millisecond)

		assert.False(t, shutdowner.WasShutdownCalled())
	})
}

// Test combined options
func TestBaseWorker_CombinedOptions(t *testing.T) {
	t.Run("WaitReady and WaitForTrafficReady", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()

		var order []string
		var mu sync.Mutex

		runFunc := func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "run")
			mu.Unlock()
			<-ctx.Done()
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{WaitReady: true, WaitForTrafficReady: true},
		}

		w.Start()
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Empty(t, order)
		mu.Unlock()

		// Only mark ready, not traffic ready
		readiness.MarkReady()
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Empty(t, order, "should still wait for traffic ready")
		mu.Unlock()

		// Now mark traffic ready
		readiness.MarkTrafficReady()
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Contains(t, order, "run")
		mu.Unlock()

		w.Stop(context.Background())
	})
}

// Test registerWorker
func TestRegisterWorker(t *testing.T) {
	t.Run("registers start and stop hooks", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()

		startCalled := atomic.Bool{}
		stopCalled := atomic.Bool{}

		runFunc := func(ctx context.Context) error {
			startCalled.Store(true)
			<-ctx.Done()
			stopCalled.Store(true)
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{},
		}

		lc := &mockLifecycle{}
		registerWorker(lc, w)

		require.Len(t, lc.hooks, 1)

		// Trigger OnStart
		err := lc.hooks[0].OnStart(context.Background())
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		assert.True(t, startCalled.Load())

		// Trigger OnStop
		err = lc.hooks[0].OnStop(context.Background())
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		assert.True(t, stopCalled.Load())
	})
}

// mockLifecycle is a mock implementation of fx.Lifecycle
type mockLifecycle struct {
	hooks []fx.Hook
}

func (m *mockLifecycle) Append(hook fx.Hook) {
	m.hooks = append(m.hooks, hook)
}

// Test concurrent operations
func TestBaseWorker_Concurrency(t *testing.T) {
	t.Run("multiple start/stop cycles", func(t *testing.T) {
		logger := zap.NewNop()
		readiness := newMockReadinessWaiter()
		readiness.MarkReady()
		readiness.MarkTrafficReady()

		counter := atomic.Int32{}
		runFunc := func(ctx context.Context) error {
			counter.Add(1)
			<-ctx.Done()
			return nil
		}

		w := &baseWorker{
			name:      "test-worker",
			log:       logger,
			runFunc:   runFunc,
			readiness: readiness,
			options:   Options{},
		}

		for i := 0; i < 5; i++ {
			w.Start()
			time.Sleep(20 * time.Millisecond)
			w.Stop(context.Background())
		}

		assert.Equal(t, int32(5), counter.Load())
	})
}
