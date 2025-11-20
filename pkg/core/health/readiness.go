package health

import (
	"context"
	"sort"
	"sync"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewReadinessModule() fx.Option {
	return fx.Provide(newReadiness)
}

type ComponentStatus struct {
	Name      string    `json:"name"`
	Ready     bool      `json:"ready"`
	StartedAt time.Time `json:"started_at"`
	ReadyAt   time.Time `json:"ready_at,omitempty"`
}

type ReadinessStatus struct {
	Ready                bool              `json:"ready"`
	Components           []ComponentStatus `json:"components"`
	ReadyAt              time.Time         `json:"ready_at,omitempty"`
	KubernetesNotifiedAt time.Time         `json:"kubernetes_notified_at,omitempty"`
}

type Readiness interface {
	AddComponent(name string)
	MarkReady(name string)
	IsReady() bool
	GetStatus() ReadinessStatus
	NotifyKubernetesProbe()                        // Called when readiness probe returns 200 OK
	IsKubernetesReady() bool                       // Check if Kubernetes already knows we're ready
	WaitReady(ctx context.Context) error           // Wait until all components are ready
	WaitKubernetesReady(ctx context.Context) error // Wait until Kubernetes has been notified
}

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
	logger               *zap.Logger
}

func newReadiness(logger *zap.Logger) Readiness {
	return &readiness{
		components:          make(map[string]*component),
		readyChan:           make(chan struct{}),
		kubernetesReadyChan: make(chan struct{}),
		logger:              logger,
	}
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

// NotifyKubernetesProbe records when Kubernetes readiness probe first received 200 OK
func (r *readiness) NotifyKubernetesProbe() {
	// Fast path: check if already notified
	if r.IsKubernetesReady() {
		return
	}

	// Only notify if we're ready
	if !r.IsReady() {
		return
	}

	// Close channel once to signal Kubernetes notification
	r.kubernetesReadyOnce.Do(func() {
		r.mu.Lock()
		r.kubernetesNotifiedAt = time.Now()
		r.mu.Unlock()
		close(r.kubernetesReadyChan)
		r.logger.Info("Kubernetes readiness probe notified",
			zap.Time("notified_at", r.kubernetesNotifiedAt),
		)
	})
}

// IsKubernetesReady returns true if Kubernetes has been notified about ready status
// This happens when readiness probe first receives 200 OK response
func (r *readiness) IsKubernetesReady() bool {
	select {
	case <-r.kubernetesReadyChan:
		return true
	default:
		return false
	}
}

// WaitReady blocks until all components are ready or context is cancelled
func (r *readiness) WaitReady(ctx context.Context) error {
	select {
	case <-r.readyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitKubernetesReady blocks until Kubernetes has been notified about readiness or context is cancelled
func (r *readiness) WaitKubernetesReady(ctx context.Context) error {
	select {
	case <-r.kubernetesReadyChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
