package rpcx

import (
	"gitee.com/gen-iot/liblpc"
	"gitee.com/gen-iot/std"
	"io"
	"log"
	"reflect"
	"time"
)

type CallableCallback func(callable Callable, err error)

type Callable interface {
	io.Closer
	liblpc.UserDataStorage

	Start()

	Call1(timeout time.Duration, name string, param interface{}, out interface{}) error
	Call2(timeout time.Duration, name string, out interface{}) error
	Call3(timeout time.Duration, name string) error

	Call4(timeout time.Duration, name string, headers map[string]string, param interface{}, out interface{}) error
	Call5(timeout time.Duration, name string, headers map[string]string, out interface{}) error
	Call6(timeout time.Duration, name string, headers map[string]string) error

	Perform(timeout time.Duration, ctx Context)

	SetOnReady(cb CallableCallback)
	SetOnClose(cb CallableCallback)
}

type rpcCallImpl struct {
	stream  *liblpc.BufferedStream
	rpc     *RPC
	readyCb CallableCallback
	closeCb CallableCallback
	middleware
	liblpc.BaseUserData
}

func (this *rpcCallImpl) Call1(timeout time.Duration, name string, param interface{}, out interface{}) error {
	return this.Call4(timeout, name, nil, param, out)
}

func (this *rpcCallImpl) Call2(timeout time.Duration, name string, out interface{}) error {
	return this.Call5(timeout, name, nil, out)
}

func (this *rpcCallImpl) Call3(timeout time.Duration, name string) error {
	return this.Call6(timeout, name, nil)
}

func (this *rpcCallImpl) Call4(timeout time.Duration, name string, headers map[string]string, param interface{}, out interface{}) error {
	std.Assert(this.stream != nil, "stream is nil!")
	msgId := std.GenRandomUUID()
	msg := &rpcRawMsg{
		Id:         msgId,
		MethodName: name,
		Headers:    headers,
		Type:       rpcReqMsg,
	}
	//add promise
	ctx := this.rpc.grabCtx()
	defer func() {
		ctx.reset()
		this.rpc.releaseCtx(ctx)
	}()
	ctx.init(this, msg)
	ctx.SetRequest(param)
	f := this.buildInvoke(timeout, ctx, out)
	h := this.buildChain(f)
	h(ctx)
	return ctx.Error()
}

func (this *rpcCallImpl) Call5(timeout time.Duration, name string, headers map[string]string, out interface{}) error {
	return this.Call4(timeout, name, headers, nil, out)
}

func (this *rpcCallImpl) Call6(timeout time.Duration, name string, headers map[string]string) error {
	return this.Call4(timeout, name, headers, nil, nil)
}

func (this *rpcCallImpl) Start() {
	this.stream.Start()
}

func (this *rpcCallImpl) Close() error {
	return this.stream.Close()
}

func (this *rpcCallImpl) buildInvoke(timeout time.Duration, ctx *contextImpl, out interface{}) HandleFunc {
	return func(Context) {
		this.invoke(timeout, out, ctx)
	}
}

func (this *rpcCallImpl) invoke(timeout time.Duration, out interface{}, ctx *contextImpl) {
	this.Perform(timeout, ctx)
	if ctx.Error() != nil || /*never happen*/ ctx.ackMsg /*never happen*/ == nil {
		return
	}
	if out == nil {
		return
	}
	outValue := reflect.ValueOf(out)
	std.Assert(outValue.Kind() == reflect.Ptr, "out must be a pointer")
	if outValue.IsNil() {
		return
	}
	err := std.MsgpackUnmarshal(ctx.ackMsg.Data, out)
	if err != nil {
		log.Println("call :MsgpackUnmarshal got err ->", err)
		ctx.SetError(err)
	}
	ctx.SetResponse(out)
}

func (this *rpcCallImpl) Perform(timeout time.Duration, c Context) {
	ctx := c.(*contextImpl)
	err := ctx.reqMsg.SetData(ctx.in)
	if err != nil {
		ctx.SetError(err)
		return
	}
	promise := std.NewPromise()
	promiseId := std.PromiseId(ctx.Id())
	//write out
	outBytes, err := encodeRpcMsg(ctx.reqMsg)
	if err != nil {
		ctx.SetError(err)
		return
	}
	this.rpc.promiseGroup.AddPromise(promiseId, promise)
	defer this.rpc.promiseGroup.RemovePromise(promiseId)
	//
	this.stream.Write(outBytes, false)
	//wait for data
	future := promise.GetFuture()
	ackMsgObj, err := future.WaitData(timeout)
	if err != nil {
		log.Println("call :future wait got err ->", err)
		ctx.SetError(err)
		return
	}
	ackMsg, ok := ackMsgObj.(*rpcRawMsg)
	std.Assert(ok, "type mismatched ,rpcRawMsg")
	ctx.ackMsg = ackMsg
}

func (this *rpcCallImpl) SetOnReady(cb CallableCallback) {
	this.readyCb = cb
}

func (this *rpcCallImpl) SetOnClose(cb CallableCallback) {
	this.closeCb = cb
}
