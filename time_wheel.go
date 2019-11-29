package rpcx

import (
	"github.com/gen-iot/liblpc"
	"io"
)

type TimeWheelEntryImpl struct {
	io.Closer
	refCount  int32
	timeWheel *liblpc.TimeWheel
}

// not thread safe
func (this *TimeWheelEntryImpl) BindTimeWheel(timeWheel *liblpc.TimeWheel) {
	this.timeWheel = timeWheel
}

func (this *TimeWheelEntryImpl) NotifyTimeWheel() {
	if this.timeWheel == nil {
		return
	}
	this.timeWheel.Entries() <- this
}

func (this *TimeWheelEntryImpl) GetRefCounter() *int32 {
	return &this.refCount
}
