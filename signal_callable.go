package rpcx

type SignalCallable struct {
	Callable
	cloSig   *sigGuard
	readySig *sigGuard
}

func NewSignalCallable(call Callable) *SignalCallable {
	out := &SignalCallable{
		Callable: call,
		cloSig:   newSigGuard(),
		readySig: newSigGuard(),
	}
	out.Callable.SetOnReady(func(callable Callable, err error) {
		out.readySig.Send(err)
	})
	out.Callable.SetOnClose(func(callable Callable, err error) {
		out.cloSig.Send(err)
	})
	return out
}

// usage : err,ok := CloseSignal(); !ok -> return , ok -> check error
func (this *SignalCallable) CloseSignal() <-chan error {
	return this.cloSig.Signal()
}

// usage : err,ok := ReadySignal(); !ok -> return , ok -> check error
func (this *SignalCallable) ReadySignal() <-chan error {
	return this.readySig.Signal()
}

func (this *SignalCallable) Close() error {
	this.cloSig.Send(nil)
	this.readySig.Send(nil)
	return this.Callable.Close()
}
