package rpcx

import (
	"errors"
	"log"
	"reflect"
)

type rpcFunc struct {
	name      string
	fun       reflect.Value
	inP0Type  reflect.Type
	outP0Type reflect.Type
	mid       middleware
	handleF   HandleFunc
}

func (this *rpcFunc) decodeInParam(data []byte) (interface{}, error) {
	elementType := this.inP0Type
	isPtr := false
	if this.inP0Type.Kind() == reflect.Ptr {
		elementType = this.inP0Type.Elem()
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
