package grpcclient

import (
	"context"

	"github.com/Sokol111/ecommerce-commons/pkg/core/logger"
	"github.com/Sokol111/ecommerce-commons/pkg/tenant"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// unaryInterceptor propagates tenant slug into outgoing gRPC metadata
// and enriches the logger with grpc_method for downstream logs.
func unaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if slug, ok := tenant.SlugFromContext(ctx); ok {
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(tenant.TenantSlugHeader, slug)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	if log := logger.Get(ctx); log != nil {
		ctx = logger.With(ctx, log.With(zap.String("grpc_method", method)))
	}

	return invoker(ctx, method, req, reply, cc, opts...)
}
