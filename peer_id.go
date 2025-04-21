package torrent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

/*
GeneratePeerID generates a random peer ID for the torrent client.
The peer ID is a 20-character string that starts with prefix "-CLOVER-".
The next 12 bytes are random hexadecimal characters.
*/
func GeneratePeerID() ([20]byte, error) {
	const PREFIX = "-CLOVER-"

	randomBytes := make([]byte, 6)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return [20]byte{}, fmt.Errorf("failed to generate random bytes: %v", err)
	}

	randomHex := hex.EncodeToString(randomBytes)

	peerID := PREFIX + randomHex
	var peerIDArray [20]byte

	if len(peerID) != 20 {
		copy(peerIDArray[:], peerID[:20])
		return peerIDArray, nil
	}

	copy(peerIDArray[:], peerID)
	return peerIDArray, nil
}
