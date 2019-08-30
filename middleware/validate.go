package middleware

import (
	"github.com/gen-iot/rpcx"
	"github.com/gen-iot/std"
	"log"
	"reflect"
)

type ValidateFlag uint8

const (
	ValidateNone  ValidateFlag = 0x00
	ValidateIn    ValidateFlag = 0x01
	ValidateOut   ValidateFlag = 0x02
	ValidateInOut              = ValidateIn | ValidateOut
)

// only validate struct , other values will ignore
func ValidateStruct(flag ValidateFlag, v std.Validator) rpcx.MiddlewareFunc {
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
					reqT := ctx.RequestType()
					if reqT.Kind() == reflect.Ptr {
						reqT = reqT.Elem()
					}
					if reqT.Kind() == reflect.Struct {
						err = v.Validate(ctx.Request())
						if err != nil {
							log.Println("validate req err")
							ctx.SetError(err)
							return
						}
					}
				}
			}
			//
			next(ctx)
			//
			if flag&ValidateOut == 0 {
				return
			}
			err := ctx.Error()
			if err != nil {
				log.Println("validate rsp abort ,caused by:", err)
				return
			}
			if ctx.LocalFuncDesc()&rpcx.RspHasData != 0 {
				rspT := ctx.ResponseType()
				if rspT.Kind() == reflect.Ptr {
					rspT = rspT.Elem()
				}
				if rspT.Kind() != reflect.Struct {
					return
				}
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
