package examples

import (
	"fmt"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx"
	"github.com/gen-iot/rpcx/middleware"
	"github.com/gen-iot/std"
	"sync"
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

func sumFn(ctx rpcx.Context) error {
	header := ctx.RequestHeader()
	for k, v := range header {
		fmt.Println("key=", k, ", value=", v)
	}
	return nil
}

func TestCall(t *testing.T) {
	rpc, err := rpcx.New()
	std.AssertError(err, "rpc new")
	const rpcFnName = "Sum"
	rpc.Use(func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			fmt.Println("tag1")
			next(ctx)
			fmt.Println("tag2")
		}
	}, middleware.Validate(std.NewValidator(std.LANG_EN)))
	rpc.RegFuncWithName(rpcFnName, sumFn)
	rpc.Start()
	addr := "127.0.0.1:8848"
	lfd, err := liblpc.NewListenerFd(addr, 128, true, true)
	std.AssertError(err, "new listener fd")
	l := liblpc.NewListener(rpc.Loop(), int(lfd), func(ln *liblpc.Listener, newFd int, err error) {
		std.AssertError(err, "accept err")
		call := rpc.NewConnCallable(newFd, nil)
		call.Start()
	})
	l.Start()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sockAddr, err := liblpc.ResolveTcpAddr(addr)
		std.AssertError(err, "resolve addr err")
		call, err := rpc.NewClientCallable(sockAddr, nil)
		std.AssertError(err, "new client callable")
		callable := rpcx.NewSignalCallable(call)
		callable.Start()
		<-callable.ReadySignal()
		fmt.Println("conn callable ready!")
		out := new(string)
		err = callable.Call4(time.Second*5, rpcFnName, map[string]string{
			"k1":    "k2",
			"hello": "world",
		}, out)
		fmt.Println("call err :", err)
		<-callable.CloseSignal()
		fmt.Println("conn callable closed")
	}()
	wg.Wait()
}
