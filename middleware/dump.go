package middleware

import (
	"github.com/gen-iot/rpcx/v2"
	"log"
	"time"
)

func Dump() rpcx.MiddlewareFunc {
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			t1 := time.Now()
			log.Printf("\n\n=============REQ=================\nREQ ID = %s\nMETHOD = %s\nTIME:%s\nERR = %v\nREQ = %v\n=============REQ=================\n\n",
				ctx.Id(),
				ctx.Method(),
				t1,
				ctx.Error(),
				ctx.Request())
			//
			next(ctx)
			//
			t2 := time.Now()
			log.Printf("\n\n=============ACK=================\nACK ID = %s\nMETHOD = %s\nCOST:%s\nERR = %v\nRSP = %v\n=============ACK=================\n\n",
				ctx.Id(),
				ctx.Method(),
				t2.Sub(t1),
				ctx.Error(),
				ctx.Response())
		}
	}
}
