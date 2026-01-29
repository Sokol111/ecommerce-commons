package health

import (
	"context"
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
	KubernetesNotifiedAt time.Time         `json:"kubernetes_notified_at,omitempty"`
}

// ComponentManager manages component registration and readiness tracking.
type ComponentManager interface {
	// AddComponent registers a component and returns a function to mark it as ready.
	AddComponent(name string) func()
}

// ReadinessChecker provides readiness status information.
type ReadinessChecker interface {
	IsReady() bool
	GetStatus() ReadinessStatus
}

// ReadinessWaiter provides methods to wait for readiness events.
type ReadinessWaiter interface {
	WaitReady(ctx context.Context) error           // Wait until all components are ready
	WaitForTrafficReady(ctx context.Context) error // Wait until ready to handle traffic
}

// TrafficController controls when the service is ready to handle traffic.
type TrafficController interface {
	MarkTrafficReady() // Manually mark the service as ready for traffic (for local/testing)
}
