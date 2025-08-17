package tracker

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/JoelVCrasta/config"
)

type TrackerManager struct {
	Trackers []*Tracker
}

type Tracker struct {
	Url          string
	Connection   *Connection
	Interval     time.Duration
	LastAnnounce time.Time
}

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
	Peers         []Peer
}

type Peer struct {
	IpAddr net.IP
	Port   uint16
}

func NewTrackerManager() *TrackerManager {
	return &TrackerManager{
		Trackers: make([]*Tracker, 0),
	}
}

/*
ConnectTrackerAll connects to all trackers provided in the trackerUrls (AnnounceList).
It initializes a Tracker object for each URL and appends it to the Trackers slice.
If no trackers are connected, it returns an error.
*/
func (tm *TrackerManager) ConnectTrackerAll(trackerUrls []string) error {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		limit = config.Config.MaxTrackerConnections // limit the number of concurrent tracker connections
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, url := range trackerUrls {
		if ctx.Err() != nil {
			break // exit to stop new spawning goroutines
		}

		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			if ctx.Err() != nil {
				return // exit if context is cancelled
			}

			conn, err := ConnectTracker(url)
			if err != nil {
				log.Printf("[tracker] Failed to connect to tracker %s: %v", url, err)
				return
			}

			mu.Lock()
			if len(tm.Trackers) < limit {
				tm.Trackers = append(tm.Trackers, &Tracker{
					Url:          url,
					Connection:   conn,
					Interval:     config.Config.DefaultTrackerInterval,
					LastAnnounce: time.Time{},
				})
				log.Printf("[tracker] Connected to tracker: %s", url)

				if len(tm.Trackers) == limit {
					cancel()
				}
			}
			mu.Unlock()
		}(url)
	}

	wg.Wait()

	if len(tm.Trackers) == 0 {
		return fmt.Errorf("no trackers connected")
	}
	return nil
}

/*
AnnounceTrackerAll announces to all trackers managed by the TrackerManager.
It sends an AnnounceRequest to each tracker and collects the peers returned by each tracker.
It returns a slice of Peer objects containing the peers from all trackers.
*/
func (tm *TrackerManager) AnnounceTrackerAll(arq AnnounceRequest, peerId [20]byte) []Peer {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		peerSet  = make(map[string]bool) // to avoid duplicate peers
		allPeers []Peer
	)

	for _, tracker := range tm.Trackers {
		if time.Since(tracker.LastAnnounce) < tracker.Interval {
			continue
		}

		wg.Add(1)
		go func(t *Tracker) {
			defer wg.Done()

			conn := t.Connection
			if conn == nil {
				log.Printf("[tracker] No connection for tracker: %s", t.Url)
				return
			}

			response, err := conn.AnnounceTracker(arq, peerId)
			if err != nil {
				log.Printf("[tracker] Failed to announce to tracker %s: %v", t.Url, err)
				return
			}

			mu.Lock()
			t.LastAnnounce = time.Now()
			t.Interval = time.Duration(response.Interval) * time.Second

			for _, peer := range response.Peers {
				key := fmt.Sprintf("%s:%d", peer.IpAddr.String(), peer.Port)
				if !peerSet[key] {
					peerSet[key] = true
					allPeers = append(allPeers, peer)
				}

			}
			mu.Unlock()

			log.Printf("[tracker] Announced to tracker %s successfully", t.Url)
		}(tracker)
	}

	wg.Wait()
	return allPeers
}

// Close closes all tracker connections managed by the TrackerManager.
func (tm *TrackerManager) Close() {
	for _, tracker := range tm.Trackers {
		if tracker.Connection != nil {
			tracker.Connection.Close()
			log.Printf("Closed connection to tracker: %s", tracker.Url)
		}
	}
	tm.Trackers = nil
}

/*
UDPTrackerConnection establishes a UDP connection to the tracker.
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
TrackerAnnounce sends an announce request to the tracker.
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
	return &a, nil
}

// decodeAnnounceResponse decodes the response from the announce request.and returns an AnnounceResponse object.
func (a *AnnounceResponse) decodeAnnounceResponse(buf []byte, n int) {
	peersCount := (n - 20) / 6
	peers := make([]Peer, peersCount)

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
