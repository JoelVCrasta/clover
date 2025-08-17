package client

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/JoelVCrasta/handshake"
	"github.com/JoelVCrasta/message"
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
	IpAddr      net.IP
	Port        uint16
	Conn        net.Conn
	PeerId      [20]byte
	Choked      bool
	Bitfield    Bitfield
	FailedCount int
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
				log.Printf("[client] Failed to connect to peer %s:%d - %v", peer.IpAddr, peer.Port, err)
				return
			}

			bitfield, err := GetBitfieldFromPeer(conn)
			if err != nil {
				log.Printf("[client] Failed to read bitfield to peer %s:%d - %v", peer.IpAddr, peer.Port, err)
				conn.Close()
				return
			}

			log.Printf("[client] Connected to peer %s:%d", peer.IpAddr, peer.Port)

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

// Disconnect closes the connection to the peer and sets the Conn field to nil.
func (p *ActivePeer) Disconnect() {
	if p.Conn != nil {
		log.Printf("Disconnecting from peer %s:%d", p.IpAddr, p.Port)
		_ = p.Conn.Close()
		p.Conn = nil
	}
}

// GetBitfieldFromPeer reads the bitfield message right after the handshake done with the peer.
func GetBitfieldFromPeer(conn net.Conn) (Bitfield, error) {
	msg, err := message.ReadMessage(conn)
	if err != nil {
		return nil, err
	}

	if msg.MessageId != message.BitfieldId {
		return nil, fmt.Errorf("expected Bitfield message, got %v", msg.MessageId)
	}

	bitfield := make(Bitfield, len(msg.Payload))
	copy(bitfield, msg.Payload)

	return bitfield, nil
}

// ------------ Messages ------------

func (ap *ActivePeer) SendChoke() error {
	choke := message.NewMessage(message.ChokeId, nil)
	_, err := ap.Conn.Write(choke.EncodeMessage())

	return err
}

func (ap *ActivePeer) SendUnchoke() error {
	unchoke := message.NewMessage(message.UnchokeId, nil)
	_, err := ap.Conn.Write(unchoke.EncodeMessage())

	return err
}

func (ap *ActivePeer) SendInterested() error {
	interested := message.NewMessage(message.InterestedId, nil)
	_, err := ap.Conn.Write(interested.EncodeMessage())

	return err
}

func (ap *ActivePeer) SendNotInterested() error {
	notInterested := message.NewMessage(message.NotInterestedId, nil)
	_, err := ap.Conn.Write(notInterested.EncodeMessage())

	return err
}

func (ap *ActivePeer) SendHave(pieceIndex int) error {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(pieceIndex))

	have := message.NewMessage(message.HaveId, payload)
	_, err := ap.Conn.Write(have.EncodeMessage())

	return err
}

func (ap *ActivePeer) SendRequest(pieceIndex, offset, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	request := message.NewMessage(message.RequestId, payload)
	_, err := ap.Conn.Write(request.EncodeMessage())

	return err
}

func (ap *ActivePeer) SendCancel(pieceIndex, offset, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(pieceIndex))
	binary.BigEndian.PutUint32(payload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	cancel := message.NewMessage(message.CancelId, payload)
	_, err := ap.Conn.Write(cancel.EncodeMessage())

	return err
}
