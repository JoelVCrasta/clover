package client

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

func (m *Message) NewMessage(id MessageId, payload []byte) *Message {
	return &Message{
		LengthPrefix: lengthPrefix[id] + len(payload),
		MessageId:    id,
		Payload:      payload,
	}
}

