package protocol

import (
	"encoding/binary"
	"github.com/hezhis/go_utils"
)

const HeaderLength = 2

// SerializeType defines serialization type of payload.
type SerializeType byte

const (
	// SerializeNone uses raw []byte and don't serialize/deserialize
	SerializeNone SerializeType = iota
	ProtoBuffer                 // ProtoBuffer for payload.
	GobBuffer                   // Gob for payload
)

// CompressType defines decompression type.
type CompressType byte

type (
	Message struct {
		*Header
		ServicePath   string
		ServiceMethod string
		Payload       []byte
	}
	Header [HeaderLength]byte
)

// NewMessage creates an empty message.
func NewMessage() *Message {
	header := Header([HeaderLength]byte{})

	return &Message{
		Header: &header,
	}
}

// SerializeType returns serialization type of payload.
func (h Header) SerializeType() SerializeType {
	return SerializeType(h[0])
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
	h[0] = byte(st)
}

func (m Message) Encode() []byte {
	spL := len(m.ServicePath)
	smL := len(m.ServiceMethod)

	totalL := HeaderLength + (4 + spL) + (4 + smL) + (4 + len(m.Payload))
	data := make([]byte, totalL)

	copy(data, m.Header[:])

	spLEnd := HeaderLength + 4
	binary.BigEndian.PutUint32(data[HeaderLength:spLEnd], uint32(spL))
	copy(data[spLEnd:spLEnd+spL], go_utils.StringToSliceByte(m.ServicePath))

	smLStart := spLEnd + spL
	binary.BigEndian.PutUint32(data[smLStart:smLStart+4], uint32(smL))
	smLEnd := smLStart + 4
	copy(data[smLEnd:smLEnd+smL], go_utils.StringToSliceByte(m.ServiceMethod))

	payLoadStart := smLEnd + smL
	binary.BigEndian.PutUint32(data[payLoadStart:payLoadStart+4], uint32(len(m.Payload)))
	copy(data[payLoadStart+4:], m.Payload)

	return data
}

func (m *Message) Decode(data []byte) error {
	n := 0
	m.Header = (*Header)(data[n:HeaderLength])

	n += HeaderLength

	l := binary.BigEndian.Uint32(data[n : n+4])

	n = n + 4

	spLEnd := n + int(l)
	m.ServicePath = go_utils.SliceByteToString(data[n:spLEnd])
	n = spLEnd

	// parse serviceMethod
	l = binary.BigEndian.Uint32(data[n : n+4])
	n = n + 4
	nEnd := n + int(l)
	m.ServiceMethod = go_utils.SliceByteToString(data[n:nEnd])
	n = nEnd

	// parse payload
	l = binary.BigEndian.Uint32(data[n : n+4])
	_ = l
	n = n + 4
	m.Payload = data[n:]

	return nil
}
