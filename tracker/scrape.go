package tracker

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/JoelVCrasta/config"
)

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
func (c Connection) Scrape() (*ScrapeResponse, error) {
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
		return nil, err
	}
	c.conn.SetDeadline(time.Now().Add(config.Config.TrackerConnectTimeout))

	buf := make([]byte, 1024)
	_, err = c.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	log.Printf("Recieved %v from scrape request\n ", n)

	var s ScrapeResponse
	s.decodeScrapeResponse(buf)
	return &s, nil
}

// decodeScrapeResponse decodes the response from the scrape request and returns a ScrapeResponse object.
func (s *ScrapeResponse) decodeScrapeResponse(buf []byte) {
	s.action = binary.BigEndian.Uint32(buf)
	s.transactionId = binary.BigEndian.Uint32(buf[4:])
	s.seeders = binary.BigEndian.Uint32(buf[8:])
	s.leechers = binary.BigEndian.Uint32(buf[12:])
	s.completed = binary.BigEndian.Uint32(buf[16:])
}
