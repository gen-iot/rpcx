package rpcx

import (
	"github.com/gen-iot/liblpc"
	"time"
)

type CallableCallback func(callable Callable, err error)

type Callable interface {
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

	liblpc.BucketEntry
	BindTimeWheel(timeWheel *liblpc.TimeWheel)
	NotifyTimeWheel()

	Writer() Writer
}
