package middleware

import (
	"github.com/gen-iot/log"
	"github.com/gen-iot/rpcx/v2"
)

// logger , such as log.Error
func ErrorLog(logger log.Logger) rpcx.MiddlewareFunc {
	ctxErrLog := func(ctx rpcx.Context) {
		method := ctx.Method()
		if err := ctx.Error(); err != nil {
			logger.Printf("Method=%s,Error=%v\n", method, err)
		}
	}
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			ctxErrLog(ctx)
			next(ctx)
			ctxErrLog(ctx)
		}
	}
}
