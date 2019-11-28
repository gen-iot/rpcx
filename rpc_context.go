package rpcx

import (
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/std"
	"reflect"
)

type RpcMsgHeader = map[string]string

type Writer interface {
	Write(data []byte, inLoop bool)
}

type Context interface {
	Callable() Callable

	Id() string

	SetMethod(string)
	Method() string

	LocalFuncDesc() FuncDesc

	SetRequestHeader(header RpcMsgHeader)
	RequestHeader() RpcMsgHeader

	SetResponseHeader(header RpcMsgHeader)
	ResponseHeader() RpcMsgHeader

	SetRequest(in interface{})
	Request() (in interface{})

	SetResponse(out interface{})
	Response() (out interface{})

	RequestType() reflect.Type
	ResponseType() reflect.Type

	SetError(err error)
	Error() error

	SetWriter(w Writer)
	Writer() Writer

	AddDefer(deferFunc func())

	liblpc.UserDataStorage
}

type contextImpl struct {
	call          Callable
	writer        Writer
	in            interface{}
	inType        reflect.Type
	out           interface{}
	outType       reflect.Type
	err           error
	reqMsg        *rpcRawMsg
	ackMsg        *rpcRawMsg
	localFnDesc   FuncDesc
	deferFuncList []func()
	liblpc.BaseUserData
}

func (this *contextImpl) reset() {
	this.call = nil
	this.in = nil
	this.out = nil
	this.err = nil
	this.reqMsg = nil
	this.ackMsg = nil
	this.localFnDesc = 0
	this.writer = nil
	this.SetUserData(nil)
	for i := len(this.deferFuncList) - 1; i >= 0; i-- {
		this.deferFuncList[i]()
	}
	this.deferFuncList = this.deferFuncList[:0]
}

func (this *contextImpl) AddDefer(deferFunc func()) {
	this.deferFuncList = append(this.deferFuncList, deferFunc)
}

func (this *contextImpl) RequestHeader() map[string]string {
	if this.reqMsg == nil {
		return nil
	}
	return this.reqMsg.Headers
}

func (this *contextImpl) SetRequestHeader(h map[string]string) {
	std.Assert(this.reqMsg != nil, "request is nil")
	this.reqMsg.Headers = h
}

func (this *contextImpl) ResponseHeader() map[string]string {
	if this.ackMsg == nil {
		return nil
	}
	return this.ackMsg.Headers
}

func (this *contextImpl) SetResponseHeader(h map[string]string) {
	std.Assert(this.ackMsg != nil, "response not ready")
	this.ackMsg.Headers = h
}

func (this *contextImpl) Method() string {
	return this.reqMsg.MethodName
}

func (this *contextImpl) SetMethod(method string) {
	this.reqMsg.MethodName = method
}

func (this *contextImpl) Id() string {
	return this.reqMsg.Id
}

func (this *contextImpl) SetRequest(in interface{}) {
	this.in = in
	_ = this.reqMsg.SetData(in)
}

func (this *contextImpl) Request() interface{} {
	return this.in
}

func (this *contextImpl) RequestType() reflect.Type {
	return this.inType
}

func (this *contextImpl) setRequestType(t reflect.Type) {
	this.inType = t
}

func (this *contextImpl) SetResponse(out interface{}) {
	this.out = out
}

func (this *contextImpl) setResponseType(t reflect.Type) {
	this.outType = t
}

func (this *contextImpl) Response() interface{} {
	return this.out
}

func (this *contextImpl) ResponseType() reflect.Type {
	return this.outType
}

func (this *contextImpl) SetError(err error) {
	this.err = err
}

func (this *contextImpl) Error() error {
	return this.err
}

func (this *contextImpl) Callable() Callable {
	return this.call
}

func (this *contextImpl) LocalFuncDesc() FuncDesc {
	return this.localFnDesc
}

func (this *contextImpl) buildOutMsg() (*rpcRawMsg, error) {
	out := this.ackMsg
	serErr := out.SetData(this.out)
	if serErr != nil {
		return nil, serErr
	}
	out.SetError(this.err)
	return out, nil
}

func (this *contextImpl) init(writer Writer, call Callable, inMsg *rpcRawMsg) {
	this.writer = writer
	this.call = call
	this.reqMsg = inMsg
	this.ackMsg = &rpcRawMsg{
		Id:         this.Id(),
		MethodName: this.Method(),
		Type:       rpcAckMsg,
		Headers:    RpcMsgHeader{},
	}
	this.localFnDesc = 0
	if this.deferFuncList != nil {
		this.deferFuncList = make([]func(), 0)
	}
}

func (this *contextImpl) SetWriter(w Writer) {
	this.writer = w
}

func (this *contextImpl) Writer() Writer {
	return this.writer
}
