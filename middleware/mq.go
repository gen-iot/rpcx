package middleware

import (
	"errors"
	"github.com/gen-iot/liblpc"
	"github.com/gen-iot/rpcx"
	"github.com/gen-iot/std"
	"sync"
)

// todo should reuse it
type fakeDataWriter struct {
	data []byte
}

func (this *fakeDataWriter) Write(data []byte, inLoop bool) {
	this.data = data
}

func (this *fakeDataWriter) Reuse() {
	this.data = this.data[:0]
}

var __fdwPool = sync.Pool{
	New: func() interface{} {
		return new(fakeDataWriter)
	},
}

type MqSendFunc func(targetTopic, replyTopic string, msg []byte)

type MQ struct {
	rpc          *rpcx.RPC
	pipeWriter   *liblpc.Stream
	pipeCallable rpcx.Callable
	mqSend       MqSendFunc
}

func NewMQ(rpc *rpcx.RPC, mqSend MqSendFunc) (*MQ, error) {
	std.Assert(rpc != nil && mqSend != nil, "bad params")
	this := &MQ{
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

func (this *MQ) Close() error {
	std.CloseIgnoreErr(this.pipeWriter)
	std.CloseIgnoreErr(this.pipeCallable)
	return nil
}

func (this *MQ) OnReceive(msg []byte) {
	this.pipeWriter.Write(msg, false)
}

func (this *MQ) Middleware() rpcx.MiddlewareFunc {
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
				ctx.SetError(errors.New("MQ MSG WITHOUT TARGET TOPIC"))
				return
			}
			oldWriter := ctx.Writer()
			defer ctx.SetWriter(oldWriter)
			fdw := __fdwPool.Get().(*fakeDataWriter)
			defer func() {
				fdw.Reuse()
				__fdwPool.Put(fdw)
			}()
			ctx.SetWriter(fdw)
			next(ctx)
			if ctx.Error() != nil {
				this.mqSend(targetTopic, replyTopic, fdw.data)
			}
		}
	}
}
