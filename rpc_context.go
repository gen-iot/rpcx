package rpcx

import (
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/std"
	"reflect"
)

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
	SetRequestType(t reflect.Type)

	ResponseType() reflect.Type
	SetResponseType(t reflect.Type)

	ReqMsg() *RawMsg
	SetReqMsg(msg *RawMsg)

	AckMsg() *RawMsg
	SetAckMsg(msg *RawMsg)

	FuncDesc() FuncDesc
	SetFuncDesc(desc FuncDesc)

	SetError(err error)
	Error() error

	SetWriter(w Writer)
	Writer() Writer

	AddDefer(deferFunc func())

	Reset()

	Init(call Callable, inMsg *RawMsg)

	BuildOutMsg() (*RawMsg, error)

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
	reqMsg        *RawMsg
	ackMsg        *RawMsg
	localFnDesc   FuncDesc
	deferFuncList []func()
	liblpc.BaseUserData
}

func (this *contextImpl) Reset() {
	//
	for i := len(this.deferFuncList) - 1; i >= 0; i-- {
		this.deferFuncList[i]()
	}
	//
	this.call = nil
	this.in = nil
	this.out = nil
	this.err = nil
	this.reqMsg = nil
	this.ackMsg = nil
	this.localFnDesc = 0
	this.writer = nil
	this.SetUserData(nil)
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

func (this *contextImpl) SetRequestType(t reflect.Type) {
	this.inType = t
}

func (this *contextImpl) SetResponse(out interface{}) {
	this.out = out
}

func (this *contextImpl) SetResponseType(t reflect.Type) {
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

func (this *contextImpl) BuildOutMsg() (*RawMsg, error) {
	out := this.ackMsg
	serErr := out.SetData(this.out)
	if serErr != nil {
		return nil, serErr
	}
	out.SetError(this.err)
	return out, nil
}

func (this *contextImpl) Init(call Callable, inMsg *RawMsg) {
	this.call = call
	this.writer = call.Writer()
	this.reqMsg = inMsg
	this.ackMsg = &RawMsg{
		Id:         this.Id(),
		MethodName: this.Method(),
		Type:       AckMsg,
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

func (this *contextImpl) FuncDesc() FuncDesc {
	return this.localFnDesc
}

func (this *contextImpl) SetFuncDesc(desc FuncDesc) {
	this.localFnDesc = desc
}

func (this *contextImpl) ReqMsg() *RawMsg {
	return this.reqMsg
}

func (this *contextImpl) SetReqMsg(msg *RawMsg) {
	this.reqMsg = msg
}

func (this *contextImpl) AckMsg() *RawMsg {
	return this.ackMsg
}

func (this *contextImpl) SetAckMsg(msg *RawMsg) {
	this.ackMsg = msg
}
