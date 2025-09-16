package metainfo

import (
	"crypto/sha1"
	"fmt"
)

// HashInfoDirectory computes the SHA1 hash of the bencoded info dictionary.
func hashInfoDirectory(info []byte) [20]byte {
	return sha1.Sum(info)
}

// splitPieces splits the pieces byte slice into an array of 20-byte hashes.
func splitPieces(pieces []byte) ([][20]byte, error) {
	if len(pieces)%20 != 0 {
		return nil, fmt.Errorf("invalid pieces length")
	}

	var hashes [][20]byte
	for i := 0; i < len(pieces); i += 20 {
		var piece [20]byte
		copy(piece[:], pieces[i:i+20])
		hashes = append(hashes, piece)
	}

	return hashes, nil
}
