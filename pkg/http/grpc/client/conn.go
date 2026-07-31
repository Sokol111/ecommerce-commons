package client

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewGrpcConn(cfg Config) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			TenantSlugUnaryInterceptor,
			TimeoutUnaryInterceptor(cfg.Timeout),
		),
	}

	return newConn(cfg, opts)
}

// NewGrpcConnWithTokenSource creates a grpc.ClientConnInterface with h2c support, optional Bearer auth,
// and lifecycle-managed shutdown. It is intended for internal M2M calls.
func NewGrpcConnWithTokenSource(cfg Config, tokenSource oauth2.TokenSource) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			TenantSlugUnaryInterceptor,
			TimeoutUnaryInterceptor(cfg.Timeout),
		),
		grpc.WithPerRPCCredentials(&tokenSourceCreds{src: tokenSource}),
	}

	return newConn(cfg, opts)
}

func newConn(cfg Config, opts []grpc.DialOption) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc new client: %w", err)
	}
	return conn, nil
}

// --- per-RPC auth credentials ---

type tokenSourceCreds struct {
	src oauth2.TokenSource
}

func (c *tokenSourceCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	tok, err := c.src.Token()
	if err != nil {
		return nil, err
	}
	return map[string]string{"authorization": "Bearer " + tok.AccessToken}, nil
}

func (c *tokenSourceCreds) RequireTransportSecurity() bool { return false }
