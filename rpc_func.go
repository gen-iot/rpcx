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
	// check reqHasData
	if this.handleFuncDesc&reqHasData != 0 {
		if len(data) == 0 {
			// request in is nil
			return nil, nil
		}
	} else {
		// request in doesn't exist
		return nil, nil
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
		log.Println("call error ", panicErr)
		ctx.SetError(errInvokeErr)
	}()
	err := ctx.Error()
	if err != nil {
		return
	}
	inParam := ctx.Request()
	if inParam == nil {
		ctx.SetError(errInParamNil)
		return
	}
	// todo 检查参数,有可能没有传入参数.
	// 如:用户可能会如此调用 err:=call.Call(ctx)
	// 检查入参是否存在
	paramV := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(inParam)}
	retV := this.fun.Call(paramV)
	if !retV[1].IsNil() {
		err = retV[1].Interface().(error)
		ctx.SetError(err)
		return
	}
	outParam := retV[0].Interface()
	ctx.SetResponse(outParam)
}
