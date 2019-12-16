package examples

import (
	"context"
	"fmt"
	"github.com/gen-iot/liblpc/v2"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/rpcx/v2/middleware"
	"github.com/gen-iot/std"
	"sync"
	"testing"
	"time"
)

func TestCallTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wheel := liblpc.NewTimeWheel(2, 4)
	go wheel.Execute(ctx)
	core, err := rpcx.New()
	std.AssertError(err, "core new")
	defer std.CloseIgnoreErr(core)
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
		fmt.Println("accept success")
		call := rpcx.NewConnStreamCallable(core, newFd, nil)
		call.BindTimeWheel(wheel)
		call.Start()
	})
	l.Start()
	wg := new(sync.WaitGroup)
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
		time.Sleep(time.Second * 13)
		_, err = callable.Call4(time.Second*5, rpcFnName, map[string]string{
			"k1":    "k2",
			"hello": "world",
		}, out)
		fmt.Println("call err :", err)
		<-callable.CloseSignal()
		fmt.Println("conn callable closed")
	}()

	wg.Wait()
	core.Run(ctx)
}
