package torrent

import (
	"log"
	"net"
	"strconv"
	"time"
)

type Handshake struct {
	Pstrlen  byte
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerId   [20]byte
}

func NewHandshake(infoHash, peerId [20]byte, peerIp uint32, peerPort uint16) (*Handshake, error) {
	request := getHandshakePayload(infoHash, peerId)
	ip := net.IPv4(byte(peerIp>>24), byte(peerIp>>16), byte(peerIp>>8), byte(peerIp))
	peerAddress := net.JoinHostPort(ip.String(), strconv.Itoa(int(peerPort)))

	conn, err := net.Dial("tcp", peerAddress)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = conn.Write(request)
	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(time.Second * 10))

	buf := make([]byte, 68)
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}

	log.Println("Handshake response:", buf)

	var h Handshake
	h.decodeHandshakeResponse(buf)
	return &h, nil
}

func getHandshakePayload(infoHash, peerId [20]byte) []byte {
	handshake := make([]byte, 68)
	reserved := [8]byte{}

	handshake[0] = 19
	copy(handshake[1:], []byte("BitTorrent protocol"))
	copy(handshake[20:], reserved[:])
	copy(handshake[28:], infoHash[:])
	copy(handshake[48:], peerId[:])

	return handshake
}

func (h *Handshake) decodeHandshakeResponse(buf []byte) {
	h.Pstrlen = buf[0]
	h.Pstr = string(buf[1:20])
	copy(h.Reserved[:], buf[20:28])
	copy(h.InfoHash[:], buf[28:48])
	copy(h.PeerId[:], buf[48:68])
}
