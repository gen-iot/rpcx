package rpcx

import (
	"context"
	"errors"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/std"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
)

//noinspection GoUnusedGlobalVariable
var Debug = true

type RPC struct {
	ioLoop       *liblpc.IOEvtLoop
	rcpFuncMap   map[string]*rpcFunc
	promiseGroup *std.PromiseGroup
	lock         *sync.RWMutex
	startFlag    int32
	middleware
	preUseMiddleware middleware
	ctxPool          sync.Pool
}

const RpcLoopDefaultBufferSize = 1024 * 1024 * 4

func New() (*RPC, error) {
	loop, err := liblpc.NewIOEvtLoop(RpcLoopDefaultBufferSize)
	if err != nil {
		return nil, err
	}
	rpc := &RPC{
		ioLoop:       loop,
		rcpFuncMap:   make(map[string]*rpcFunc),
		promiseGroup: std.NewPromiseGroup(),
		lock:         &sync.RWMutex{},
		startFlag:    0,
	}
	rpc.ctxPool.New = func() interface{} {
		return new(contextImpl)
	}
	return rpc, nil
}

func (this *RPC) grabCtx() *contextImpl {
	ctxImpl := this.ctxPool.Get().(*contextImpl)
	return ctxImpl
}

func (this *RPC) releaseCtx(ctx *contextImpl) {
	std.Assert(ctx != nil, "return ctx is nil")
	this.ctxPool.Put(ctx)
}

func (this *RPC) PreUse(m ...MiddlewareFunc) {
	this.preUseMiddleware.Use(m...)
}

func (this *RPC) Loop() liblpc.EventLoop {
	return this.ioLoop
}
func (this *RPC) getFunc(name string) *rpcFunc {
	this.lock.RLock()
	defer this.lock.RUnlock()
	fn, ok := this.rcpFuncMap[name]
	if !ok {
		return nil
	}
	return fn
}

func (this *RPC) RegFuncWithName(fname string, f interface{}, m ...MiddlewareFunc) {
	fv, ok := f.(reflect.Value)
	if !ok {
		fv = reflect.ValueOf(f)
	}
	std.Assert(fv.Kind() == reflect.Func, "f not func!")
	fvType := fv.Type()
	//check in/out param
	inParamType, inParamDesc := checkInParam(fvType)
	outParamType, outParamDesc := checkOutParam(fvType)
	//
	this.lock.Lock()
	defer this.lock.Unlock()
	//
	fn := &rpcFunc{
		name:           fname,
		fun:            fv,
		inParamType:    inParamType,
		outParamType:   outParamType,
		handleFuncDesc: inParamDesc | outParamDesc,
	}
	fn.mid.Use(m...)
	fn.handleFunc = fn.mid.buildChain(fn.____invoke)
	this.rcpFuncMap[fname] = fn
}

func (this *RPC) RegFunc(f interface{}, m ...MiddlewareFunc) {
	fv, ok := f.(reflect.Value)
	if !ok {
		fv = reflect.ValueOf(f)
	}
	std.Assert(fv.Kind() == reflect.Func, "f not func!")
	fname := getFuncName(fv)
	this.RegFuncWithName(fname, fv, m...)
}

func (this *RPC) Start(ctx context.Context) {
	go this.Run(ctx)
}

func (this *RPC) Run(ctx context.Context) {
	if atomic.CompareAndSwapInt32(&this.startFlag, 0, 1) {
		this.ioLoop.Run(ctx)
	}
}

func (this *RPC) Close() error {
	this.ioLoop.Break()
	return this.ioLoop.Close()
}

func (this *RPC) newCallable(stream *liblpc.BufferedStream, userData interface{}, m []MiddlewareFunc) *rpcCallImpl {
	s := &rpcCallImpl{
		stream: stream,
		rpc:    this,
	}
	s.TimeWheelEntryImpl.Closer = s
	//
	s.Use(m...)
	//
	s.SetUserData(userData)
	s.stream.SetUserData(s)
	//
	return s
}

func (this *RPC) callableClosed(sw liblpc.StreamWriter, err error) {
	log.Println("CALLABLE CLOSED ERR=", err)
	std.CloseIgnoreErr(sw)
	udata := sw.GetUserData()
	if udata == nil {
		return
	}
	if call, ok := udata.(Callable); ok {
		std.CloseIgnoreErr(call)
	}
	return
}

