package interceptor

import (
	"context"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ValidationModule provides a protovalidate interceptor that validates
// incoming request messages against constraints defined in .proto files.
// Recommended priority: 25 (after logger, before timeout).
func ValidationModule(priority int) fx.Option {
	return fx.Provide(
		fx.Annotate(
			func(log *zap.Logger) (Interceptor, error) {
				validator, err := protovalidate.New()
				if err != nil {
					return Interceptor{}, err
				}
				log.Info("Connect validation interceptor initialized")
				return Interceptor{
					Priority: priority,
					Handler:  newValidationInterceptor(validator),
				}, nil
			},
			fx.ResultTags(`group:"connect_interceptor"`),
		),
	)
}

func newValidationInterceptor(v protovalidate.Validator) connect.Interceptor {
	return connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				if msg, ok := req.Any().(proto.Message); ok {
					if err := v.Validate(msg); err != nil {
						return nil, connect.NewError(connect.CodeInvalidArgument, err)
					}
				}
				return next(ctx, req)
			}
		},
	)
}
