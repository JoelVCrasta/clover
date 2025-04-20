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
	IpAddr     net.IP
	Port       uint16
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      string
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
	peers         []Peer
}

type Peer struct {
	IpAddr net.IP
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
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Time{})

	action := binary.BigEndian.Uint32(buf)
	connectionId := binary.BigEndian.Uint64(buf[4:])
	transactionId := binary.BigEndian.Uint32(buf[8:])

	log.Println("Received response:", action, connectionId, transactionId)

	return &Connection{
		conn:          conn,
		connectionId:  connectionId,
		transactionId: transactionId,
	}, nil
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


