package grpcclient

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ConnParams holds dependencies for constructing a gRPC connection via FX.
type ConnParams struct {
	fx.In
	Log         *zap.Logger
	TokenSource oauth2.TokenSource `optional:"true"`
}

// NewConn creates a grpc.ClientConnInterface with h2c support, optional Bearer auth,
// and lifecycle-managed shutdown. It is intended for internal M2M calls.
func NewConn(cfg Config, p ConnParams, lc fx.Lifecycle) (grpc.ClientConnInterface, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			unaryInterceptor,
			timeoutUnaryInterceptor(cfg.Timeout),
		),
	}

	if p.TokenSource != nil {
		opts = append(opts, grpc.WithPerRPCCredentials(&tokenSourceCreds{src: p.TokenSource}))
	}

	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc new client: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})

	p.Log.Info("gRPC client connected", zap.String("address", cfg.Address))
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

// timeoutUnaryInterceptor wraps RPC calls with a context timeout.
func timeoutUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
