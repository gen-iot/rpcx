package rpcx

import (
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/std"
	"io"
	"log"
	"reflect"
	"time"
)

type RpcMsgHeader = map[string]string

type Writer interface {
	Write(ctx Context, data []byte, inLoop bool)
}

type WriterCloser interface {
	io.Closer
	Writer
}

type CallableCallbacks struct {
	delegate Callable
	readyCb  CallableCallback // 剥离 callbacks
	closeCb  CallableCallback // 剥离 callbacks
}

func (this *CallableCallbacks) DoReady(err error) {
	if this.readyCb != nil {
		this.readyCb(this.delegate, err)
	}
}

func (this *CallableCallbacks) DoClosed(err error) {
	if this.closeCb != nil {
		this.closeCb(this.delegate, err)
	}
}

func (this *CallableCallbacks) SetOnReady(cb CallableCallback) {
	this.readyCb = cb
}

func (this *CallableCallbacks) SetOnClose(cb CallableCallback) {
	this.closeCb = cb
}

type BaseCallable struct {
	core   Core
	writer WriterCloser
	CallableCallbacks
	middleware
	liblpc.BaseUserData
	TimeWheelEntryImpl
}

func NewBaseCallable(core Core, writer WriterCloser, driven Callable) *BaseCallable {
	bCall := &BaseCallable{
		core:   core,
		writer: writer,
	}
	if driven == nil {
		driven = bCall
	}
	bCall.delegate = driven
	return bCall
}

func (this *BaseCallable) Call(timeout time.Duration, name string, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, nil, nil, mids...)
	return err
}

func (this *BaseCallable) Call0(timeout time.Duration, name string, headers RpcMsgHeader, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	return this.Call6(timeout, name, headers, nil, nil, mids...)
}

func (this *BaseCallable) Call1(timeout time.Duration, name string, in interface{}, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, in, nil, mids...)
	return err
}

func (this *BaseCallable) Call2(timeout time.Duration, name string, headers RpcMsgHeader, in interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	return this.Call6(timeout, name, headers, in, nil, mids...)
}

func (this *BaseCallable) Call3(timeout time.Duration, name string, out interface{}, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, nil, out, mids...)
	return err
}

func (this *BaseCallable) Call4(timeout time.Duration, name string, headers RpcMsgHeader, out interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	return this.Call6(timeout, name, headers, nil, out, mids...)
}

func (this *BaseCallable) Call5(timeout time.Duration, name string, in, out interface{}, mids ...MiddlewareFunc) error {
	_, err := this.Call6(timeout, name, nil, in, out, mids...)
	return err
}

func (this *BaseCallable) Call6(timeout time.Duration, name string, headers RpcMsgHeader, in, out interface{}, mids ...MiddlewareFunc) (ackHeader RpcMsgHeader, err error) {
	std.Assert(this.writer != nil, "stream is nil!")
	msgId := std.GenRandomUUID()
	msg := &RawMsg{
		Id:         msgId,
		MethodName: name,
		Headers:    headers,
		Type:       ReqMsg,
	}
	//add promise
	ctx := this.core.GrabContext()
	defer func() {
		ctx.Reset()
		this.core.ReleaseContext(ctx)
	}()
	ctx.Init(this, msg)
	if in != nil {
		ctx.SetRequestType(reflect.TypeOf(in))
		ctx.SetFuncDesc(ctx.FuncDesc() | ReqHasData)
		ctx.SetRequest(in)
	}
	if out != nil {
		// checking
		outValue := reflect.ValueOf(out)
		// check pointer
		std.Assert(outValue.Kind() == reflect.Ptr, "out must be a pointer")
		std.Assert(!outValue.IsNil(), "out underlying ptr must not be nil")
		ctx.SetFuncDesc(ctx.FuncDesc() | RspHasData)
		ctx.SetResponseType(outValue.Type())
	}
	invoke := this.buildInvoke(timeout, ctx, out)
	var handleF HandleFunc = nil
	if len(mids) == 0 && this.middleware.Len() == 0 {
		handleF = this.core.BuildChain(invoke) // use core default chain
	} else if len(mids) == 0 {
		handleF = this.middleware.buildChain(invoke) // use callable itself chain
	} else {
		// if mids not empty ,override callable itself mids
		handleF = middlewareList(mids).build(invoke)
	}
	handleF = this.core.BuildPreUsedChain(handleF) // prepend preUsed chain
	handleF(ctx)
	return ctx.ResponseHeader(), ctx.Error()
}

func (this *BaseCallable) Start() {
	this.NotifyTimeWheel()
}

func (this *BaseCallable) Close() error {
	if this.writer != nil {
		return this.writer.Close()
	}
	return nil
}

func (this *BaseCallable) buildInvoke(timeout time.Duration, ctx Context, out interface{}) HandleFunc {
	return func(Context) {
		this.____invoke(timeout, out, ctx)
	}
}

// dont call this func directly
func (this *BaseCallable) ____invoke(timeout time.Duration, out interface{}, ctx Context) {
	this.Perform(timeout, ctx)
	if ctx.AckMsg() == nil {
		return
	}
	if out == nil {
		//if len(ctx.ackMsg.Data) != 0 {
		//	log.Printf("call [%s]:callee response with data , but caller doesn't specified a out param\n",
		//		ctx.Method())
		//}
		return
	}
	if len(ctx.AckMsg().Data) == 0 {
		log.Printf("call [%s]:callee response data is empty, but caller has out param type:(%T)\n",
			ctx.Method(), out)
		ctx.SetResponse(out)
		return
	}
	err := gRpcSerialization.UnMarshal(ctx.AckMsg().Data, out)
	if err != nil {
		log.Printf("call [%s]:MsgpackUnmarshal got err ->%v\n", ctx.Method(), err)
		if ctx.Error() != nil {
			ctx.SetError(std.CombinedErrors{ctx.Error(), err})
		} else {
			ctx.SetError(err)
		}
	}
	ctx.SetResponse(out)
}

func (this *BaseCallable) Perform(timeout time.Duration, c Context) {
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
	this.core.PromiseGroup().AddPromise(promiseId, promise)
	defer this.core.PromiseGroup().RemovePromise(promiseId)
	//
	if writer := ctx.Writer(); writer != nil {
		writer.Write(ctx, outBytes, false)
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
	ackMsg, ok := ackMsgObj.(*RawMsg)
	std.Assert(ok, "type mismatched ,RawMsg")
	ctx.ackMsg = ackMsg
	ctx.SetError(ackMsg.GetError())
}

func (this *BaseCallable) Writer() Writer {
	return this.writer
}
