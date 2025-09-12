package tracker

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/JoelVCrasta/clover/config"
	"github.com/JoelVCrasta/clover/peer"
)

type TrackerManager struct {
	// Trackers    []*Tracker
	trackerUrls []string
	infoHash    [20]byte
	peerId      [20]byte
	ctx         context.Context
	cancel      context.CancelFunc
}

// Maybe needed for stats and reconnects in future
// type Tracker struct {
// 	Url          string
// 	Connection   *Connection
// 	Interval     time.Duration
// 	LastAnnounce time.Time
// }

type Connection struct {
	conn          *net.UDPConn
	connectionId  uint64
	transactionId uint32
}

type AnnounceRequest struct {
	ConnectionId  uint64
	Action        uint32
	TransactionId uint32
	InfoHash      [20]byte
	PeerId        [20]byte
	Downloaded    uint64
	Left          uint64
	Uploaded      uint64
	Event         uint32
	IpAddr        uint32
	Key           uint32
	Numwant       uint32
	Port          uint16
}

type AnnounceResponse struct {
	Action        uint32
	TransactionId uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	Peers         []peer.Peer
}

func NewTrackerManager(trackerUrls []string, infoHash, peerId [20]byte) *TrackerManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &TrackerManager{
		// Trackers:    make([]*Tracker, 0),
		trackerUrls: trackerUrls,
		infoHash:    infoHash,
		peerId:      peerId,
		ctx:         ctx,
		cancel:      cancel,
	}
}

/*
Start connects to all trackers and starts announcing.
It will periodically re-announce to the trackers.
It returns a channel of Peer objects that can be used to connect to peers.
*/
func (tm *TrackerManager) StartTracker() (<-chan peer.Peer, error) {
	peerChan := make(chan peer.Peer)

	for _, url := range tm.trackerUrls {
		go func(trackerUrl string) {
			conn, err := ConnectTracker(trackerUrl)
			if err != nil {
				log.Printf("[tracker] failed to connect to tracker %s: %v", trackerUrl, err)
				return
			}
			defer conn.Close()

			// Create an announce request
			arq := AnnounceRequest{
				Key:        rand.Uint32(),
				InfoHash:   tm.infoHash,
				IpAddr:     0,
				Port:       6881,
				Uploaded:   0,
				Downloaded: 0,
				Left:       500,
				Event:      0,
				Numwant:    50,
			}

			response, err := conn.AnnounceTracker(arq, tm.peerId)
			if err != nil {
				log.Printf("[tracker] announce failed for %s: %v", trackerUrl, err)
				return
			}

			log.Printf("[tracker] recieved peers from %s", trackerUrl)

			for _, p := range response.Peers {
				if p.IpAddr.IsUnspecified() {
					continue
				}

				select {
				case peerChan <- p:
				case <-tm.ctx.Done():
					return
				}

			}

			// Periodically re-announce to the tracker
			interval := response.Interval
			if response.Interval <= 0 {
				interval = config.Config.DefaultTrackerInterval
			}

			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					response, err = conn.AnnounceTracker(arq, tm.peerId)
					if err != nil {
						log.Printf("[tracker] re-announce failed for %s: %v", trackerUrl, err)
						return
					}

					for _, p := range response.Peers {
						if p.IpAddr.IsUnspecified() {
							continue
						}

						select {
						case peerChan <- p:
						case <-tm.ctx.Done():
							return
						}
					}

				case <-tm.ctx.Done():
					return
				}
			}

		}(url)
	}

	return peerChan, nil
}

// Stop stops the tracker manager and closes all connections.
func (tm *TrackerManager) StopTracker() {
	if tm.cancel != nil {
		tm.cancel()
	}

	log.Println("[tracker] stopped trackers")
}

/*
ConnectTracker establishes a UDP connection to the tracker.
It sends a connection packet which includes a BitTorrent UDP magic constant, action, and transaction ID.
It returns a Connection object containing the trackers connection ID and transaction ID.
*/
func ConnectTracker(trackerUrl string) (*Connection, error) {
	udpAddress, err := net.ResolveUDPAddr("udp", trackerUrl)
	if err != nil {
		return nil, err
	}

	if udpAddress.IP.String() == "127.0.0.1" {
		return nil, fmt.Errorf("resolving to localhost is not allowed")
	}

	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return nil, err
	}

	connectionPacket := getConnectionPacket()

	_, err = conn.Write(connectionPacket[:])
	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(config.Config.TrackerConnectTimeout))
	buf := make([]byte, 32)
	_, err = conn.Read(buf)

	conn.SetDeadline(time.Time{})
	if err != nil {
		return nil, err
	}

	action := binary.BigEndian.Uint32(buf)
	transactionId := binary.BigEndian.Uint32(buf[4:])
	connectionId := binary.BigEndian.Uint64(buf[8:])

	if action == 3 {
		return nil, fmt.Errorf("tracker returned error")
	}

	// log.Printf("Recieved %v bytes from tracker connection request\n", n)
	// log.Printf("Response from server, Action: %d, transactionId: %d, connectionId: %d\n", action, transactionId, connectionId)

	return &Connection{
		conn:          conn,
		connectionId:  connectionId,
		transactionId: transactionId,
	}, nil
}

