package client

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/JoelVCrasta/clover/handshake"
	"github.com/JoelVCrasta/clover/message"
	"github.com/JoelVCrasta/clover/peer"
)

type Client struct {
	peerChan   <-chan peer.Peer
	dedupePeer map[string]time.Time
	infoHash   [20]byte
	peerId     [20]byte
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// PeerInfo represents information about a peer connected to clover.
type ActivePeer struct {
	Peer        peer.Peer
	Conn        net.Conn
	PeerId      [20]byte
	Choked      bool
	Bitfield    Bitfield
	FailedCount int
}

func NewClient(peerChan <-chan peer.Peer, infoHash [20]byte, peerId [20]byte) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		peerChan:   peerChan,
		dedupePeer: make(map[string]time.Time),
		infoHash:   infoHash,
		peerId:     peerId,
		mu:         sync.Mutex{},
		ctx:        ctx,
		cancel:     cancel,
	}
}

/*
StartClient starts the client and listens for incoming peers.
It returns a channel of active peers that can be used to interact with the connected peers.
*/
func (c *Client) StartClient() <-chan *ActivePeer {
	activePeerChan := make(chan *ActivePeer, 500)

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case p, ok := <-c.peerChan:
				if !ok {
					return
				}
				// time.Sleep(100 * time.Millisecond)
				go c.AddPeer(p, activePeerChan)
			}
		}
	}()

	return activePeerChan
}

// AddPeer connects to a peer and adds it to the active peers list.
func (c *Client) AddPeer(p peer.Peer, apC chan<- *ActivePeer) {
	if c.ctx.Err() != nil {
		return // Client is stopped
	}

	if !c.validatePeer(p) {
		// log.Printf("[client] invalid peer: %s:%d", p.IpAddr, p.Port)
		return
	}

	conn, res, err := handshake.SendHandshake(c.infoHash, c.peerId, p.IpAddr, p.Port)
	if err != nil {
		// log.Printf("[client] failed to connect to peer %s:%d: %v", p.IpAddr, p.Port, err)
		return
	}

	bitfield, err := GetBitfieldFromPeer(conn)
	if err != nil {
		conn.Close()
		// log.Printf("[client] failed to read bitfield from peer %s:%d: %v", p.IpAddr, p.Port, err)
		return
	}

	// log.Printf("[client] connected to peer %s:%d", p.IpAddr, p.Port)

	activePeer := &ActivePeer{
		Peer:        p,
		Conn:        conn,
		PeerId:      res.PeerId,
		Choked:      true,
		Bitfield:    bitfield,
		FailedCount: 0,
	}

	select {
	case apC <- activePeer:
	case <-c.ctx.Done():
		activePeer.Disconnect()
		return
	default:
		activePeer.Disconnect()
	}
}

// RemovePeer removes a peer from the active peers list and disconnects it.
// func (c *Client) RemovePeer(ap *ActivePeer) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	for i, cap := range c.ActivePeers {
// 		if cap.Peer.IpAddr.Equal(ap.Peer.IpAddr) && cap.Peer.Port == ap.Peer.Port {
// 			cap.Disconnect()
// 			c.ActivePeers = slices.Delete(c.ActivePeers, i, i+1)
// 			delete(c.dedupePeer, cap.Peer.String())
// 		}
// 	}
// }

// validatePeer checks if the peer is valid and not already in the dedupe map.
func (c *Client) validatePeer(p peer.Peer) bool {
	if p.IpAddr == nil || p.IpAddr.IsUnspecified() {
		return false // Invalid IP address
	}

	key := p.String()

	c.mu.Lock()
	defer c.mu.Unlock()

	if lastSeen, exists := c.dedupePeer[key]; exists {
		if time.Since(lastSeen) < 5*time.Minute {
			return false // Cooldown period not over
		}
	}
	c.dedupePeer[key] = time.Time{}

	return true // New peer
}

// Disconnect closes the connection to the peer.
func (ap *ActivePeer) Disconnect() {
	if ap.Conn != nil {
		// log.Printf("[client] disconnecting from peer %s:%d", ap.Peer.IpAddr, ap.Peer.Port)
		_ = ap.Conn.Close()
		ap.Conn = nil
	}
}

// StopClient stops the client
func (c *Client) StopClient() {
	c.cancel()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dedupePeer = make(map[string]time.Time)
	// log.Println("[client] stopped")
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
