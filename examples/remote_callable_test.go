package examples

import (
	"context"
	"github.com/gen-iot/liblpc/v2"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/std"
	"testing"
	"time"
)

func mockRemoteRPC(core rpcx.Core, t *testing.T, cancelFunc context.CancelFunc) {
	defer cancelFunc()
	sockAddr, err := liblpc.ResolveTcpAddr("127.0.0.1:12345")
	std.AssertError(err, "mockRemoteRPC resolve addr")
	callable, err := rpcx.NewClientStreamCallable(core, sockAddr, nil)
	std.AssertError(err, "new client callable")
	callable.Start()
	out := new(string)
	err = callable.Call3(time.Second*5, "hello", out)
	std.AssertError(err, "call failed")
	t.Log("remote ack:", *out, ",err=", err)

}

func startAcceptor(core rpcx.Core, t *testing.T) {
	lfd, err := liblpc.NewListenerFd(
		"127.0.0.1:12345",
		1024,
		true,
		true)
	std.AssertError(err, "new listener fd")
	l := liblpc.NewListener(core.Loop(), int(lfd),
		func(ln *liblpc.Listener, newFd int, err error) {
			callable := rpcx.NewConnStreamCallable(core, newFd, nil)
			callable.Start()
		})
	l.Start()
}

func TestAcceptRemote(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	core, err := rpcx.New()
	std.AssertError(err, "new rpc")
	defer std.CloseIgnoreErr(core)
	core.RegFuncWithName("hello", func(ctx rpcx.Context) (string, error) {
		return "hello world", nil
	})
	startAcceptor(core, t)
	go mockRemoteRPC(core, t, cancel)
	core.Run(ctx)
}
