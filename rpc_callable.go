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

type TimeWheelEntryImpl struct {
	io.Closer
	refCount  int32
	timeWheel *liblpc.TimeWheel
}

// not thread safe
func (this *TimeWheelEntryImpl) BindTimeWheel(timeWheel *liblpc.TimeWheel) {
	this.timeWheel = timeWheel
}

func (this *TimeWheelEntryImpl) NotifyTimeWheel() {
	if this.timeWheel == nil {
		return
	}
	this.timeWheel.Entries() <- this
}

func (this *TimeWheelEntryImpl) GetRefCounter() *int32 {
	return &this.refCount
}

type Callable interface {
	liblpc.BucketEntry
	liblpc.UserDataStorage

	Start()

	Call(timeout time.Duration, name string, mids ...MiddlewareFunc) error
	Call0(timeout time.Duration, name string, headers RpcMsgHeader, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error)

	Call1(timeout time.Duration, name string, in interface{}, mids ...MiddlewareFunc) error
	Call2(timeout time.Duration, name string, headers RpcMsgHeader, in interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error)

	Call3(timeout time.Duration, name string, out interface{}, mids ...MiddlewareFunc) error
	Call4(timeout time.Duration, name string, headers RpcMsgHeader, out interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error)

	Call5(timeout time.Duration, name string, in, out interface{}, mids ...MiddlewareFunc) error
	Call6(timeout time.Duration, name string, headers RpcMsgHeader, in, out interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error)

	Perform(timeout time.Duration, ctx Context)

	SetOnReady(cb CallableCallback)
	SetOnClose(cb CallableCallback)

	BindTimeWheel(timeWheel *liblpc.TimeWheel)
	NotifyTimeWheel()
}

type rpcCallImpl struct {
	stream  *liblpc.BufferedStream
	rpc     *RPC
	readyCb CallableCallback
	closeCb CallableCallback
	middleware
	liblpc.BaseUserData
	TimeWheelEntryImpl
}

func (this *rpcCallImpl) Call(timeout time.Duration, name string, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, nil, nil, mids...)
	return err
}

func (this *rpcCallImpl) Call0(timeout time.Duration, name string, headers RpcMsgHeader, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	return this.Call6(timeout, name, headers, nil, nil, mids...)
}

func (this *rpcCallImpl) Call1(timeout time.Duration, name string, in interface{}, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, in, nil, mids...)
	return err
}

func (this *rpcCallImpl) Call2(timeout time.Duration, name string, headers RpcMsgHeader, in interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	return this.Call6(timeout, name, headers, in, nil, mids...)
}

func (this *rpcCallImpl) Call3(timeout time.Duration, name string, out interface{}, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, nil, out, mids...)
	return err
}

func (this *rpcCallImpl) Call4(timeout time.Duration, name string, headers RpcMsgHeader, out interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	return this.Call6(timeout, name, headers, nil, out, mids...)
}

func (this *rpcCallImpl) Call5(timeout time.Duration, name string, in, out interface{}, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, in, out, mids...)
	return err
}

func (this *rpcCallImpl) Call6(timeout time.Duration, name string, headers RpcMsgHeader, in, out interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
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
	ctx.init(this.stream, this, msg)
	if in != nil {
		ctx.setRequestType(reflect.TypeOf(in))
		ctx.localFnDesc |= ReqHasData
		ctx.SetRequest(in)
	}
	if out != nil {
		// checking
		outValue := reflect.ValueOf(out)
		// check pointer
		std.Assert(outValue.Kind() == reflect.Ptr, "out must be a pointer")
		std.Assert(!outValue.IsNil(), "out underlying ptr must not be nil")
		ctx.localFnDesc |= RspHasData
		ctx.setResponseType(outValue.Type())
	}
	invoke := this.buildInvoke(timeout, ctx, out)
	var handleF HandleFunc = nil
	if len(mids) == 0 && this.middleware.Len() == 0 {
		handleF = this.rpc.buildChain(invoke) // use rpc default chain
	} else if len(mids) == 0 {
		handleF = this.middleware.buildChain(invoke) // use callable itself chain
	} else {
		// if mids not empty ,override callable itself mids
		handleF = middlewareList(mids).build(invoke)
	}
	if this.rpc.preUseMiddleware.Len() != 0 {
		handleF = this.rpc.preUseMiddleware.buildChain(handleF) // prepend preUsed chain
	}
	handleF(ctx)
	return ctx.ResponseHeader(), ctx.Error()
}

func (this *rpcCallImpl) Start() {
	this.NotifyTimeWheel()
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
	if ctx.ackMsg == nil {
		return
	}
	if out == nil {
		//if len(ctx.ackMsg.Data) != 0 {
		//	log.Printf("call [%s]:callee response with data , but caller doesn't specified a out param\n",
		//		ctx.Method())
		//}
		return
	}
	if len(ctx.ackMsg.Data) == 0 {
		log.Printf("call [%s]:callee response data is empty, but caller has out param type:(%T)\n",
			ctx.Method(), out)
		ctx.SetResponse(out)
		return
	}
	err := gRpcSerialization.UnMarshal(ctx.ackMsg.Data, out)
	if err != nil {
		log.Printf("call [%s]:MsgpackUnmarshal got err ->%v\n", ctx.Method(), err)
		if ctx.err != nil {
			ctx.SetError(std.CombinedErrors{ctx.err, err})
		} else {
			ctx.SetError(err)
		}
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
	if writer := ctx.Writer(); writer != nil {
		writer.Write(outBytes, false)
	}
	//wait for data
	future := promise.GetFuture()
	ackMsgObj, err := future.WaitData(timeout)
	if err != nil {
		ctx.SetError(err)
	}
	if ackMsgObj == nil {
		return
	}
	ackMsg, ok := ackMsgObj.(*rpcRawMsg)
	std.Assert(ok, "type mismatched ,rpcRawMsg")
	ctx.ackMsg = ackMsg
	ctx.SetError(ackMsg.GetError())
}

func (this *rpcCallImpl) SetOnReady(cb CallableCallback) {
	this.readyCb = cb
}

func (this *rpcCallImpl) SetOnClose(cb CallableCallback) {
	this.closeCb = cb
}
