package rpcx

import (
	"fmt"
	"github.com/gen-iot/std"
	"testing"
)

type exampleStruct struct {
	Name string                 `json:"name"`
	Age  int                    `json:"age"`
	Meta map[string]interface{} `json:"meta"`
}

func newExampleStruct() *exampleStruct {
	return &exampleStruct{
		Name: "suzhen",
		Age:  100,
		Meta: map[string]interface{}{
			"k1": "v1",
			"k2": 1,
			"k3": []string{"a", "b", "c"},
		},
	}
}

func TestEncodeMessage_MSGPACK(t *testing.T) {
	msg := &rpcRawMsg{
		Id:         std.GenRandomUUID(),
		MethodName: "sum",
		Type:       rpcReqMsg,
	}
	msg.SetErrorString("try set error")
	std.AssertError(msg.SetData(newExampleStruct()), "set data error")
	bytes, err := encodeRpcMsg(msg)
	std.AssertError(err, "encodeRpcMsg")
	fmt.Println("encode -> ", string(bytes))
	buffer := std.NewByteBuffer()
	buffer.Write(bytes)
	outMsg, err := decodeRpcMsg(buffer, 1024*1024*4)
	std.AssertError(err, "decodeRpcMsg")
	outExampleStruct := new(exampleStruct)
	err = outMsg.BindData(outExampleStruct)
	std.AssertError(err, "bind data error")
	fmt.Println(*outExampleStruct)
}

func TestUUID(t *testing.T) {
	uuid := std.GenRandomUUID()
	fmt.Println("uuid -> ", uuid, " len -> ", len(uuid))
}
