package examples

import (
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/std"
	"testing"
	"time"
)

func mockRemoteRPC(core rpcx.Core, t *testing.T) {
	sockAddr, err := liblpc.ResolveTcpAddr("127.0.0.1:12345")
	std.AssertError(err, "mockRemoteRPC resolve addr")
	callable, err := rpcx.NewClientStreamCallable(core, sockAddr, nil)
	std.AssertError(err, "new client callable")
	callable.Start()
	out := new(string)
	err = callable.Call3(time.Second*5, "hello", out)
	std.AssertError(err, "call failed")
	t.Log("remote ack:", *out, ",err=", err)
	std.CloseIgnoreErr(core)
}

func startAcceptor(core rpcx.Core, t *testing.T) *liblpc.Listener {
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
	return l
}

func TestAcceptRemote(t *testing.T) {
	rpc, err := rpcx.New()
	std.AssertError(err, "new rpc")
	rpc.RegFuncWithName("hello", func(ctx rpcx.Context) (string, error) {
		return "hello world", nil
	})
	listener := startAcceptor(rpc, t)
	defer std.CloseIgnoreErr(listener)
	go mockRemoteRPC(rpc, t)
	rpc.Run(nil)
}
