package rpcx

import (
	"gitee.com/Puietel/std"
	"gitee.com/SuzhenProjects/liblpc"
)

type Context interface {
	Callable() Callable

	Id() string

	SetMethod(string)
	Method() string

	RequestHeader() map[string]string
	SetRequestHeader(map[string]string)

	SetRequest(in interface{})
	Request() interface{}

	ResponseHeader() map[string]string
	SetResponseHeader(map[string]string)

	SetResponse(out interface{})
	Response() interface{}

	SetError(err error)
	Error() error

	liblpc.UserDataStorage
}

type contextImpl struct {
	call   Callable
	in     interface{}
	out    interface{}
	err    error
	reqMsg *rpcRawMsg
	ackMsg *rpcRawMsg
	liblpc.BaseUserData
}

func (this *contextImpl) reset() {
	this.call = nil
	this.in = nil
	this.out = nil
	this.err = nil
	this.reqMsg = nil
	this.ackMsg = nil
	this.SetUserData(nil)
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

func (this *contextImpl) SetResponse(out interface{}) {
	this.out = out
}

func (this *contextImpl) Response() interface{} {
	return this.out
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

func (this *contextImpl) buildOutMsg() *rpcRawMsg {
	this.ackMsg = &rpcRawMsg{
		Id:         this.Id(),
		MethodName: this.Method(),
		Type:       rpcAckMsg,
	}
	out := this.ackMsg
	if this.err != nil {
		out.SetError(this.err)
	} else {
		err := out.SetData(this.out)
		if err != nil {
			out.SetError(err)
		}
	}
	return out
}

func (this *contextImpl) init(call Callable, inMsg *rpcRawMsg) {
	this.call = call
	this.reqMsg = inMsg
}
