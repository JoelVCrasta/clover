package client

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
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
	ActivePeers []*ActivePeer
	Torrent     parsing.Torrent

	StartTime time.Time
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

/*
NewClient initializes a new torrent client with the given torrent and peers.
It connects to each peer, performs a handshake, and retrieves the bitfield from each peer.
It returns a pointer to the Client and an error if any occurred during the process.
*/
func NewClient(torrent parsing.Torrent, peers []tracker.Peer, peerId [20]byte) (*Client, error) {
	c := &Client{
		Torrent:   torrent,
		StartTime: time.Now(),
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

// GetBitfieldFromPeer reads the bitfield message right after the handshake done with the peer.
func GetBitfieldFromPeer(conn net.Conn) (Bitfield, error) {
	msg, err := readMessage(conn)
	if err != nil {
		return nil, err
	}

	if msg.MessageId != BitfieldId {
		return nil, fmt.Errorf("expected Bitfield message, got %v", msg.MessageId)
	}

	bitfield := make(Bitfield, len(msg.Payload))
	copy(bitfield, msg.Payload)

	return bitfield, nil
}

/*
decodeMessage reads a message from the given connection.
It checks the length of the message, if the length is 0, it returns a KeepAlive message.
If the length is greater than 0, then it is decoded into a Message struct.
*/
func readMessage(conn net.Conn) (*Message, error) {
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

	var message Message
	message.decodeMessage(fullMessage)
	return &message, nil
}

// ------------ Messages ------------

func (ap *ActivePeer) SendChoke() error {
	message := NewMessage(ChokeId, nil)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}

func (ap *ActivePeer) SendUnchoke() error {
	message := NewMessage(UnchokeId, nil)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}

func (ap *ActivePeer) SendInterested() error {
	message := NewMessage(InterestedId, nil)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}

func (ap *ActivePeer) SendNotInterested() error {
	message := NewMessage(NotInterestedId, nil)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}

func (ap *ActivePeer) SendHave(pieceIndex int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(pieceIndex))

	message := NewMessage(HaveId, payload)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}

func (ap *ActivePeer) SendRequest(pieceIndex, offset, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	message := NewMessage(RequestId, payload)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}

func (ap *ActivePeer) SendCancel(pieceIndex, offset, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	message := NewMessage(CancelId, payload)
	_, err := ap.Conn.Write(message.encodeMessage())

	return err
}
