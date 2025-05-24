package client

import (
	"encoding/binary"
	"net"
	"sync"
)

// Client represents a torrent client that manages connections to peers and downloads pieces of the torrent.
type Client struct {
	InfoHash    [20]byte
	Peers       []*PeerInfo
	PieceLength int
	TotalLength int

	Pieces     []Piece
	PieceQueue chan int
	Downloaded map[int]bool

	Mutex sync.Mutex
}

// PeerInfo represents information about a peer connected to clover.
type PeerInfo struct {
	IpAddr   net.IP
	Port     uint16
	Conn     net.Conn
	PeerId   [20]byte
	Choked   bool
	Bitfield Bitfield
}

type Piece struct {
	Index int
	Hash  [20]byte
	Size  int

	Data       []byte
	Downloaded bool
}

/*
readMessage reads a message from the given connection.
It checks the length of the message, if the length is 0, it returns a KeepAlive message.
If the length is greater than 0, it reads the message and returns the message ID and payload.
*/
func ReadMessage(conn net.Conn) (id byte, payload []byte, err error) {
	lengthBuf := make([]byte, 4)

	if _, err = conn.Read(lengthBuf); err != nil {
		return 0, nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		return 0, nil, nil // KeepAlive message
	}

	msg := make([]byte, length)
	if _, err = conn.Read(msg); err != nil {
		return
	}

	id = msg[0]
	payload = msg[1:]

	return id, payload, nil
}
