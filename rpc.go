package rpcx

import (
	"context"
	"errors"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/std"
	"io"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
)

//noinspection GoUnusedGlobalVariable
var Debug = true
var gRpcSerialization = std.MsgPackSerialization

type Core interface {
	Loop() *liblpc.IOEvtLoop
	RegFunc(f interface{}, m ...MiddlewareFunc)
	RegFuncWithName(fname string, f interface{}, m ...MiddlewareFunc)
	PreUse(m ...MiddlewareFunc)
	Use(m ...MiddlewareFunc)
	BuildChain(h HandleFunc) HandleFunc
	BuildPreUsedChain(h HandleFunc) HandleFunc
	Run(ctx context.Context)
	Start(ctx context.Context)
	GrabContext() Context
	ReleaseContext(c Context)
	PromiseGroup() *std.PromiseGroup
	NotifyCallableRead(call Callable, buf std.ReadableBuffer)
	io.Closer
}

type coreImpl struct {
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

func New() (core Core, err error) {
	loop, err := liblpc.NewIOEvtLoop(RpcLoopDefaultBufferSize)
	if err != nil {
		return nil, err
	}
	rpc := &coreImpl{
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

func (this *coreImpl) GrabContext() Context {
	ctxImpl := this.ctxPool.Get().(*contextImpl)
	return ctxImpl
}

func (this *coreImpl) ReleaseContext(ctx Context) {
	std.Assert(ctx != nil, "return ctx is nil")
	this.ctxPool.Put(ctx)
}

func (this *coreImpl) PromiseGroup() *std.PromiseGroup {
	return this.promiseGroup
}

func (this *coreImpl) PreUse(m ...MiddlewareFunc) {
	this.preUseMiddleware.Use(m...)
}

func (this *coreImpl) Loop() *liblpc.IOEvtLoop {
	return this.ioLoop
}
func (this *coreImpl) getFunc(name string) *rpcFunc {
	this.lock.RLock()
	defer this.lock.RUnlock()
	fn, ok := this.rcpFuncMap[name]
	if !ok {
		return nil
	}
	return fn
}

func (this *coreImpl) RegFuncWithName(fname string, f interface{}, m ...MiddlewareFunc) {
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

func (this *coreImpl) RegFunc(f interface{}, m ...MiddlewareFunc) {
	fv, ok := f.(reflect.Value)
	if !ok {
		fv = reflect.ValueOf(f)
	}
	std.Assert(fv.Kind() == reflect.Func, "f not func!")
	fname := getFuncName(fv)
	this.RegFuncWithName(fname, fv, m...)
}

func (this *coreImpl) Start(ctx context.Context) {
	go this.Run(ctx)
}

func (this *coreImpl) Run(ctx context.Context) {
	if atomic.CompareAndSwapInt32(&this.startFlag, 0, 1) {
		this.ioLoop.Run(ctx)
	}
}

func (this *coreImpl) Close() error {
	this.ioLoop.Break()
	return this.ioLoop.Close()
}

const kMaxRpcMsgBodyLen = 1024 * 1024 * 32

func (this *coreImpl) NotifyCallableRead(call Callable, buf std.ReadableBuffer) {
	for {
		rawMsg, err := decodeRpcMsg(buf, kMaxRpcMsgBodyLen)
		if err != nil {
			break
		}
		isReq := rawMsg.Type == ReqMsg
		call.NotifyTimeWheel()
		if isReq {
			go this.handleReq(call, rawMsg)
		} else {
			this.handleAck(rawMsg)
		}
	}
}

var errRpcFuncNotFound = errors.New("core func not found")

func (this *coreImpl) execWithMiddleware(c Context) {
	ctx := c.(*contextImpl)
	fn := this.getFunc(ctx.reqMsg.MethodName)
	var fnProxy HandleFunc = nil
	if fn != nil {
		ctx.SetRequestType(fn.inParamType)
		ctx.SetResponseType(fn.outParamType)
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

func (this *coreImpl) handleReq(cli Callable, inMsg *RawMsg) {
	ctx := this.GrabContext()
	defer func() {
		ctx.Reset()
		this.ReleaseContext(ctx)
	}()
	ctx.Init(cli, inMsg)
	//
	proxy := this.execWithMiddleware
	if this.preUseMiddleware.Len() != 0 {
		proxy = this.preUseMiddleware.buildChain(proxy)
	}
	proxy(ctx)
	//
	outMsg, err := ctx.BuildOutMsg()
	if err != nil {
		log.Printf("coreImpl handle REQ Id -> %s,build output msg error -> %v\n", inMsg.Id, err)
		return // build rpcMsg failed
	}
	sendBytes, err := encodeRpcMsg(outMsg)
	if err != nil {
		log.Printf("coreImpl handle REQ Id -> %s,marshal output msg error -> %v\n", inMsg.Id, err)
		return // encode rpcMsg failed
	}
	if writer := ctx.Writer(); writer != nil {
		writer.Write(ctx, sendBytes, false)
	}
}

func (this *coreImpl) handleAck(inMsg *RawMsg) {
	this.promiseGroup.DonePromise(std.PromiseId(inMsg.Id), inMsg.GetError(), inMsg)
}

func (this *coreImpl) BuildChain(h HandleFunc) HandleFunc {
	return this.buildChain(h)
}

func (this *coreImpl) BuildPreUsedChain(h HandleFunc) HandleFunc {
	return this.preUseMiddleware.buildChain(h)
}
