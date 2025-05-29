package download

import (
	"bytes"
	"crypto/sha1"
	"sync"
	"time"

	"github.com/JoelVCrasta/client"
	"github.com/JoelVCrasta/config"
)

const MAX_BLOCK_SIZE = 16384 // 16 KiB

type DownloadManager struct {
	Client *client.Client

	PieceQueue chan int
	Downloaded map[int]bool
	Requested  map[int]bool

	Mutex sync.Mutex
}

type NewPiece struct {
	index  int
	buf    []byte
	hash   [20]byte
	length int
}

func NewDownloadManager(c *client.Client) *DownloadManager {
	return &DownloadManager{
		Client:     c,
		PieceQueue: make(chan int, len(c.Torrent.PiecesHash)),
		Downloaded: make(map[int]bool),
		Requested:  make(map[int]bool),
	}
}

func (np *NewPiece) downloadPiece(peer *client.ActivePeer) error {
	var (
		downloadedBytes int = 0
		offset         	int = 0
	)

	_ = peer.Conn.SetDeadline(time.Now().Add(config.Config.PieceMessageTimeout))
	defer peer.Conn.SetDeadline(time.Time{})

	for downloadedBytes < np.length {
		remainingBytes := np.length - downloadedBytes

		if !peer.Choked {
			requestSize := MAX_BLOCK_SIZE
			if remainingBytes < MAX_BLOCK_SIZE {
				requestSize = remainingBytes
			}

			err := peer.SendRequest(np.index, offset, requestSize)
			if err != nil {
				return err
			}

			
		}

	}

	return nil
}

/*
calculatePieceLength calculates the length of a piece based on its index.
It returns the specifies piece length, if its the last piece, it returns the remaining length.
*/
func (dm *DownloadManager) calculatePieceLength(index int) int {
	if index < len(dm.Client.Torrent.PiecesHash)-1 {
		return dm.Client.Torrent.Info.PieceLength
	}

	lastPieceLength := dm.Client.Torrent.Info.Length % dm.Client.Torrent.Info.PieceLength
	if lastPieceLength == 0 {
		return dm.Client.Torrent.Info.PieceLength
	}
	return lastPieceLength
}

// verify checks if the piece's sha1 hash matches the expected hash.
func (f *FullPiece) verify() bool {
	pieceHash := sha1.Sum(f.buf)
	return bytes.Equal(pieceHash[:], f.hash[:])
}
