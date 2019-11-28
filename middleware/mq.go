package middleware

import (
	"fmt"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx"
	"github.com/gen-iot/std"
	"github.com/pkg/errors"
	"sync"
)

// todo should reuse it
type fakeDataWriter struct {
	sendF  MqSendFunc
	target string
	reply  string
}

func (this *fakeDataWriter) Write(data []byte, inLoop bool) {
	this.sendF(this.target, this.reply, data)
}

var __fdwPool = sync.Pool{
	New: func() interface{} {
		return new(fakeDataWriter)
	},
}

type MqSendFunc func(targetTopic, replyTopic string, msg []byte)

type Mq struct {
	rpc          *rpcx.RPC
	pipeWriter   *liblpc.Stream
	pipeCallable rpcx.Callable
	mqSend       MqSendFunc
}

func NewMq(rpc *rpcx.RPC, mqSend MqSendFunc) (*Mq, error) {
	std.Assert(rpc != nil, "rpc is required")
	std.Assert(mqSend != nil, "mqSend is required")
	this := &Mq{
		rpc:    rpc,
		mqSend: mqSend,
	}
	fds, err := liblpc.MakeIpcSockpair(true)
	if err != nil {
		return nil, err
	}
	this.pipeWriter = liblpc.NewConnStream(rpc.Loop().(*liblpc.IOEvtLoop), fds[0], nil)
	this.pipeWriter.Start()
	this.pipeCallable = rpc.NewConnCallable(fds[1], nil)
	this.pipeCallable.Start()
	return this, nil
}

func (this *Mq) Close() error {
	std.CloseIgnoreErr(this.pipeWriter)
	std.CloseIgnoreErr(this.pipeCallable)
	return nil
}

func (this *Mq) OnReceive(msg []byte) {
	this.pipeWriter.Write(msg, false)
}

func (this *Mq) Middleware() rpcx.MiddlewareFunc {
	return func(next rpcx.HandleFunc) rpcx.HandleFunc {
		return func(ctx rpcx.Context) {
			header := ctx.RequestHeader()
			_, mqFlag := header["MQ_PROTOCOL"]
			if !mqFlag {
				next(ctx)
				return
			}
			targetTopic, ok1 := header["MQ_TARGET_TOPIC"]
			replyTopic, _ := header["MQ_REPLY_TOPIC"]
			if !ok1 {
				ctx.SetError(errors.New("Mq MSG WITHOUT TARGET TOPIC"))
				return
			}
			fdw := __fdwPool.Get().(*fakeDataWriter)
			defer __fdwPool.Put(fdw)
			fdw.target = targetTopic
			fdw.reply = replyTopic
			fdw.sendF = this.mqSend
			fmt.Println("change writer!")
			ctx.SetWriter(fdw)
			next(ctx)
		}
	}
}

func MqMakeHeader(target, reply string) rpcx.RpcMsgHeader {
	return rpcx.RpcMsgHeader{
		"MQ_PROTOCOL":     "1",
		"MQ_TARGET_TOPIC": target,
		"MQ_REPLY_TOPIC":  reply,
	}
}
