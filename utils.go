package rpcx

import (
	"gitee.com/gen-iot/std"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()
var typeOfContext = reflect.TypeOf((*Context)(nil)).Elem()

func checkInParam(t reflect.Type) {
	inNum := t.NumIn()
	std.Assert(inNum == 2, "func in1 param len != 1")
	in0 := t.In(0)
	std.Assert(in0 == typeOfContext, "param[0] must be rpcx.Context")
	in1 := t.In(1)
	in1Kind := in1.Kind()
	std.Assert(in1Kind == reflect.Ptr || in1Kind == reflect.Struct, "param[1] must be prt of struct")
}

type returnType int

const (
	kResponseAndErr returnType = iota
	kErrOnly
)

func checkOutParam(t reflect.Type) returnType {
	outNum := t.NumOut()
	rtType := returnType(-1)
	switch outNum {
	case 1:
		out1 := t.Out(0)
		std.Assert(out1 == typeOfError, "first param of out_param ,type must be `error`")
		rtType = kErrOnly
	case 2:
		out1 := t.Out(1)
		std.Assert(out1 == typeOfError, "last param of out_param ,type must be `error`")
		rtType = kResponseAndErr
	default:
		std.Assert(false, "illegal func return type num")
	}
	return rtType
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

func getValueElement(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
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
