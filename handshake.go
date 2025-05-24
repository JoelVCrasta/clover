package torrent

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/JoelVCrasta/config"
)

type Handshake struct {
	Pstrlen  byte
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerId   [20]byte
}

func NewHandshake(infoHash, peerId [20]byte, peerIp net.IP, peerPort uint16) (net.Conn, *Handshake, error) {
	request := getHandshakePayload(infoHash, peerId)

	peerAddress := net.JoinHostPort(peerIp.String(), strconv.Itoa(int(peerPort)))

	log.Println("Peer address:", peerAddress)

	conn, err := net.DialTimeout("tcp", peerAddress, config.Config.PeerHandshakeTimeout)
	if err != nil {
		return nil, nil, err
	}

	_, err = conn.Write(request)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	conn.SetDeadline(time.Now().Add(time.Second * 10))

	buf := make([]byte, 68)
	_, err = conn.Read(buf)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	var h Handshake
	h.decodeHandshakeResponse(buf)
	return conn, &h, nil
}

func getHandshakePayload(infoHash, peerId [20]byte) []byte {
	handshake := make([]byte, 68)

	handshake[0] = 19
	copy(handshake[1:], "BitTorrent protocol")
	copy(handshake[20:], make([]byte, 8))
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
