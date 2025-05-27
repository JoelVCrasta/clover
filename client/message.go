package client

import "encoding/binary"

// <lengthPrefix : 4 bytes> <messageId : 1 byte> <payload : variable size>
type MessageId int

const (
	ChokeId MessageId = iota
	UnchokeId
	InterestedId
	NotInterestedId
	HaveId
	BitfieldId
	RequestId
	PieceId
	CancelId
	PortId
)

// lengthPrefix is the length of the message prefix for each message type
var lengthPrefix = map[MessageId]int{
	ChokeId:         1,
	UnchokeId:       1,
	InterestedId:    1,
	NotInterestedId: 1,
	HaveId:          5,
	RequestId:       13,
	CancelId:        13,
	PortId:          3,
}

// payloadSize is the size of the payload for each message type
var payloadSize = map[MessageId]int{
	ChokeId:         0,
	UnchokeId:       0,
	InterestedId:    0,
	NotInterestedId: 0,
	HaveId:          4,
	RequestId:       12,
	CancelId:        12,
	PortId:          2,
}

// KeepAlive is to send to peer to keep the connection alive
var KeepAlive = []byte{0, 0, 0, 0}

type Message struct {
	LengthPrefix int
	MessageId    MessageId
	Payload      []byte
}

// NewMessage creates a new message with the given id and payload.
func NewMessage(id MessageId, payload []byte) *Message {
	return &Message{
		LengthPrefix: lengthPrefix[id] + len(payload),
		MessageId:    id,
		Payload:      payload,
	}
}

// encodeMessage encodes the message into a byte slice.
func (m *Message) encodeMessage() []byte {
	buf := make([]byte, 4+m.LengthPrefix)

	binary.BigEndian.PutUint32(buf, uint32(m.LengthPrefix))
	buf[4] = byte(m.MessageId)
	if m.Payload != nil {
		copy(buf[5:], m.Payload)
	}

	return buf
}

// decodeMessage reads a message from the given byte slice.
func (m *Message) decodeMessage(buf []byte) {
	m.LengthPrefix = int(binary.BigEndian.Uint32(buf[:4]))

	if m.LengthPrefix == 0 {
		return // KeepAlive message
	}

	m.MessageId = MessageId(buf[4])

	var size int
	if s, ok := payloadSize[m.MessageId]; ok {
		size = s
	} else {
		size = m.LengthPrefix - 1
	}

	if size > 0 {
		m.Payload = make([]byte, size)
		copy(m.Payload, buf[5:5+size])
	} else {
		m.Payload = nil
	}
}
