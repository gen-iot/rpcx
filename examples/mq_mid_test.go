package examples

import (
	"context"
	"fmt"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/rpcx/v2/middleware"
	"github.com/gen-iot/std"
	"testing"
	"time"
)

type fakeMqMsg struct {
	target string
	reply  string
	msg    []byte
}

var ch1 = make(chan *fakeMqMsg, 2)

func mqSend(ch chan<- *fakeMqMsg, target, reply string, msg []byte) {
	ch <- &fakeMqMsg{
		target: target,
		reply:  reply,
		msg:    msg,
	}
}

func wrapMqSend(ch chan *fakeMqMsg) middleware.MqSendFunc {
	return func(targetTopic, replyTopic string, msg []byte) {
		mqSend(ch, targetTopic, replyTopic, msg)
	}
}

func fakeMq(ctx context.Context, mq *middleware.Mq) {
	timer := time.NewTimer(time.Second * 5)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch1:
			mq.OnReceive(msg.msg)
		}
	}
}

func fakeCall(core rpcx.Core, cancelFunc func()) {
	defer cancelFunc()
	sockFd, err := liblpc.NewTcpSocketFd(4, true, true)
	std.AssertError(err, "new sock fd")
	callable := rpcx.NewConnStreamCallable(core, int(sockFd), nil)
	callable.Start()
	out := new(string)
	_, err = callable.Call6(time.Second*5, "hello", middleware.MqMakeHeader("abc", "xyz"), "client msg", out)
	std.AssertError(err, "call error")
	fmt.Printf("go ack:%s\n", *out)
}

func TestMq(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	//
	rpc, err := rpcx.New()
	std.AssertError(err, "new rpc")
	defer std.CloseIgnoreErr(rpc)
	//
	mq := middleware.NewMq(rpc, wrapMqSend(ch1))
	//
	rpc.Use(mq.Middleware())
	//
	rpc.RegFuncWithName("hello", func(ctx rpcx.Context, req string) (string, error) {
		return fmt.Sprintf("server recv req:%s", req), nil
	})
	//
	go fakeMq(ctx, mq)
	go fakeCall(rpc, cancelFunc)
	//
	rpc.Run(ctx)
}
