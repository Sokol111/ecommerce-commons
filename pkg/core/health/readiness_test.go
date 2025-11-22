package health

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewReadiness(t *testing.T) {
	logger := zap.NewNop()
	r := newReadiness(logger, false)

	assert.NotNil(t, r)
	assert.False(t, r.IsReady())
}

func TestAddComponent(t *testing.T) {
	t.Run("successfully adds component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		assert.Len(t, r.components, 1)
		assert.Contains(t, r.components, "database")
		assert.False(t, r.components["database"].ready)
	})

	t.Run("panics on empty component name", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		assert.Panics(t, func() {
			r.AddComponent("")
		})
	})

	t.Run("logs warning when adding duplicate component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.AddComponent("database") // Should log warning but not panic

		assert.Len(t, r.components, 1)
	})
}

func TestMarkReady(t *testing.T) {
	t.Run("successfully marks component ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		assert.True(t, r.components["database"].ready)
		assert.False(t, r.components["database"].readyAt.IsZero())
	})

	t.Run("panics on empty component name", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		assert.Panics(t, func() {
			r.MarkReady("")
		})
	})

	t.Run("panics on non-existent component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		assert.PanicsWithValue(t, "readiness: component 'database' does not exist, must call AddComponent first", func() {
			r.MarkReady("database")
		})
	})

	t.Run("handles marking already ready component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")
		firstReadyAt := r.components["database"].readyAt

		r.MarkReady("database") // Should not update readyAt

		assert.Equal(t, firstReadyAt, r.components["database"].readyAt)
	})
}

func TestIsReady(t *testing.T) {
	t.Run("not ready when no components", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		assert.False(t, r.IsReady())
	})

	t.Run("not ready when components exist but not marked ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.AddComponent("cache")

		assert.False(t, r.IsReady())
	})

	t.Run("not ready when only some components are ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.AddComponent("cache")
		r.MarkReady("database")

		assert.False(t, r.IsReady())
	})

	t.Run("ready when all components are ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.AddComponent("cache")
		r.MarkReady("database")
		r.MarkReady("cache")

		assert.True(t, r.IsReady())
	})

	t.Run("ready when single component is ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		assert.True(t, r.IsReady())
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("returns empty status when no components", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		status := r.GetStatus()

		assert.False(t, status.Ready)
		assert.Empty(t, status.Components)
		assert.True(t, status.ReadyAt.IsZero())
		assert.True(t, status.KubernetesNotifiedAt.IsZero())
	})

	t.Run("returns components in sorted order", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("zookeeper")
		r.AddComponent("database")
		r.AddComponent("cache")

		status := r.GetStatus()

		require.Len(t, status.Components, 3)
		assert.Equal(t, "cache", status.Components[0].Name)
		assert.Equal(t, "database", status.Components[1].Name)
		assert.Equal(t, "zookeeper", status.Components[2].Name)
	})

	t.Run("returns correct ready status", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.AddComponent("cache")

		status := r.GetStatus()
		assert.False(t, status.Ready)

		r.MarkReady("database")
		status = r.GetStatus()
		assert.False(t, status.Ready)

		r.MarkReady("cache")
		status = r.GetStatus()
		assert.True(t, status.Ready)
		assert.False(t, status.ReadyAt.IsZero())
	})

	t.Run("readyAt is the latest component readyAt", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		time.Sleep(10 * time.Millisecond) // Ensure different timestamps

		r.AddComponent("cache")
		r.MarkReady("cache")

		status := r.GetStatus()

		assert.True(t, status.Ready)
		// ReadyAt should be the latest (max) of all component readyAt times
		var maxComponentReadyAt time.Time
		for _, comp := range status.Components {
			if comp.ReadyAt.After(maxComponentReadyAt) {
				maxComponentReadyAt = comp.ReadyAt
			}
		}
		assert.Equal(t, maxComponentReadyAt, status.ReadyAt)
	})
}

func TestWaitReady(t *testing.T) {
	t.Run("blocks until ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		ready := make(chan struct{})
		go func() {
			ctx := context.Background()
			err := r.WaitReady(ctx)
			assert.NoError(t, err)
			close(ready)
		}()

		// Should not be ready yet
		select {
		case <-ready:
			t.Fatal("should not be ready yet")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		r.MarkReady("database")

		// Should become ready
		select {
		case <-ready:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be ready now")
		}
	})

	t.Run("returns immediately when already ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := r.WaitReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error when context cancelled", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan error, 1)
		go func() {
			done <- r.WaitReady(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
			assert.Equal(t, context.Canceled, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitReady should return immediately with cancelled context")
		}
	})

	t.Run("returns error when context times out", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := r.WaitReady(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestMarkTrafficReady(t *testing.T) {
	t.Run("marks traffic ready in local mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		r.MarkTrafficReady()

		status := r.GetStatus()
		assert.False(t, status.KubernetesNotifiedAt.IsZero())
	})

	t.Run("marks traffic ready in kubernetes mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, true)

		r.AddComponent("database")
		r.MarkReady("database")

		r.MarkTrafficReady()

		status := r.GetStatus()
		assert.False(t, status.KubernetesNotifiedAt.IsZero())
	})

	t.Run("marks traffic ready only once", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		r.MarkTrafficReady()
		firstNotifiedAt := r.GetStatus().KubernetesNotifiedAt

		time.Sleep(10 * time.Millisecond)
		r.MarkTrafficReady()
		secondNotifiedAt := r.GetStatus().KubernetesNotifiedAt

		assert.Equal(t, firstNotifiedAt, secondNotifiedAt)
	})
}

