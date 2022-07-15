package protocol

import "testing"

func TestNewMessage(t *testing.T) {
	msg := NewMessage()
	msg.SetSerializeType(ProtoBuffer)
	msg.ServicePath = "actorData"
	msg.ServiceMethod = "loadActorCache"
	msg.Payload = []byte{1, 2, 3, 4, 5}

	data := msg.Encode()

	decodeMsg := NewMessage()
	decodeMsg.Decode(data)
}
