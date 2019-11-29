package examples

import (
	"context"
	"errors"
	"fmt"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/rpcx/v2/middleware"
	"github.com/gen-iot/std"
	"testing"
	"time"
)

type sumReq struct {
	A int
	B int
}

type sumRsp struct {
	Sum int
}

func sumFn(ctx rpcx.Context) (string, error) {
	header := ctx.RequestHeader()
	responseHeader := ctx.ResponseHeader()
	responseHeader["ack1"] = "ack1"
	responseHeader["ack2"] = "ack2"
	responseHeader["ack3"] = "ack3"
	for k, v := range header {
		fmt.Println("key=", k, ", value=", v)
	}
	return "abc", errors.New("fucking error")
}

func TestCall(t *testing.T) {
	core, err := rpcx.New()
	std.AssertError(err, "rpc new")
	defer std.CloseIgnoreErr(core)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const rpcFnName = "Sum"
	core.Use(func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			fmt.Println("tag1")
			next(ctx)
			fmt.Println("tag2")
		}
	}, middleware.ValidateStruct(middleware.ValidateInOut, std.NewValidator(std.LANG_EN)))
	core.RegFuncWithName(rpcFnName, sumFn)

	addr := "127.0.0.1:8848"
	lfd, err := liblpc.NewListenerFd(addr, 128, true, true)
	std.AssertError(err, "new listener fd")
	l := liblpc.NewListener(core.Loop(), int(lfd), func(ln *liblpc.Listener, newFd int, err error) {
		std.AssertError(err, "accept err")
		call := rpcx.NewConnStreamCallable(core, newFd, nil)
		call.Start()
	})
	l.Start()
	go func() {
		defer cancel()
		sockAddr, err := liblpc.ResolveTcpAddr(addr)
		std.AssertError(err, "resolve addr err")
		call, err := rpcx.NewClientStreamCallable(core, sockAddr, nil)
		std.AssertError(err, "new client callable")
		callable := rpcx.NewSignalCallable(call)
		callable.Start()
		<-callable.ReadySignal()
		fmt.Println("conn callable ready!")
		out := new(string)
		ackHeader, err := callable.Call4(time.Second*5, rpcFnName, map[string]string{
			"k1":    "k2",
			"hello": "world",
		}, out)
		fmt.Println("ack header=", ackHeader)
		fmt.Println("call err :", err)
		fmt.Println("call out :", *out)
		std.CloseIgnoreErr(callable)
		<-callable.CloseSignal()
		fmt.Println("conn callable closed")
	}()
	core.Run(ctx)
}
