package torrent

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

type Connection struct {
	conn          *net.UDPConn
	connectionId  uint64
	transactionId uint32
}

type AnnounceRequest struct {
	Key        uint32
	Action     uint32
	InfoHash   [20]byte
	PeerId     [20]byte
	IpAddr     uint32
	Port       uint16
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      uint32
	Numwant    uint32

	ConnectionId  uint64
	TransactionId uint32
}

type AnnounceResponse struct {
	action        uint32
	transactionId uint32
	interval      uint32
	leechers      uint32
	seeders       uint32
	Peers         []Peer
}

type Peer struct {
	IpAddr uint32
	Port   uint16
}

/*
NewUDPTrackerConnection establishes a UDP connection to the tracker.
It sends a connection packet which includes a BitTorrent UDP magic constant, action, and transaction ID.
It returns a Connection object containing the trackers connection ID and transaction ID.
*/
func NewUDPTrackerConnection(trackerUrl string) (*Connection, error) {
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
	log.Println("Connection packet:", connectionPacket)

	_, err = conn.Write(connectionPacket[:])
	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 32)
	n, err := conn.Read(buf)

	conn.SetDeadline(time.Time{})
	if err != nil {
		return nil, err
	}

	action := binary.BigEndian.Uint32(buf)
	connectionId := binary.BigEndian.Uint64(buf[4:])
	transactionId := binary.BigEndian.Uint32(buf[8:])

	log.Printf("Recieved %v from announce request\n ", n)
	log.Println("Received response:", action, connectionId, transactionId)

	return &Connection{
		conn:          conn,
		connectionId:  connectionId,
		transactionId: transactionId,
	}, nil
}

/*
TrackerAnnounce sends an announce request to the tracker.
It includes the action, transaction ID, info hash, peer ID, and other parameters.
It returns an AnnounceResponse object containing the response from the tracker.
*/
func (c Connection) TrackerAnnounce(arq AnnounceRequest, peerId [20]byte) (AnnounceResponse, error) {
	arq.Action = 1
	arq.ConnectionId = c.connectionId
	arq.TransactionId = c.transactionId
	arq.PeerId = peerId

	packet := make([]byte, 98)
	binary.BigEndian.PutUint64(packet, arq.ConnectionId)       // 8 bytes
	binary.BigEndian.PutUint32(packet[8:], arq.Action)         // 4 bytes
	binary.BigEndian.PutUint32(packet[12:], arq.TransactionId) // 4 bytes
	copy(packet[16:], arq.InfoHash[:])                         // 20 bytes
	copy(packet[36:], arq.PeerId[:])                           // 20 bytes
	binary.BigEndian.PutUint64(packet[56:], arq.Downloaded)    // 2 bytes
	binary.BigEndian.PutUint64(packet[64:], arq.Left)          // 8 bytes
	binary.BigEndian.PutUint64(packet[72:], arq.Uploaded)      // 8 bytes
	binary.BigEndian.PutUint32(packet[80:], arq.Event)         // 2 bytes
	binary.BigEndian.PutUint32(packet[84:], arq.IpAddr)        // 4 bytes
	binary.BigEndian.PutUint32(packet[88:], arq.Key)           // 4 bytes
	binary.BigEndian.PutUint32(packet[92:], arq.Numwant)       // 4 bytes
	binary.BigEndian.PutUint16(packet[96:], arq.Port)          // 2 bytes

	n, err := c.conn.Write(packet[:])
	if err != nil {
		return AnnounceResponse{}, err
	}
	c.conn.SetDeadline(time.Now().Add(time.Second * 10))

	buf := make([]byte, 1024)
	_, err = c.conn.Read(buf)
	if err != nil {
		return AnnounceResponse{}, err
	}

	log.Printf("Recieved %v from announce request\n ", n)

	decoded := decodeAnnounceResponse(buf, n)
	return decoded, nil
}

type ScrapeRequest struct {
	ConnectionId  uint64
	Action        uint32
	TransactionId uint32
	InfoHash      [20]byte
}

type ScrapeResponse struct {
	action        uint32
	transactionId uint32
	seeders       uint32
	leechers      uint32
	completed     uint32
}

/*
Scrape sends a scrape request to the tracker.
It includes the connectionID, action, transaction ID, and info hash.
It returns a ScrapeResponse object containing the response from the tracker.
*/
func (c Connection) Scrape() (ScrapeResponse, error) {
	sr := ScrapeRequest{
		ConnectionId:  c.connectionId,
		Action:        2,
		TransactionId: c.transactionId,
	}

	packet := make([]byte, 32)
	binary.BigEndian.PutUint64(packet, sr.ConnectionId)       // 8 bytes
	binary.BigEndian.PutUint32(packet[8:], sr.Action)         // 4 bytes
	binary.BigEndian.PutUint32(packet[12:], sr.TransactionId) // 4 bytes
	copy(packet[16:], sr.InfoHash[:])                         // 20 bytes

	n, err := c.conn.Write(packet[:])
	if err != nil {
		return ScrapeResponse{}, err
	}
	c.conn.SetDeadline(time.Now().Add(time.Second * 10))

	buf := make([]byte, 1024)
	_, err = c.conn.Read(buf)
	if err != nil {
		return ScrapeResponse{}, err
	}

	log.Printf("Recieved %v from scrape request\n ", n)

	scrapeResponse := 
	return scrapeResponse, nil

}

// decodeAnnounceResponse decodes the response from the announce request.and returns an AnnounceResponse object.
func decodeAnnounceResponse(buf []byte, n int) AnnounceResponse {
	peersCount := (n - 20) / 6
	peers := make([]Peer, peersCount)

	arp := AnnounceResponse{
		action:        binary.BigEndian.Uint32(buf),
		transactionId: binary.BigEndian.Uint32(buf[4:]),
		interval:      binary.BigEndian.Uint32(buf[8:]),
		leechers:      binary.BigEndian.Uint32(buf[12:]),
		seeders:       binary.BigEndian.Uint32(buf[16:]),
		Peers:         peers,
	}

	for i := range peersCount {
		peers[i].IpAddr = binary.BigEndian.Uint32(buf[20+i*6:])
		peers[i].Port = binary.BigEndian.Uint16(buf[24+i*6:])
	}

	return arp
}

// decodeScrapeResponse decodes the response from the scrape request and returns a ScrapeResponse object.
func decodeScrapeResponse(buf []byte, n int) ScrapeResponse {
	sr := ScrapeResponse{
		action:        binary.BigEndian.Uint32(buf),
		transactionId: binary.BigEndian.Uint32(buf[4:]),
		seeders:       binary.BigEndian.Uint32(buf[8:]),
		leechers:      binary.BigEndian.Uint32(buf[12:]),
		completed:     binary.BigEndian.Uint32(buf[16:]),
	}
	
	return sr
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
