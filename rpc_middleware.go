package rpcx

import "github.com/gen-iot/std"

type HandleFunc func(ctx Context)

type MiddlewareFunc func(next HandleFunc) HandleFunc

type middlewareList []MiddlewareFunc

func (this middlewareList) build(h HandleFunc) HandleFunc {
	for i := len(this) - 1; i >= 0; i-- {
		h = this[i](h)
	}
	return h
}

//noinspection SpellCheckingInspection
type middleware struct {
	midwares []MiddlewareFunc
}

func (this *middleware) Use(m ...MiddlewareFunc) {
	if this.midwares == nil {
		this.midwares = make([]MiddlewareFunc, 0, 4)
	}
	this.midwares = append(this.midwares, m...)
}

func (this *middleware) Len() int {
	return len(this.midwares)
}

func (this *middleware) buildChain(h HandleFunc) HandleFunc {
	std.Assert(h != nil, "buildMiddleware, h == nil")
	return middlewareList(this.midwares).build(h)
}