func (this *RPC) NewConnCallable(fd int, userData interface{}, m ...MiddlewareFunc) Callable {
	stream := liblpc.NewBufferedConnStream(this.ioLoop, fd, this.genericRead)
	pCall := this.newCallable(stream, userData, m)
	stream.SetOnConnect(func(sw liblpc.StreamWriter, err error) {
		if pCall.readyCb != nil {
			pCall.readyCb(pCall, err)
		}
		if err != nil {
			std.CloseIgnoreErr(pCall)
		}
	})
	stream.SetOnClose(func(sw liblpc.StreamWriter, err error) {
		if pCall.closeCb != nil {
			pCall.closeCb(pCall, err)
		}
		std.CloseIgnoreErr(pCall)
	})
	stream.SetOnClose(this.callableClosed)
	return pCall
}

type ClientCallableOnConnect func(callable Callable, err error)

func (this *RPC) NewClientCallable(
	addr *liblpc.SyscallSockAddr,
	userData interface{},
	m ...MiddlewareFunc) (Callable, error) {
	fd, err := liblpc.NewConnFd2(addr.Version, addr.Sockaddr)
	if err != nil {
		return nil, err
	}
	stream := liblpc.NewBufferedClientStream(this.ioLoop, int(fd), this.genericRead)
	pCall := this.newCallable(stream, userData, m)
	stream.SetOnConnect(func(sw liblpc.StreamWriter, err error) {
		if pCall.readyCb != nil {
			pCall.readyCb(pCall, err)
		}
		if err != nil {
			std.CloseIgnoreErr(pCall)
		}
	})
	stream.SetOnClose(func(sw liblpc.StreamWriter, err error) {
		if pCall.closeCb != nil {
			pCall.closeCb(pCall, err)
		}
		std.CloseIgnoreErr(pCall)
	})
	stream.SetOnClose(this.callableClosed)
	return pCall, nil
}

const kMaxRpcMsgBodyLen = 1024 * 1024 * 32

func (this *RPC) genericRead(sw liblpc.StreamWriter, buf std.ReadableBuffer) {
	for {
		rawMsg, err := decodeRpcMsg(buf, kMaxRpcMsgBodyLen)
		if err != nil {
			break
		}
		isReq := rawMsg.Type == rpcReqMsg
		call := sw.GetUserData().(Callable)
		call.NotifyTimeWheel()
		if isReq {
			go this.handleReq(call, sw, rawMsg)
		} else {
			this.handleAck(rawMsg)
		}
	}
}

var gRpcSerialization = std.MsgPackSerialization

var errRpcFuncNotFound = errors.New("rpc func not found")

//func (this *RPC) lastWriteFn(outMsg *rpcRawMsg, ctx Context) {
//	err := ctx.Error()
//	if err != nil {
//		outMsg.SetError(err)
//	} else {
//		outBytes, err := gRpcSerialization.Marshal(ctx.Response())
//		if err != nil {
//			outMsg.SetError(err)
//		} else {
//			outMsg.Data = outBytes
//		}
//	}
//}

func (this *RPC) execWithMiddleware(c Context) {
	ctx := c.(*contextImpl)
	fn := this.getFunc(ctx.reqMsg.MethodName)
	var fnProxy HandleFunc = nil
	if fn != nil {
		ctx.setRequestType(fn.inParamType)
		ctx.setResponseType(fn.outParamType)
		inParam, err := fn.decodeInParam(ctx.reqMsg.Data)
		if err != nil {
			ctx.SetError(err)
			return
		} else {
			ctx.SetRequest(inParam)
		}
		fnProxy = fn.handleFunc
		ctx.localFnDesc = fn.handleFuncDesc
	} else {
		ctx.SetError(errRpcFuncNotFound)
		return
	}
	//
	fnProxy = this.buildChain(fnProxy)
	fnProxy(ctx)
}

func (this *RPC) handleReq(cli Callable, sw liblpc.StreamWriter, inMsg *rpcRawMsg) {
	ctx := this.grabCtx()
	defer func() {
		ctx.reset()
		this.releaseCtx(ctx)
	}()
	ctx.init(sw, cli, inMsg)
	//
	proxy := this.execWithMiddleware
	if this.preUseMiddleware.Len() != 0 {
		proxy = this.preUseMiddleware.buildChain(proxy)
	}
	proxy(ctx)
	//
	outMsg, err := ctx.buildOutMsg()
	if err != nil {
		log.Printf("RPC handle REQ Id -> %s,build output msg error -> %v\n", inMsg.Id, err)
		return // build rpcMsg failed
	}
	sendBytes, err := encodeRpcMsg(outMsg)
	if err != nil {
		log.Printf("RPC handle REQ Id -> %s,marshal output msg error -> %v\n", inMsg.Id, err)
		return // encode rpcMsg failed
	}
	if writer := ctx.Writer(); writer != nil {
		writer.Write(sendBytes, false)
	}
}

func (this *RPC) handleAck(inMsg *rpcRawMsg) {
	this.promiseGroup.DonePromise(std.PromiseId(inMsg.Id), inMsg.GetError(), inMsg)
}
