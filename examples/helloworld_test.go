package examples

import (
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx"
	"github.com/gen-iot/rpcx/middleware"
	"github.com/gen-iot/std"
	"testing"
	"time"
)

func createConnCallable(rpc *rpcx.RPC, fd int, t *testing.T) {
	callable := rpc.NewConnCallable(fd, nil)
	defer std.CloseIgnoreErr(callable)
	callable.Start()

	out := new(string)
	err := callable.Call3(time.Second*5, "hello", out)
	std.AssertError(err, "call 'hello'")
	t.Log("remote ack: ", *out)
	// after recv msg
	// close rpc
	std.CloseIgnoreErr(rpc)
}

func TestHelloworld(t *testing.T) {
	rpc, err := rpcx.New()
	std.AssertError(err, "new rpc")
	rpc.Use(
		middleware.Recover(true),
		middleware.ValidateStruct(
			middleware.ValidateInOut,
			std.NewValidator(std.LANG_EN)),
	)
	rpc.RegFuncWithName("hello",
		func(ctx rpcx.Context, msg string) (string, error) {
			return "hello from server", nil
		},
		middleware.Recover(true), // recover
		middleware.ValidateStruct( // validator
			middleware.ValidateInOut,
			std.NewValidator(std.LANG_EN)),
	)
	fds, err := liblpc.MakeIpcSockpair(true)
	std.AssertError(err, "new sock pair")

	go createConnCallable(rpc, fds[0], t)

	callable := rpc.NewConnCallable(fds[1], nil)
	callable.Start()
	rpc.Run()
}
