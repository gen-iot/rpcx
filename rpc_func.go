package rpcx

import (
	"errors"
	"log"
	"reflect"
)

type funcDesc uint8

const (
	reqHasData funcDesc = 0x01
	rspHasData funcDesc = 0x02
)

type rpcFunc struct {
	name           string
	fun            reflect.Value
	inParamType    reflect.Type
	outParamType   reflect.Type
	mid            middleware
	handleFunc     HandleFunc
	handleFuncDesc funcDesc
}

func (this *rpcFunc) decodeInParam(data []byte) (interface{}, error) {
	// fastpath
	if this.handleFuncDesc&reqHasData == 0 {
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
	newOut := reflect.New(elementType).Interface()
	err := gRpcSerialization.UnMarshal(data, newOut)
	if err != nil {
		return nil, err
	}
	if !isPtr {
		newOut = reflect.ValueOf(newOut).Elem().Interface()
	}
	return newOut, nil
}

var errInvokeErr = errors.New("invoke failed")
var errInParamNil = errors.New("inParam is nil")

func (this *rpcFunc) invoke(c Context) {
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
		log.Println("call error ", panicErr)
		ctx.SetError(errInvokeErr)
	}()
	err := ctx.Error()
	if err != nil {
		return
	}
	var outParam interface{} = nil
	var paramV []reflect.Value = nil
	if this.handleFuncDesc&reqHasData != 0 {
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
	if this.handleFuncDesc&rspHasData != 0 {
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
