package rpcx

import (
	"errors"
	"log"
	"reflect"
)

type FuncDesc uint8

const (
	ReqHasData FuncDesc = 0x01
	RspHasData FuncDesc = 0x02
)

type rpcFunc struct {
	name           string
	fun            reflect.Value
	inParamType    reflect.Type
	outParamType   reflect.Type
	mid            middleware
	handleFunc     HandleFunc
	handleFuncDesc FuncDesc
}

func (this *rpcFunc) decodeInParam(data []byte) (interface{}, error) {
	// fastpath
	if this.handleFuncDesc&ReqHasData == 0 {
		// request in is nil
		return nil, nil
	}
	// check reqHasData
	if len(data) == 0 {
		return reflect.Zero(this.inParamType).Interface(), nil
	}
	elementType := this.inParamType
	isPtr := false
	if this.inParamType.Kind() == reflect.Ptr {
		elementType = this.inParamType.Elem()
		isPtr = true
	}
	newOutValue := reflect.New(elementType)
	newOut := newOutValue.Interface()
	err := gRpcSerialization.UnMarshal(data, newOut)
	if err != nil {
		return nil, err
	}
	if !isPtr {
		newOut = newOutValue.Elem().Interface()
	}
	return newOut, nil
}

var errInvokeErr = errors.New("invoke failed")
var errInParamNil = errors.New("inParam is nil")

// dont call this func directly
func (this *rpcFunc) ____invoke(c Context) {
	ctx := c.(*contextImpl)
	defer func() {
		//
		panicErr := recover()
		if panicErr == nil {
			return
		}

		if Debug {
			panic(panicErr)
		}
		log.Printf("call [%s] error:%v\n", ctx.Method(), panicErr)
		ctx.SetError(errInvokeErr)
	}()
	err := ctx.Error()
	if err != nil {
		return
	}
	var outParam interface{} = nil
	var paramV []reflect.Value = nil
	if this.handleFuncDesc&ReqHasData != 0 {
		inParam := ctx.Request()
		if inParam == nil {
			ctx.SetError(errInParamNil)
			return
		}
		paramV = []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(inParam)}
	} else {
		paramV = []reflect.Value{reflect.ValueOf(ctx)}
	}
	retV := this.fun.Call(paramV)
	rspErrIdx := -1
	rspDataIdx := -1
	if this.handleFuncDesc&RspHasData != 0 {
		rspErrIdx = 1
		rspDataIdx = 0
	} else {
		rspErrIdx = 0
	}
	if !retV[rspErrIdx].IsNil() { // check error
		err = retV[rspErrIdx].Interface().(error)
		ctx.SetError(err)
		return
	}
	if rspDataIdx != -1 {
		outParam = retV[rspDataIdx].Interface()
		ctx.SetResponse(outParam)
	}
}
