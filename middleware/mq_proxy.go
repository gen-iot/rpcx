package middleware

import (
	"github.com/gen-iot/rpcx/v2"
	"github.com/gen-iot/std"
	"github.com/pkg/errors"
	"sync"
)

type fakeDataWriter struct {
	sendF  MqSendFunc
	target string
	reply  string
}

func (this *fakeDataWriter) Write(ctx rpcx.Context, data []byte, inLoop bool) {
	this.sendF(this.target, this.reply, data)
}

func (this *fakeDataWriter) reset() {
	this.sendF = nil
	this.target = this.target[:0]
	this.reply = this.reply[:0]
}

var __fdwPool = sync.Pool{
	New: func() interface{} {
		return new(fakeDataWriter)
	},
}

type MqSendFunc func(targetTopic, replyTopic string, msg []byte)

type Mq struct {
	core   rpcx.Core
	vCall  *rpcx.VirtualCallable
	mqSend MqSendFunc
}

func NewMq(core rpcx.Core, mqSend MqSendFunc) *Mq {
	std.Assert(core != nil, "rpc is required")
	std.Assert(mqSend != nil, "mqSend is required")
	this := &Mq{
		core:   core,
		mqSend: mqSend,
	}
	this.vCall = rpcx.NewVirtualCallable(core, nil)
	return this
}

func (this *Mq) OnReceive(msg []byte) {
	this.vCall.MockReadData(msg)
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
			ctx.AddDefer(func() {
				fdw.reset()
				__fdwPool.Put(fdw)
			})
			fdw.target = targetTopic
			fdw.reply = replyTopic
			fdw.sendF = this.mqSend
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
