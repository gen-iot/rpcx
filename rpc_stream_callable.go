package rpcx

import (
	"github.com/gen-iot/liblpc/v2"
	"github.com/gen-iot/std"
)

type streamCallImpl struct {
	stream *liblpc.BufferedStream
	*BaseCallable
}

func (this *streamCallImpl) Start() {
	std.Assert(this.stream != nil, "stream is nil")
	this.BaseCallable.Start()
	this.stream.Start()
}

type bufferedStreamWrapper struct {
	stream *liblpc.BufferedStream
}

func (this *bufferedStreamWrapper) Write(ctx Context, data []byte, inLoop bool) {
	this.stream.Write(data, inLoop)
}

func (this *bufferedStreamWrapper) Close() error {
	return this.stream.Close()
}

func newStreamCall(core Core, stream *liblpc.BufferedStream, userData interface{}, m ...MiddlewareFunc) *streamCallImpl {
	pCall := &streamCallImpl{
		stream: stream,
	}
	pCall.BaseCallable = NewBaseCallable(core, &bufferedStreamWrapper{
		stream: stream,
	}, pCall)
	pCall.TimeWheelEntryImpl.Closer = pCall
	//
	pCall.Use(m...)
	//
	pCall.SetUserData(userData)
	pCall.stream.SetUserData(pCall)
	return pCall
}

func NewConnStreamCallable(core Core, fd int, userData interface{}, m ...MiddlewareFunc) Callable {
	stream := liblpc.NewBufferedConnStream(core.Loop(), fd,
		func(sw liblpc.StreamWriter, buf std.ReadableBuffer) {
			call := sw.GetUserData().(Callable)
			core.NotifyCallableRead(call, buf)
		})
	pCall := newStreamCall(core, stream, userData, m...)
	//
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
	return pCall
}

func NewClientStreamCallable(core Core, addr *liblpc.SyscallSockAddr,
	userData interface{},
	m ...MiddlewareFunc) (Callable, error) {
	fd, err := liblpc.NewConnFd2(addr.Version, addr.Sockaddr)
	if err != nil {
		return nil, err
	}
	return NewConnStreamCallable(core, int(fd), userData, m...), nil
}
