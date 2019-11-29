package rpcx

import (
	"errors"
	"github.com/gen-iot/std"
	"log"
)

// HEADER(FE FE) 2 |DATA_LEN 4| DATA N|

const kHeaderLen = 2
const kDataLen = 4

const kHeaderOffset = 0
const kDataLenOffset = kHeaderOffset + kHeaderLen
const kDataOffset = kDataLenOffset + kDataLen

const kMinMsgLen = kDataOffset

var ErrNeedMore = errors.New("codec want read more bytes")

type MsgType int

const (
	ReqMsg MsgType = iota
	AckMsg
)

type RawMsg struct {
	Id         string            `json:"msgId"`
	MethodName string            `json:"methodName"`
	Headers    map[string]string `json:"headers"`
	Type       MsgType           `json:"type"` // req or ack
	Err        *string           `json:"err"`  // fast path for ack error
	Data       []byte            `json:"data"` // req = param
}

func (this *RawMsg) GetError() error {
	if this.Err == nil {
		return nil
	}
	return errors.New(*this.Err)
}

func (this *RawMsg) SetErrorString(es string) {
	this.Err = &es
}

func (this *RawMsg) SetError(err error) {
	if err == nil {
		return
	}
	es := err.Error()
	this.Err = &es
}

func (this *RawMsg) BindData(v interface{}) error {
	return gRpcSerialization.UnMarshal(this.Data, v)
}

func (this *RawMsg) SetData(v interface{}) error {
	if v == nil {
		this.Data = nil
		return nil
	}
	bytes, err := gRpcSerialization.Marshal(v)
	if err != nil {
		return err
	}
	this.Data = bytes
	return nil
}

func decodeRpcMsg(buf std.ReadableBuffer, maxBodyLen int) (*RawMsg, error) {
	std.Assert(maxBodyLen > 0, "maxBodyLen must > 0")
	for {
		if buf.ReadableLen() < kMinMsgLen {
			return nil, ErrNeedMore
		}
		header := buf.PeekUInt16(kHeaderOffset)
		if header != 0xFEFE {
			buf.PopN(1)
			continue
		}
		dataLen := buf.PeekInt32(kDataLenOffset)
		if dataLen < 0 {
			buf.PopN(1)
			continue
		}
		if dataLen > int32(buf.ReadableLen()-kMinMsgLen) {
			return nil, ErrNeedMore
		}
		if int(dataLen) > maxBodyLen {
			buf.PopN(kMinMsgLen + int(dataLen))
			continue
		}
		buf.PopN(kDataOffset)
		data := buf.ReadN(int(dataLen))
		outMsg := new(RawMsg)
		err := gRpcSerialization.UnMarshal(data, outMsg)
		if err != nil {
			log.Println("unmarshal rpcx msg failed -> ", err)
			continue
		}
		return outMsg, nil
	}
}

func encodeRpcMsg(msg *RawMsg) ([]byte, error) {
	std.Assert(len(msg.Id) == 32, "msgId.Len != 32")
	buffer := std.NewByteBuffer()
	datas, err := gRpcSerialization.Marshal(msg)
	if err != nil {
		return nil, err
	}
	buffer.WriteUInt16(0xFEFE)
	buffer.WriteInt32(int32(len(datas)))
	buffer.Write(datas)
	return buffer.ToArray(), nil
}
