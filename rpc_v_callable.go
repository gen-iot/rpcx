package rpcx

import "github.com/gen-iot/std"

type VirtualCallable struct {
	buffer std.RwBuffer
	*BaseCallable
}

func NewVirtualCallable(core Core, writer WriterCloser) *VirtualCallable {
	pCall := new(VirtualCallable)
	pCall.BaseCallable = NewBaseCallable(core, writer, pCall)
	pCall.buffer = std.NewByteBuffer()
	return pCall
}

func (this *VirtualCallable) Start() {
	this.BaseCallable.Start()
	this.DoReady(nil)
}

func (this *VirtualCallable) Close() error {
	this.DoClosed(nil)
	return nil
}

// not guarantee data seq, don't use in multi goroutine
func (this *VirtualCallable) MockReadData(data []byte) {
	this.core.Loop().RunInLoop(func() {
		this.buffer.Write(data)
		this.core.NotifyCallableRead(this, this.buffer)
	})
}
