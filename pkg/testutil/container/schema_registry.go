package container

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SchemaRegistryContainer wraps the testcontainers Schema Registry container.
type SchemaRegistryContainer struct {
	Container testcontainers.Container
	URL       string
}

// SchemaRegistryOption configures the Schema Registry container.
type SchemaRegistryOption func(*schemaRegistryOptions)

type schemaRegistryOptions struct {
	image string
}

// WithSchemaRegistryImage sets the Schema Registry image to use.
func WithSchemaRegistryImage(image string) SchemaRegistryOption {
	return func(o *schemaRegistryOptions) {
		o.image = image
	}
}

// StartSchemaRegistryContainer starts a Schema Registry container with an embedded Kafka.
// This uses Redpanda which includes both Kafka and Schema Registry in one container.
func StartSchemaRegistryContainer(ctx context.Context, opts ...SchemaRegistryOption) (*SchemaRegistryContainer, error) {
	options := &schemaRegistryOptions{
		image: "redpandadata/redpanda:v24.1.1",
	}
	for _, opt := range opts {
		opt(options)
	}

	// Start Redpanda container (includes Kafka + Schema Registry)
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        options.image,
			ExposedPorts: []string{"8081/tcp", "9092/tcp"},
			Cmd: []string{
				"redpanda", "start",
				"--mode", "dev-container",
				"--smp", "1",
				"--memory", "512M",
				"--reserve-memory", "0M",
				"--overprovisioned",
				"--node-id", "0",
				"--kafka-addr", "PLAINTEXT://0.0.0.0:9092",
				"--advertise-kafka-addr", "PLAINTEXT://localhost:9092",
				"--schema-registry-addr", "0.0.0.0:8081",
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("8081/tcp"),
				wait.ForListeningPort("9092/tcp"),
			).WithDeadline(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start redpanda container: %w", err)
	}

	// Get Schema Registry URL
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "8081")
	if err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		return nil, fmt.Errorf("failed to get schema registry port: %w", err)
	}

	schemaRegistryURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// Wait for Schema Registry to be ready
	if err := waitForSchemaRegistry(ctx, schemaRegistryURL, 30*time.Second); err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		return nil, fmt.Errorf("schema registry not ready: %w", err)
	}

	return &SchemaRegistryContainer{
		Container: container,
		URL:       schemaRegistryURL,
	}, nil
}

// Terminate terminates the container.
func (s *SchemaRegistryContainer) Terminate(ctx context.Context) error {
	if s.Container != nil {
		return s.Container.Terminate(ctx)
	}
	return nil
}

// KafkaBroker returns the Kafka broker address (useful if you need real Kafka too).
func (s *SchemaRegistryContainer) KafkaBroker(ctx context.Context) (string, error) {
	host, err := s.Container.Host(ctx)
	if err != nil {
		return "", err
	}

	port, err := s.Container.MappedPort(ctx, "9092")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func waitForSchemaRegistry(ctx context.Context, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for schema registry at %s", url)
		default:
			resp, err := client.Get(url + "/subjects")
			if err == nil {
				_ = resp.Body.Close() //nolint:errcheck // best effort cleanup
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
