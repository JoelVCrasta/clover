package client

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/JoelVCrasta/handshake"
	"github.com/JoelVCrasta/parsing"
	"github.com/JoelVCrasta/tracker"
)

// Client represents a torrent client that manages connections to peers and downloads pieces of the torrent.
type Client struct {
	InfoHash    [20]byte
	ActivePeers []*ActivePeer
	PieceLength int
	TotalLength int
	StartTime   time.Time

	PieceQueue chan int
	Downloaded map[int]bool
	Requested  map[int]bool

	Mutex sync.Mutex
}

// PeerInfo represents information about a peer connected to clover.
type ActivePeer struct {
	IpAddr   net.IP
	Port     uint16
	Conn     net.Conn
	PeerId   [20]byte
	Choked   bool
	Bitfield Bitfield
}

func NewClient(torrent parsing.Torrent, peers []tracker.Peer, peerId [20]byte) (*Client, error) {
	c := &Client{
		InfoHash:    torrent.InfoHash,
		PieceLength: torrent.Info.PieceLength,
		TotalLength: torrent.Info.Length,
		StartTime:   time.Now(),

		PieceQueue: make(chan int, torrent.PieceCount),
		Downloaded: make(map[int]bool),
		Requested:  make(map[int]bool),

		Mutex: sync.Mutex{},
	}

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		connectCount int = 0
	)

	for _, peer := range peers {
		wg.Add(1)

		go func(peer tracker.Peer) {
			defer wg.Done()

			conn, res, err := handshake.NewHandshake(torrent.InfoHash, peerId, peer.IpAddr, peer.Port)
			if err != nil {
				log.Printf("Failed to connect to peer %s:%d - %v", peer.IpAddr, peer.Port, err)
				return
			}

			bitfield, err := GetBitfieldFromPeer(conn)
			if err != nil {
				log.Printf("Failed to read bitfield to peer %s:%d - %v", peer.IpAddr, peer.Port, err)
				conn.Close()
				return
			}

			log.Printf("Connected to peer %s:%d", peer.IpAddr, peer.Port)

			active := &ActivePeer{
				IpAddr:   peer.IpAddr,
				Port:     peer.Port,
				Conn:     conn,
				PeerId:   res.PeerId,
				Choked:   true,
				Bitfield: bitfield,
			}

			mu.Lock()
			c.ActivePeers = append(c.ActivePeers, active)
			connectCount++
			mu.Unlock()
		}(peer)
	}
	wg.Wait()

	if connectCount == 0 {
		return nil, fmt.Errorf("no peers connected")
	}

	return c, nil
}

func GetBitfieldFromPeer(conn net.Conn) (Bitfield, error) {
	id, payload, err := ReadMessage(conn)
	if err != nil {
		return nil, err
	}

	if id != byte(BitfieldId) {
		return nil, nil // Not a Bitfield message
	}

	bitfield := make(Bitfield, len(payload))
	copy(bitfield, payload)

	return bitfield, nil
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

func (c *Client) SendChoke(peer *ActivePeer) error {
	message := NewMessage(ChokeId, nil)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}

func (c *Client) SendUnchoke(peer *ActivePeer) error {
	message := NewMessage(UnchokeId, nil)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}

func (c *Client) SendInterested(peer *ActivePeer) error {
	message := NewMessage(InterestedId, nil)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}

func (c *Client) SendNotInterested(peer *ActivePeer) error {
	message := NewMessage(NotInterestedId, nil)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}

func (c *Client) SendHave(peer *ActivePeer, pieceIndex int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(pieceIndex))

	message := NewMessage(HaveId, payload)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}

func (c *Client) SendRequest(peer *ActivePeer, pieceIndex, offset, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	message := NewMessage(RequestId, payload)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}

func (c *Client) SendCancel(peer *ActivePeer, pieceIndex, offset, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	message := NewMessage(CancelId, payload)
	_, err := peer.Conn.Write(message.encodeMessage())

	return err
}
