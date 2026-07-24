package interceptor

import (
	"context"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
)

func NewValidationInterceptor(v protovalidate.Validator) connect.Interceptor {
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
