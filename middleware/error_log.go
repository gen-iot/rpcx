package middleware

import (
	"github.com/gen-iot/log"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/std"
)

// logger , such as log.Error
func ErrorLog(logger log.Logger) rpcx.MiddlewareFunc {
	std.Assert(logger != nil, "logger is nil")
	ctxErrLog := func(ctx rpcx.Context, previousErr error) error {
		method := ctx.Method()
		if err := ctx.Error(); err != nil && previousErr != err {
			logger.Printf("Method=%s,Error=%v\n", method, err)
			return err
		}
		return nil
	}
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			err1 := ctxErrLog(ctx, nil)
			next(ctx)
			_ = ctxErrLog(ctx, err1)
		}
	}
}
