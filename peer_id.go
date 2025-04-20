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
func GeneratePeerID() (string, error) {
	const PREFIX = "-CLOVER-"

	randomBytes := make([]byte, 6)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %v", err)
	}

	randomHex := hex.EncodeToString(randomBytes)

	peerID := PREFIX + randomHex
	if len(peerID) != 20 {
		return peerID[:20], nil
	}

	return peerID, nil
}
