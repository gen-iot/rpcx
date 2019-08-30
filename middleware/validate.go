package middleware

import (
	"github.com/gen-iot/rpcx"
	"github.com/gen-iot/std"
	"log"
)

type ValidateFlag uint8

const (
	ValidateNone ValidateFlag = 0x00
	ValidateIn   ValidateFlag = 0x01
	ValidateOut  ValidateFlag = 0x02
)

func Validate(flag ValidateFlag, v std.Validator) rpcx.MiddlewareFunc {
	std.Assert(v != nil, "validator is nil")
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			if flag&ValidateIn != 0 {
				err := ctx.Error()
				if err != nil {
					log.Println("validate req abort ,caused by:", err)
					return
				}
				if ctx.LocalFuncDesc()&rpcx.ReqHasData != 0 {
					err = v.Validate(ctx.Request())
					if err != nil {
						log.Println("validate req err")
						ctx.SetError(err)
						return
					}
				}
			}
			//
			next(ctx)
			//
			if flag&ValidateOut != 0 {
				err := ctx.Error()
				if err != nil {
					log.Println("validate rsp abort ,caused by:", err)
					return
				}
				if ctx.LocalFuncDesc()&rpcx.RspHasData != 0 {
					err = v.Validate(ctx.Response())
					if err != nil {
						log.Println("validate rsp err")
						ctx.SetError(err)
						return
					}
				}
			}
		}
	}
}
