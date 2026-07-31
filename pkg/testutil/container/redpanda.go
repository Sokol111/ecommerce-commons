package container

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedpandaContainer wraps the testcontainers Redpanda container.
type RedpandaContainer struct {
	container         testcontainers.Container
	SchemaRegistryURL string
	KafkaBroker       string
}

// RedpandaOption configures the Redpanda container.
type RedpandaOption func(*redpandaOptions)

type redpandaOptions struct {
	image string
}

// WithRedpandaImage sets the Redpanda image to use.
func WithRedpandaImage(image string) RedpandaOption {
	return func(o *redpandaOptions) {
		o.image = image
	}
}

func StartDefaultRedpandaContainer() *RedpandaContainer {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return StartRedpandaContainer(ctx)
}

// StartRedpandaContainer starts a Redpanda container with an embedded Kafka and Schema Registry.
func StartRedpandaContainer(ctx context.Context, opts ...RedpandaOption) *RedpandaContainer {
	options := &redpandaOptions{
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
		panic(fmt.Errorf("failed to start redpanda container: %w", err))
	}

	// Get host URL
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		panic(fmt.Errorf("failed to get container host: %w", err))
	}

	kafkaPort, err := container.MappedPort(ctx, "9092")
	if err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		panic(fmt.Errorf("failed to get kafka port: %w", err))
	}

	schemaRegistryPort, err := container.MappedPort(ctx, "8081")
	if err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		panic(fmt.Errorf("failed to get schema registry port: %w", err))
	}

	schemaRegistryURL := fmt.Sprintf("http://%s:%s", host, schemaRegistryPort.Port())
	kafkaBroker := fmt.Sprintf("%s:%s", host, kafkaPort.Port())

	// Wait for Redpanda to be ready
	if err := waitForRedpanda(ctx, schemaRegistryURL, 30*time.Second); err != nil {
		_ = container.Terminate(ctx) //nolint:errcheck // best effort cleanup
		panic(fmt.Errorf("redpanda not ready: %w", err))
	}

	return &RedpandaContainer{
		container:         container,
		SchemaRegistryURL: schemaRegistryURL,
		KafkaBroker:       kafkaBroker,
	}
}

// Terminate terminates the container.
func (s *RedpandaContainer) Terminate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if s.container != nil {
		return s.container.Terminate(ctx)
	}
	return nil
}

func waitForRedpanda(ctx context.Context, url string, timeout time.Duration) error {
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
