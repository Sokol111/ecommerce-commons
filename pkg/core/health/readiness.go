package health

import (
	"context"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

type component struct {
	name      string
	ready     bool
	startedAt time.Time
	readyAt   time.Time
}

type readiness struct {
	mu                   sync.RWMutex
	components           map[string]*component
	readyChan            chan struct{}
	readyOnce            sync.Once
	kubernetesReadyChan  chan struct{}
	kubernetesReadyOnce  sync.Once
	kubernetesNotifiedAt time.Time
	isKubernetes         bool
	logger               *zap.Logger
}

func newReadiness(logger *zap.Logger, isKubernetes bool) *readiness {
	r := &readiness{
		components:          make(map[string]*component),
		readyChan:           make(chan struct{}),
		kubernetesReadyChan: make(chan struct{}),
		isKubernetes:        isKubernetes,
		logger:              logger,
	}

	// In local mode, automatically mark as ready for traffic
	if !isKubernetes {
		logger.Info("Running in local mode - traffic readiness will be automatic")
	}

	return r
}

func (r *readiness) AddComponent(name string) {
	if name == "" {
		panic("readiness: component name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[name]; !exists {
		r.components[name] = &component{
			name:      name,
			ready:     false,
			startedAt: time.Now(),
		}
		r.logger.Debug("Component added",
			zap.String("component", name),
		)
	} else {
		r.logger.Warn("Component already exists",
			zap.String("component", name),
		)
	}
}

func (r *readiness) MarkReady(name string) {
	if name == "" {
		panic("readiness: component name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	comp, exists := r.components[name]
	if !exists {
		panic("readiness: component '" + name + "' does not exist, must call AddComponent first")
	}

	if comp.ready {
		r.logger.Debug("Component already marked as ready",
			zap.String("component", name),
		)
		return
	}

	comp.ready = true
	comp.readyAt = time.Now()
	duration := comp.readyAt.Sub(comp.startedAt)

	r.logger.Info("Component ready",
		zap.String("component", name),
		zap.Duration("initialization_time", duration),
	)

	// Check if all components are ready
	allReady := len(r.components) > 0
	for _, c := range r.components {
		if !c.ready {
			allReady = false
			break
		}
	}

	if allReady {
		r.readyOnce.Do(func() {
			close(r.readyChan)
			r.logger.Info("All components are ready",
				zap.Int("component_count", len(r.components)),
				zap.Time("ready_at", time.Now()),
			)
		})
	}
}

func (r *readiness) IsReady() bool {
	select {
	case <-r.readyChan:
		return true
	default:
		return false
	}
}

func (r *readiness) GetStatus() ReadinessStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ready := r.IsReady()
	var readyAt time.Time
	if ready && len(r.components) > 0 {
		for _, comp := range r.components {
			if comp.readyAt.After(readyAt) {
				readyAt = comp.readyAt
			}
		}
	}

	status := ReadinessStatus{
		Ready:                ready,
		Components:           make([]ComponentStatus, 0, len(r.components)),
		ReadyAt:              readyAt,
		KubernetesNotifiedAt: r.kubernetesNotifiedAt,
	}

	// Collect and sort components by name for deterministic output
	for _, comp := range r.components {
		status.Components = append(status.Components, ComponentStatus{
			Name:      comp.name,
			Ready:     comp.ready,
			StartedAt: comp.startedAt,
			ReadyAt:   comp.readyAt,
		})
	}

	sort.Slice(status.Components, func(i, j int) bool {
		return status.Components[i].Name < status.Components[j].Name
	})

	return status
}

// WaitReady blocks until all components are ready or context is cancelled.
func (r *readiness) WaitReady(ctx context.Context) error {
	select {
	case <-r.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *readiness) WaitForTrafficReady(ctx context.Context) error {
	// First wait for all components to be ready
	if err := r.WaitReady(ctx); err != nil {
		return err
	}

	// In local mode, we're ready to handle traffic immediately
	if !r.isKubernetes {
		r.logger.Debug("Local mode - ready for traffic immediately")
		return nil
	}

	// In Kubernetes mode, wait for the readiness probe to confirm
	r.logger.Info("Kubernetes mode - waiting for readiness probe confirmation")
	select {
	case <-r.kubernetesReadyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *readiness) MarkTrafficReady() {
	// Check if all components are ready first
	if !r.IsReady() {
		r.logger.Warn("Attempted to mark traffic ready before all components are ready - ignoring")
		return
	}

	select {
	case <-r.kubernetesReadyChan:
		return
	default:
	}

	r.kubernetesReadyOnce.Do(func() {
		r.mu.Lock()
		r.kubernetesNotifiedAt = time.Now()
		r.mu.Unlock()
		close(r.kubernetesReadyChan)
		r.logger.Info("Service is now ready to receive traffic (Kubernetes readiness probe passed)",
			zap.Bool("kubernetes_mode", r.isKubernetes),
		)
	})
}
