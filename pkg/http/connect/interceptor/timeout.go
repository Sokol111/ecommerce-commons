package interceptor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
)

func NewTimeoutInterceptor(timeout time.Duration) connect.Interceptor {
	return connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				ctx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				resp, err := next(ctx, req)

				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return resp, connect.NewError(connect.CodeDeadlineExceeded, errors.New("request timeout"))
				}

				return resp, err
			}
		},
	)
}
