package rpcx

import (
	"gitee.com/gen-iot/std"
	"log"
	"reflect"
	"strings"
	"testing"
)

func doIt(p *int) {
	if p == nil {
		log.Println("nil")
		return
	}
	log.Println("not nil")
}

func TestPassNil(t *testing.T) {
	var p *int = nil
	pt := reflect.TypeOf(p)
	reflect.ValueOf(doIt).Call([]reflect.Value{reflect.Zero(pt)})
	bytes, err := gRpcSerialization.Marshal("abcd")
	std.AssertError(err, "marshal str err")
	out := new(string)
	err = gRpcSerialization.UnMarshal(bytes, out)
	std.AssertError(err, "unmarshal str err")
	std.Assert(strings.Compare("abcd", *out) == 0, "not matcheds")
}
