package rpcx

import (
	"github.com/gen-iot/std"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var typeOfContext = reflect.TypeOf((*Context)(nil)).Elem()

func checkInParam(t reflect.Type) (reflect.Type, funcDesc) {
	fnInDesc := funcDesc(0)
	inNum := t.NumIn()
	var inParamType reflect.Type = nil
	std.Assert(inNum > 0 && inNum <= 2, "inNum len must less or equal than 2")
	//
	in0 := t.In(0)
	std.Assert(in0 == typeOfContext, "first in param must be rpcx.Context")
	//
	switch inNum {
	case 1:
		// func foo(context)
		fnInDesc = 0
	case 2:
		// func foo(context,param1)
		fnInDesc = reqHasData
		in1 := t.In(1)
		inParamType = in1
	default:
		std.Assert(false, "illegal func in params num")
	}
	return inParamType, fnInDesc
}

func checkOutParam(t reflect.Type) (reflect.Type, funcDesc) {
	outNum := t.NumOut()
	fnOutDesc := funcDesc(0)
	var outParamType reflect.Type = nil
	switch outNum {
	case 1:
		out1 := t.Out(0)
		std.Assert(out1 == typeOfError, "first param of out_param ,type must be `error`")
	case 2:
		fnOutDesc = rspHasData
		outParamType = t.Out(0)
		out1 := t.Out(1)
		std.Assert(out1 == typeOfError, "last param of out_param ,type must be `error`")
	default:
		std.Assert(false, "illegal func return params num")
	}
	return outParamType, fnOutDesc
}

func getFuncName(fv reflect.Value) string {
	fname := runtime.FuncForPC(reflect.Indirect(fv).Pointer()).Name()
	idx := strings.LastIndex(fname, ".")
	if idx != -1 {
		fname = fname[idx+1:]
	}
	idx = strings.LastIndex(fname, "-")
	if idx != -1 {
		fname = fname[:idx]
	}
	return fname
}

type sigGuard struct {
	sig  chan error
	flag int32
}

func newSigGuard() *sigGuard {
	return &sigGuard{
		sig:  make(chan error, 1),
		flag: 0,
	}
}

func (this *sigGuard) Signal() <-chan error {
	return this.sig
}

func (this *sigGuard) Send(err error) bool {
	if atomic.CompareAndSwapInt32(&this.flag, 0, 1) {
		this.sig <- err
		close(this.sig)
		return true
	}
	return false
}
