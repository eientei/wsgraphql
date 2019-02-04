package proto

import (
	"encoding/json"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage("abc", 123, GQLStop)
	if msg.Id != "abc" {
		t.Error("msg id invalid", msg.Id)
	}
	if msg.Payload == nil {
		t.Error("msg payload nil")
	} else {
		if msg.Payload.Value != 123 {
			t.Error("msg payload invalid", msg.Payload.Value)
		}
		if msg.Payload.Bytes != nil {
			t.Error("msg bytes non-empty", msg.Payload.Bytes)
		}
	}
	if msg.Type != GQLStop {
		t.Error("msg type invalid", msg.Type)
	}
}

func TestNewMessage_cycle(t *testing.T) {
	msg := NewMessage("abc", 123, GQLStop)
	bs, err := json.Marshal(msg)
	if err != nil {
		t.Error("marshal error occurred", err)
	}
	res := &Message{}
	err = json.Unmarshal(bs, res)
	if err != nil {
		t.Error("unmarshal error occurred", err)
	}
	if string(res.Payload.Bytes) != "123" {
		t.Error("payload invalid", string(res.Payload.Bytes))
	}
}
