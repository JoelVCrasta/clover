package message

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

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

	Extended = 20
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
		LengthPrefix: 1 + len(payload),
		MessageId:    id,
		Payload:      payload,
	}
}

/*
decodeMessage reads a message from the given connection.
It checks the length of the message, if the length is 0, it returns a KeepAlive message.
If the length is greater than 0, then it is decoded into a Message struct.
*/
func ReadMessage(conn net.Conn) (*Message, error) {
	reader := bufio.NewReader(conn)
	lengthBuf := make([]byte, 4)

	if _, err := io.ReadFull(reader, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		return nil, nil // KeepAlive message
	}

	msg := make([]byte, length)
	if _, err := io.ReadFull(reader, msg); err != nil {
		return nil, err
	}

	fullMessage := make([]byte, 4+length)
	copy(fullMessage, lengthBuf)
	copy(fullMessage[4:], msg)

	var m Message
	m.decodeMessage(fullMessage)
	return &m, nil
}

func ReadPieceMessage(reader io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)

	if _, err := io.ReadFull(reader, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		return nil, nil // KeepAlive message
	}

	msg := make([]byte, length)
	if _, err := io.ReadFull(reader, msg); err != nil {
		return nil, err
	}

	fullMessage := make([]byte, 4+length)
	copy(fullMessage, lengthBuf)
	copy(fullMessage[4:], msg)

	var m Message
	m.decodeMessage(fullMessage)
	return &m, nil
}

// encodeMessage encodes the message into a byte slice.
func (m *Message) EncodeMessage() []byte {
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
	if m.MessageId == PieceId || m.MessageId == BitfieldId {
		size = m.LengthPrefix - 1
	} else {
		size = payloadSize[m.MessageId]
	}

	if size > 0 {
		m.Payload = make([]byte, size)
		copy(m.Payload, buf[5:5+size])
	} else {
		m.Payload = nil
	}
}

// decodeHave decodes a Have message from the peer and updates the peer's bitfield.
func (m *Message) DecodeHave() (int, error) {
	if len(m.Payload) != payloadSize[m.MessageId] {
		return 0, fmt.Errorf("invalid Have payload length: %d", len(m.Payload))
	}

	pieceIndex := binary.BigEndian.Uint32(m.Payload[:])

	return int(pieceIndex), nil
}

// decodeBitfield decodes a Bitfield message from the peer and returns the bitfield as a byte slice.
func (m *Message) DecodeBitfield() ([]byte, error) {
	if len(m.Payload) < 1 {
		return nil, fmt.Errorf("invalid Bitfield payload length: %d", len(m.Payload))
	}

	bitfield := make([]byte, len(m.Payload))
	copy(bitfield, m.Payload)

	return bitfield, nil
}

// DecodePiece decodes a Piece message from the peer and writes the block to the provided buffer.
func (m *Message) DecodePiece(expectedIndex, bufLength int) (int, []byte, error) {
	if len(m.Payload) < 8 {
		return 0, nil, fmt.Errorf("invalid Piece payload length: %d", len(m.Payload))
	}

	index := int(binary.BigEndian.Uint32(m.Payload[:4]))
	offset := int(binary.BigEndian.Uint32(m.Payload[4:8]))
	block := m.Payload[8:]

	if index != expectedIndex {
		return 0, nil, fmt.Errorf("piece index mismatch: expected %d, got %d", expectedIndex, index)
	}
	if offset+len(block) > bufLength {
		return 0, nil, fmt.Errorf("block exceeds buffer size: offset %d, block size %d, buffer size %d", offset, len(block), bufLength)
	}

	return offset, block, nil
}
