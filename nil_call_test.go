package rpcx

import (
	"fmt"
	"github.com/gen-iot/liblpc"
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

func sumFn(ctx Context, req sumReq) (string, error) {
	header := ctx.RequestHeader()
	for k, v := range header {
		fmt.Println("key=", k, ", value=", v)
	}
	return "hello world", nil
}

func TestCall(t *testing.T) {
	rpc, err := New()
	std.AssertError(err, "rpc new")
	rpcFnName := "Sum"
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
		callable := NewSignalCallable(call)
		callable.Start()
		<-callable.ReadySignal()
		fmt.Println("conn callable ready!")
		//out := new(string)
		err = callable.Call0(time.Second*5, rpcFnName, map[string]string{
			"k1":    "k2",
			"hello": "world",
		})
		fmt.Println("call err :", err)
		<-callable.CloseSignal()
		fmt.Println("conn callable closed")
	}()
	wg.Wait()
}