// Close closes the UDP connection to the tracker.
func (c *Connection) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

/*
AnnounceTracker sends an announce request to the tracker.
It includes the action, transaction ID, info hash, peer ID, and other parameters.
It returns an AnnounceResponse object containing the response from the tracker.
*/
func (c Connection) AnnounceTracker(arq AnnounceRequest, peerId [20]byte) (*AnnounceResponse, error) {

	arq.Action = 1
	arq.ConnectionId = c.connectionId
	arq.TransactionId = rand.Uint32()
	arq.PeerId = peerId

	packet := make([]byte, 98)
	binary.BigEndian.PutUint64(packet, arq.ConnectionId)       // 8 bytes
	binary.BigEndian.PutUint32(packet[8:], arq.Action)         // 4 bytes
	binary.BigEndian.PutUint32(packet[12:], arq.TransactionId) // 4 bytes
	copy(packet[16:], arq.InfoHash[:])                         // 20 bytes
	copy(packet[36:], arq.PeerId[:])                           // 20 bytes
	binary.BigEndian.PutUint64(packet[56:], arq.Downloaded)    // 8 bytes
	binary.BigEndian.PutUint64(packet[64:], arq.Left)          // 8 bytes
	binary.BigEndian.PutUint64(packet[72:], arq.Uploaded)      // 8 bytes
	binary.BigEndian.PutUint32(packet[80:], arq.Event)         // 4 bytes
	binary.BigEndian.PutUint32(packet[84:], arq.IpAddr)        // 4 bytes
	binary.BigEndian.PutUint32(packet[88:], arq.Key)           // 4 bytes
	binary.BigEndian.PutUint32(packet[92:], arq.Numwant)       // 4 bytes
	binary.BigEndian.PutUint16(packet[96:], arq.Port)          // 2 bytes

	// log.Println("Announce packet:", packet[:])

	n, err := c.conn.Write(packet[:])
	if err != nil {
		return nil, err
	}
	c.conn.SetDeadline(time.Now().Add(config.Config.TrackerConnectTimeout))

	buf := make([]byte, 1024)
	_, err = c.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// log.Printf("Recieved %v bytes from announce request\n ", n)

	var a AnnounceResponse
	a.decodeAnnounceResponse(buf, n)

	if a.Action == 3 {
		return nil, fmt.Errorf("tracker error on announce")
	}

	return &a, nil
}

// decodeAnnounceResponse decodes the response from the announce request.and returns an AnnounceResponse object.
func (a *AnnounceResponse) decodeAnnounceResponse(buf []byte, n int) {
	peersCount := (n - 20) / 6
	peers := make([]peer.Peer, peersCount)

	a.Action = binary.BigEndian.Uint32(buf[0:])
	a.TransactionId = binary.BigEndian.Uint32(buf[4:])
	a.Interval = binary.BigEndian.Uint32(buf[8:])
	a.Leechers = binary.BigEndian.Uint32(buf[12:])
	a.Seeders = binary.BigEndian.Uint32(buf[16:])
	a.Peers = peers

	for i := range peersCount {
		offset := i * 6

		a.Peers[i].IpAddr = net.IP(buf[20+offset : 20+offset+4])
		a.Peers[i].Port = binary.BigEndian.Uint16(buf[24+offset:])
	}
}

type ConnPacket [16]byte

// getConnectionPacket creates a connection packet for the BitTorrent protocol.
func getConnectionPacket() ConnPacket {
	var cp ConnPacket

	magicConstant := uint64(0x41727101980)
	action := uint32(0)
	transactionId := rand.Uint32()

	binary.BigEndian.PutUint64(cp[0:8], magicConstant)
	binary.BigEndian.PutUint32(cp[8:12], action)
	binary.BigEndian.PutUint32(cp[12:16], transactionId)

	return cp
}
