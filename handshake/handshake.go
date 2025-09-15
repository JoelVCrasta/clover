package handshake

import (
	"net"
	"strconv"
	"time"

	"github.com/JoelVCrasta/clover/config"
)

type Handshake struct {
	Pstrlen  byte
	Pstr     string
	Reserved [8]byte
	InfoHash [20]byte
	PeerId   [20]byte
}

/*
NewHandshake establishes a TCP connection to a peer and performs the BitTorrent handshake.
It sends a handshake request containing the info hash and peer ID, and waits for a response.
It returns the connection, the handshake response, and any error encountered.
*/
func SendHandshake(infoHash, peerId [20]byte, peerIp net.IP, peerPort uint16) (net.Conn, *Handshake, error) {
	request := getHandshakePayload(infoHash, peerId)
	peerAddress := net.JoinHostPort(peerIp.String(), strconv.Itoa(int(peerPort)))

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

// getHandshakePayload constructs the handshake payload.
func getHandshakePayload(infoHash, peerId [20]byte) []byte {
	handshake := make([]byte, 68)

	handshake[0] = 19
	copy(handshake[1:], "BitTorrent protocol")
	copy(handshake[20:], make([]byte, 8))
	copy(handshake[28:], infoHash[:])
	copy(handshake[48:], peerId[:])

	return handshake
}

// decodeHandshakeResponse decodes the handshake response from the peer.
func (h *Handshake) decodeHandshakeResponse(buf []byte) {
	h.Pstrlen = buf[0]
	h.Pstr = string(buf[1:20])
	copy(h.Reserved[:], buf[20:28])
	copy(h.InfoHash[:], buf[28:48])
	copy(h.PeerId[:], buf[48:68])
}
