package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewReadiness(t *testing.T) {
	logger := zap.NewNop()
	r := newReadiness(logger)

	assert.NotNil(t, r)
	assert.False(t, r.IsReady())
	assert.False(t, r.IsKubernetesReady())
}

func TestAddComponent(t *testing.T) {
	t.Run("successfully adds component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger).(*readiness)

		r.AddComponent("database")

		assert.Len(t, r.components, 1)
		assert.Contains(t, r.components, "database")
		assert.False(t, r.components["database"].ready)
	})

	t.Run("panics on empty component name", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		assert.Panics(t, func() {
			r.AddComponent("")
		})
	})

	t.Run("logs warning when adding duplicate component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger).(*readiness)

		r.AddComponent("database")
		r.AddComponent("database") // Should log warning but not panic

		assert.Len(t, r.components, 1)
	})
}

func TestMarkReady(t *testing.T) {
	t.Run("successfully marks component ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger).(*readiness)

		r.AddComponent("database")
		r.MarkReady("database")

		assert.True(t, r.components["database"].ready)
		assert.False(t, r.components["database"].readyAt.IsZero())
	})

	t.Run("panics on empty component name", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		assert.Panics(t, func() {
			r.MarkReady("")
		})
	})

	t.Run("panics on non-existent component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		assert.PanicsWithValue(t, "readiness: component 'database' does not exist, must call AddComponent first", func() {
			r.MarkReady("database")
		})
	})

	t.Run("handles marking already ready component", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger).(*readiness)

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
		r := newReadiness(logger)

		assert.False(t, r.IsReady())
	})

	t.Run("not ready when components exist but not marked ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.AddComponent("cache")

		assert.False(t, r.IsReady())
	})

	t.Run("not ready when only some components are ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.AddComponent("cache")
		r.MarkReady("database")

		assert.False(t, r.IsReady())
	})

	t.Run("ready when all components are ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.AddComponent("cache")
		r.MarkReady("database")
		r.MarkReady("cache")

		assert.True(t, r.IsReady())
	})

	t.Run("ready when single component is ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		assert.True(t, r.IsReady())
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("returns empty status when no components", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		status := r.GetStatus()

		assert.False(t, status.Ready)
		assert.Empty(t, status.Components)
		assert.True(t, status.ReadyAt.IsZero())
		assert.True(t, status.KubernetesNotifiedAt.IsZero())
	})

	t.Run("returns components in sorted order", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

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
		r := newReadiness(logger)

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
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		r.AddComponent("cache")
		r.MarkReady("cache")

		status := r.GetStatus()

		assert.True(t, status.Ready)
		// ReadyAt should be the latest (max) of all component readyAt times
		for _, comp := range status.Components {
			assert.True(t, status.ReadyAt.After(comp.ReadyAt) || status.ReadyAt.Equal(comp.ReadyAt))
		}
	})
}

func TestNotifyKubernetesProbe(t *testing.T) {
	t.Run("does not notify when not ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.NotifyKubernetesProbe()

		assert.False(t, r.IsKubernetesReady())
		status := r.GetStatus()
		assert.True(t, status.KubernetesNotifiedAt.IsZero())
	})

	t.Run("notifies when ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		assert.True(t, r.IsReady())
		assert.False(t, r.IsKubernetesReady())

		r.NotifyKubernetesProbe()

		assert.True(t, r.IsKubernetesReady())
		status := r.GetStatus()
		assert.False(t, status.KubernetesNotifiedAt.IsZero())
	})

	t.Run("notifies only once", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		r.NotifyKubernetesProbe()
		firstNotifiedAt := r.GetStatus().KubernetesNotifiedAt

		r.NotifyKubernetesProbe()
		secondNotifiedAt := r.GetStatus().KubernetesNotifiedAt

		assert.Equal(t, firstNotifiedAt, secondNotifiedAt)
	})
}

func TestWaitReady(t *testing.T) {
	t.Run("blocks until ready", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

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
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := r.WaitReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error when context cancelled", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

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
		r := newReadiness(logger)

		r.AddComponent("database")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := r.WaitReady(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestWaitKubernetesReady(t *testing.T) {
	t.Run("blocks until kubernetes notified", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		ready := make(chan struct{})
		go func() {
			ctx := context.Background()
			err := r.WaitKubernetesReady(ctx)
			assert.NoError(t, err)
			close(ready)
		}()

		// Should not be notified yet
		select {
		case <-ready:
			t.Fatal("should not be notified yet")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}

		r.NotifyKubernetesProbe()

		// Should become notified
		select {
		case <-ready:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be notified now")
		}
	})

	t.Run("returns immediately when already notified", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")
		r.NotifyKubernetesProbe()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := r.WaitKubernetesReady(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns error when context cancelled", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		r.AddComponent("database")
		r.MarkReady("database")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan error, 1)
		go func() {
			done <- r.WaitKubernetesReady(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
			assert.Equal(t, context.Canceled, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitKubernetesReady should return immediately with cancelled context")
		}
	})
}

func TestConcurrency(t *testing.T) {
	t.Run("concurrent AddComponent and MarkReady", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

		componentCount := 100

		// Add components concurrently
		for i := 0; i < componentCount; i++ {
			go func(idx int) {
				name := time.Now().Format("component-2006-01-02-15:04:05.000000000") + string(rune(idx))
				r.AddComponent(name)
				r.MarkReady(name)
			}(i)
		}

		// Wait for all to be ready
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := r.WaitReady(ctx)
		assert.NoError(t, err)
		assert.True(t, r.IsReady())
	})

	t.Run("concurrent GetStatus calls", func(t *testing.T) {
		logger := zap.NewNop()
		r := newReadiness(logger)

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
