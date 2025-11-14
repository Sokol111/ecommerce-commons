package health

import (
	"sync"
	"time"
)

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
	KubernetesNotifiedAt time.Time         `json:"kubernetes_notified_at,omitempty"` // When K8s first got 200 OK
}

type Readiness interface {
	AddComponent(name string)
	MarkReady(name string)
	IsReady() bool
	getStatus() ReadinessStatus
	notifyKubernetesProbe()  // Called when readiness probe returns 200 OK
	IsKubernetesReady() bool // Check if Kubernetes already knows we're ready
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
	ready                bool
	readyAt              time.Time
	kubernetesNotifiedAt time.Time
}

func newReadiness() *readiness {
	return &readiness{
		components: make(map[string]*component),
	}
}

func (r *readiness) StartWatching() {
	// Check readiness periodically
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			r.mu.Lock()
			if r.ready {
				r.mu.Unlock()
				return
			}

			allReady := true
			for _, comp := range r.components {
				if !comp.ready {
					allReady = false
					break
				}
			}

			if allReady && len(r.components) > 0 {
				r.ready = true
				r.readyAt = time.Now()
			}
			r.mu.Unlock()
		}
	}()
}

func (r *readiness) AddComponent(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[name]; !exists {
		r.components[name] = &component{
			name:      name,
			ready:     false,
			startedAt: time.Now(),
		}
	}
}

func (r *readiness) MarkReady(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if comp, exists := r.components[name]; exists {
		if !comp.ready {
			comp.ready = true
			comp.readyAt = time.Now()
		}
	}
}

func (r *readiness) IsReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ready
}

func (r *readiness) getStatus() ReadinessStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := ReadinessStatus{
		Ready:                r.ready,
		Components:           make([]ComponentStatus, 0, len(r.components)),
		ReadyAt:              r.readyAt,
		KubernetesNotifiedAt: r.kubernetesNotifiedAt,
	}

	for _, comp := range r.components {
		status.Components = append(status.Components, ComponentStatus{
			Name:      comp.name,
			Ready:     comp.ready,
			StartedAt: comp.startedAt,
			ReadyAt:   comp.readyAt,
		})
	}

	return status
}

// NotifyKubernetesProbe records when Kubernetes readiness probe first received 200 OK
func (r *readiness) notifyKubernetesProbe() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Only record the first time
	if r.ready && r.kubernetesNotifiedAt.IsZero() {
		r.kubernetesNotifiedAt = time.Now()
	}
}

// IsKubernetesReady returns true if Kubernetes has been notified about ready status
// This happens when readiness probe first receives 200 OK response
func (r *readiness) IsKubernetesReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return !r.kubernetesNotifiedAt.IsZero()
}
