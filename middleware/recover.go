package middleware

import (
	"fmt"
	"gitee.com/SuzhenProjects/libgen/rpcx"
	"github.com/pkg/errors"
	"log"
	"runtime"
)

func Recover(allStack bool) rpcx.MiddlewareFunc {
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				ctx.SetError(errors.New(fmt.Sprintf("PANIC:%v", r)))
				stackSize := 1024 * 16
				buf := make([]byte, stackSize)
				stackInfoLen := runtime.Stack(buf, allStack)
				log.Printf("PANIC RECOVERED,PANIC=%v,STACK=%s\n", r, buf[:stackInfoLen])
			}()
			next(ctx)
		}
	}
}
