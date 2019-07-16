package rpcx

import "gitee.com/Puietel/std"

type HandleFunc = func(ctx Context)

type MiddlewareFunc = func(next HandleFunc) HandleFunc

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
	for i := len(this.midwares) - 1; i >= 0; i-- {
		h = this.midwares[i](h)
	}
	return h
}
