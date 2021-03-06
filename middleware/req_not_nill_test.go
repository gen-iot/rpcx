package middleware

import (
	"context"
	"fmt"
	"github.com/gen-iot/liblpc/v2"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/std"
	"sync"
	"testing"
	"time"
)

type fun1Req struct {
	A int
	B int
}

type fun1Rsp struct {
	Sum int
}

func funWithBasicReqRsp(ctx rpcx.Context, req *fun1Req) (*fun1Rsp, error) {
	return &fun1Rsp{
		Sum: req.A + req.B,
	}, nil
}

func funWithStringRequest(ctx rpcx.Context, req string) error {
	fmt.Println("funWithStringRequest >> ", req)
	return nil
}

func funWithStringPtrRequest(ctx rpcx.Context, req *string) error {
	fmt.Println("funWithStringPtrRequest >> ", *req)
	return nil
}

func TestRequestNotNil(t *testing.T) {
	fncLifeContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	fds, err := liblpc.MakeIpcSockpair(true)
	std.AssertError(err, "socketPair error")

	core, err := rpcx.New()
	std.AssertError(err, "new rpcx")
	defer std.CloseIgnoreErr(core)
	core.PreUse(RequestNotNil())
	core.Start(fncLifeContext)

	core.RegFunc(funWithBasicReqRsp)
	core.RegFunc(funWithStringRequest)
	core.RegFunc(funWithStringPtrRequest)

	call := rpcx.NewConnStreamCallable(core, fds[0], nil)
	call.Start()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fd := fds[1]
		core2, err := rpcx.New()
		std.AssertError(err, "new rpcx")
		defer std.CloseIgnoreErr(core2)
		core2.Start(fncLifeContext)
		callable := rpcx.NewConnStreamCallable(core2, fd, nil)
		cliCall := rpcx.NewSignalCallable(callable)
		cliCall.Start()
		<-cliCall.ReadySignal()
		//test basic req &rsp
		rsp1 := new(fun1Rsp)
		err = callable.Call5(time.Second*5, "funWithBasicReqRsp", &fun1Req{A: 10, B: 100}, rsp1)
		std.AssertError(err, "call funWithBasicReqRsp error")
		std.Assert(rsp1.Sum == 10+100, "result error")
		err = callable.Call5(time.Second*5, "funWithBasicReqRsp", nil, rsp1)
		std.Assert(err != nil, "should be error")

		//test string req
		err = nil
		err = callable.Call1(time.Second*5, "funWithStringRequest", "hello test", nil)
		std.AssertError(err, "call funWithStringRequest error")
		err = callable.Call1(time.Second*5, "funWithStringRequest", "", nil)
		std.AssertError(err, "call funWithStringRequest error")
		err = callable.Call1(time.Second*5, "funWithStringRequest", nil, nil)
		std.AssertError(err, "call funWithStringRequest error")

		err = nil
		err = callable.Call1(time.Second*5, "funWithStringPtrRequest", "hello test", nil)
		std.AssertError(err, "call funWithStringPtrRequest error")
		err = callable.Call1(time.Second*5, "funWithStringPtrRequest", "", nil)
		std.AssertError(err, "call funWithStringPtrRequest error")
		err = callable.Call1(time.Second*5, "funWithStringPtrRequest", nil, nil)
		std.Assert(err != nil, "call funWithStringPtrRequest should be error")
	}()
	wg.Wait()
}