func TestWaitForTrafficReady(t *testing.T) {
	t.Run("waits in local mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		ready := make(chan struct{})
		go func() {
			ctx := context.Background()
			err := r.WaitForTrafficReady(ctx)
			assert.NoError(t, err)
			close(ready)
		}()

		// Should not be ready yet
		select {
		case <-ready:
			t.Fatal("should not be ready yet")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		r.MarkReady("database")

		// Should become ready immediately in local mode
		select {
		case <-ready:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be ready now")
		}
	})

	t.Run("waits for traffic ready in kubernetes mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, true)

		r.AddComponent("database")

		ready := make(chan struct{})
		go func() {
			ctx := context.Background()
			err := r.WaitForTrafficReady(ctx)
			assert.NoError(t, err)
			close(ready)
		}()

		// Should not be ready yet
		select {
		case <-ready:
			t.Fatal("should not be ready yet")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		r.MarkReady("database")

		// Still should not be ready - waiting for traffic ready signal
		select {
		case <-ready:
			t.Fatal("should not be ready yet - waiting for traffic ready")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		r.MarkTrafficReady()

		// Should become ready now
		select {
		case <-ready:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be ready now")
		}
	})

	t.Run("returns immediately when already ready in local mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := r.WaitForTrafficReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns immediately when already ready in kubernetes mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, true)

		r.AddComponent("database")
		r.MarkReady("database")
		r.MarkTrafficReady()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := r.WaitForTrafficReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error when context cancelled before ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := r.WaitForTrafficReady(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("returns error when context cancelled before traffic ready in kubernetes mode", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, true)

		r.AddComponent("database")
		r.MarkReady("database")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := r.WaitForTrafficReady(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestWaitKubernetesReady(t *testing.T) {
	t.Run("blocks until kubernetes notified", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.MarkReady("database")

		ready := make(chan struct{})
		go func() {
			ctx := context.Background()
			err := r.WaitForTrafficReady(ctx)
			assert.NoError(t, err)
			close(ready)
		}()

		// In local mode, should be ready immediately after components are ready
		select {
		case <-ready:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be ready in local mode")
		}
	})

	t.Run("returns immediately when already notified", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, true)

		r.AddComponent("database")
		r.MarkReady("database")
		r.MarkTrafficReady()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := r.WaitForTrafficReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error when context cancelled", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, true)

		r.AddComponent("database")
		r.MarkReady("database")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan error, 1)
		go func() {
			done <- r.WaitForTrafficReady(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
			assert.Equal(t, context.Canceled, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitForTrafficReady should return immediately with cancelled context")
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("component lifecycle timing", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		startTime := time.Now()
		r.AddComponent("database")

		time.Sleep(50 * time.Millisecond)
		r.MarkReady("database")

		status := r.GetStatus()
		require.Len(t, status.Components, 1)

		comp := status.Components[0]
		assert.True(t, comp.StartedAt.After(startTime) || comp.StartedAt.Equal(startTime))
		assert.True(t, comp.ReadyAt.After(comp.StartedAt))
		duration := comp.ReadyAt.Sub(comp.StartedAt)
		assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
	})

	t.Run("multiple WaitReady calls", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		var wg sync.WaitGroup
		var mu sync.Mutex
		readyCount := 0

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()
				err := r.WaitReady(ctx)
				if err == nil {
					mu.Lock()
					readyCount++
					mu.Unlock()
				}
			}()
		}

		time.Sleep(50 * time.Millisecond)
		r.MarkReady("database")

		wg.Wait()
		assert.Equal(t, 10, readyCount)
	})

	t.Run("GetStatus with zero-value times", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")

		status := r.GetStatus()
		assert.False(t, status.Ready)
		assert.True(t, status.ReadyAt.IsZero()) // Should be zero when not ready
		require.Len(t, status.Components, 1)
		assert.False(t, status.Components[0].StartedAt.IsZero())
		assert.True(t, status.Components[0].ReadyAt.IsZero()) // Not marked ready yet
	})

	t.Run("concurrent component operations", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		var wg sync.WaitGroup
		componentCount := 10

		// Add and mark components ready concurrently
		for i := 0; i < componentCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				name := time.Now().Format("comp-150405.000000000-") + string(rune('A'+idx))
				r.AddComponent(name)
				time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
				r.MarkReady(name)
			}(i)
		}

		wg.Wait()

		// Verify all components are ready
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := r.WaitReady(ctx)
		assert.NoError(t, err)
		assert.True(t, r.IsReady())

		status := r.GetStatus()
		assert.Len(t, status.Components, componentCount)
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent AddComponent and MarkReady", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		componentCount := 100
		var wg sync.WaitGroup

		// Add components concurrently with unique names
		for i := 0; i < componentCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				name := time.Now().Format("component-150405.000000000-") + string(rune('A'+idx/26)) + string(rune('A'+idx%26))
				r.AddComponent(name)
				r.MarkReady(name)
			}(i)
		}

		wg.Wait()

		// Wait for all to be ready
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := r.WaitReady(ctx)
		assert.NoError(t, err)
		assert.True(t, r.IsReady())
		status := r.GetStatus()
		assert.Len(t, status.Components, componentCount)
	})

	t.Run("concurrent GetStatus calls", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger, false)

		r.AddComponent("database")
		r.AddComponent("cache")

		done := make(chan struct{})
		for i := 0; i < 50; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					status := r.GetStatus()
					assert.Len(t, status.Components, 2)
				}
				done <- struct{}{}
			}()
		}

		// Collect all goroutines with timeout
		timeout := time.After(5 * time.Second)
		for i := 0; i < 50; i++ {
			select {
			case <-done:
				// Expected
			case <-timeout:
				t.Fatal("timeout waiting for concurrent GetStatus calls to complete")
			}
		}
	})
}
