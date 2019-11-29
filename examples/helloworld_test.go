package examples

import (
	"context"
	"errors"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/rpcx/v2/middleware"
	"github.com/gen-iot/std"
	"testing"
	"time"
)

func createConnCallable(core rpcx.Core, fd int, t *testing.T, cancelFunc context.CancelFunc) {
	defer cancelFunc()
	callable := rpcx.NewConnStreamCallable(core, fd, nil)
	defer std.CloseIgnoreErr(callable)
	callable.Start()

	out := new(string)
	err := callable.Call3(time.Second*5, "hello", out)
	t.Log("remote ack: ", *out)
	t.Log("remote ack error: ", err)
}

func TestHelloworld(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	core, err := rpcx.New()
	std.AssertError(err, "new rpc")
	defer std.CloseIgnoreErr(core)
	core.Use(
		middleware.Recover(true),
		//middleware.ValidateStruct(
		//	middleware.ValidateInOut,
		//	std.NewValidator(std.LANG_EN)),
	)
	core.RegFuncWithName("hello",
		func(ctx rpcx.Context, msg string) (string, error) {
			return "hello from server", errors.New("abc")
		},
		middleware.Recover(true), // recover
		middleware.ValidateStruct( // validator
			middleware.ValidateInOut,
			std.NewValidator(std.LANG_EN)),
	)
	fds, err := liblpc.MakeIpcSockpair(true)
	std.AssertError(err, "new sock pair")

	go createConnCallable(core, fds[0], t, cancel)

	callable := rpcx.NewConnStreamCallable(core, fds[1], nil)
	callable.Start()
	core.Run(ctx)
}
