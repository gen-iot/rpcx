package middleware

import (
	"errors"
	"github.com/gen-iot/rpcx"
	"reflect"
)

var errRequestNil = errors.New("request must not be nil")

func RequestNotNil() rpcx.MiddlewareFunc {
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			reqInterface := ctx.Request()
			reqV := reflect.ValueOf(reqInterface)
			k := reqV.Kind()
			switch k {
			case reflect.Invalid:
				goto error
			case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
				if reqV.IsNil() {
					goto error
				}
			}
			goto next
		error:
			ctx.SetError(errRequestNil)
			return
		next:
			next(ctx)
		}
	}
}
