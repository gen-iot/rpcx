package rpcx

import (
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/std"
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

	Call(timeout time.Duration, name string) error
	Call0(timeout time.Duration, name string, headers map[string]string) error

	Call1(timeout time.Duration, name string, in interface{}) error
	Call2(timeout time.Duration, name string, headers map[string]string, in interface{}) error

	Call3(timeout time.Duration, name string, out interface{}) error
	Call4(timeout time.Duration, name string, headers map[string]string, out interface{}) error

	Call5(timeout time.Duration, name string, in, out interface{}) error
	Call6(timeout time.Duration, name string, headers map[string]string, in, out interface{}) error

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

func (this *rpcCallImpl) Call(timeout time.Duration, name string) error {
	return this.Call6(timeout, name, nil, nil, nil)
}

func (this *rpcCallImpl) Call0(timeout time.Duration, name string, headers map[string]string) error {
	return this.Call6(timeout, name, headers, nil, nil)
}

func (this *rpcCallImpl) Call1(timeout time.Duration, name string, in interface{}) error {
	return this.Call6(timeout, name, nil, in, nil)
}

func (this *rpcCallImpl) Call2(timeout time.Duration, name string, headers map[string]string, in interface{}) error {
	return this.Call6(timeout, name, headers, in, nil)
}

func (this *rpcCallImpl) Call3(timeout time.Duration, name string, out interface{}) error {
	return this.Call6(timeout, name, nil, nil, out)
}

func (this *rpcCallImpl) Call4(timeout time.Duration, name string, headers map[string]string, out interface{}) error {
	return this.Call6(timeout, name, headers, nil, out)
}

func (this *rpcCallImpl) Call5(timeout time.Duration, name string, in, out interface{}) error {
	return this.Call6(timeout, name, nil, in, out)
}

func (this *rpcCallImpl) Call6(timeout time.Duration, name string, headers map[string]string, in, out interface{}) error {
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
	if in != nil {
		ctx.localFnDesc |= ReqHasData
	}
	if out != nil {
		// checking
		outValue := reflect.ValueOf(out)
		// check pointer
		std.Assert(outValue.Kind() == reflect.Ptr, "out must be a pointer")
		std.Assert(!outValue.IsNil(), "out underlying ptr must not be nil")
		ctx.localFnDesc |= RspHasData
	}
	ctx.SetRequest(in)
	f := this.buildInvoke(timeout, ctx, out)
	h := this.buildChain(f)
	h(ctx)
	return ctx.Error()
}

func (this *rpcCallImpl) Start() {
	this.stream.Start()
}

func (this *rpcCallImpl) Close() error {
	return this.stream.Close()
}

func (this *rpcCallImpl) buildInvoke(timeout time.Duration, ctx *contextImpl, out interface{}) HandleFunc {
	return func(Context) {
		this.____invoke(timeout, out, ctx)
	}
}

// dont call this func directly
func (this *rpcCallImpl) ____invoke(timeout time.Duration, out interface{}, ctx *contextImpl) {
	this.Perform(timeout, ctx)
	if ctx.Error() != nil || /*never happen*/ ctx.ackMsg /*never happen*/ == nil {
		return
	}
	if out == nil {
		return
	}
	if len(ctx.ackMsg.Data) == 0 {
		log.Printf("call :callee response data is empty, but caller has out param type:(%T)\n", out)
		ctx.SetResponse(out)
		return
	}
	err := gRpcSerialization.UnMarshal(ctx.ackMsg.Data, out)
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
	std.Assert(ackMsgObj != nil, "ackMsg must not be nil")
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
