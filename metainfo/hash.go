package metainfo

import (
	"crypto/sha1"
	"fmt"
)

// The InfoHash function computes the SHA-1 hash of the given info byte slice.
func (t Torrent) HashInfoDirectory(info []byte) [20]byte {
	return sha1.Sum(info)
}

// splitPieces splits the pieces byte slice into an array of 20-byte hashes.
func (t Torrent) SplitPieces(pieces []byte) ([][20]byte, error) {
	if len(pieces)%20 != 0 {
		return nil, fmt.Errorf("pieces length is not a multiple of 20")
	}

	var hashes [][20]byte
	for i := 0; i < len(pieces); i += 20 {
		var piece [20]byte
		copy(piece[:], pieces[i:i+20])
		hashes = append(hashes, piece)
	}

	return hashes, nil
}
