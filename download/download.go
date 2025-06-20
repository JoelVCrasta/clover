package download

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/JoelVCrasta/client"
	"github.com/JoelVCrasta/config"
	"github.com/JoelVCrasta/message"
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

	downloadedBytes int

	peer *client.ActivePeer
}

func NewDownloadManager(c *client.Client) *DownloadManager {
	return &DownloadManager{
		Client:     c,
		PieceQueue: make(chan int, len(c.Torrent.PiecesHash)),
		Downloaded: make(map[int]bool),
		Requested:  make(map[int]bool),
	}
}


/*
downloadPiece downloads a piece from a peer.
It sends requests to the peer until the entire piece is downloaded.
If the peer is choked, it will wait until the peer unchokes before continuing.
*/
func (np *NewPiece) downloadPiece(peer *client.ActivePeer, mu *sync.Mutex) error {
	_ = peer.Conn.SetDeadline(time.Now().Add(config.Config.PieceMessageTimeout))
	defer peer.Conn.SetDeadline(time.Time{})

	for np.downloadedBytes < np.length {
		// Sleep if the peer is choked
		if peer.Choked {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Calculate the size of the next request
		remainingBytes := np.length - np.downloadedBytes
		requestSize := min(remainingBytes, MAX_BLOCK_SIZE)

		err := peer.SendRequest(np.index, np.downloadedBytes, requestSize)
		if err != nil {
			return err
		}

		// Read the piece data from the peer
		err = np.read(mu)
		if err != nil {
			return err
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
func (np *NewPiece) verify() bool {
	pieceHash := sha1.Sum(np.buf)
	return bytes.Equal(pieceHash[:], np.hash[:])
}

/*
read reads messages from the peer connection.
It processes different message types such as Choke, Unchoke, Have, Bitfield, Piece, and Port (for now only leeching)
It updates the peer's state based on the received messages.
*/
func (np *NewPiece) read(mu *sync.Mutex) error {
	msg, err := message.ReadMessage(np.peer.Conn)
	if err != nil {
		// Peer connection closed
		if err == io.EOF {
			return fmt.Errorf("connection closed by peer %s:%d", np.peer.IpAddr, np.peer.Port)
		}

		return err
	}

	// KeepAlive message
	if msg == nil {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	switch msg.MessageId {
	case message.ChokeId:
		np.peer.Choked = true

	case message.UnchokeId:
		np.peer.Choked = false

	case message.HaveId:
		pieceIndex, err := msg.DecodeHave()
		if err != nil {
			return err
		}
		np.peer.Bitfield.Set(pieceIndex)

	case message.BitfieldId:
		bitfield, err := msg.DecodeBitfield()
		if err != nil {
			return err
		}
		np.peer.Bitfield = bitfield

	case message.PieceId:
		offset, block, err := msg.DecodePiece(np.index, np.length)
		if err != nil {
			return err
		}
		copy(np.buf[offset:], block)
		np.downloadedBytes += len(block)

	case message.PortId:
		log.Printf("Peer %s:%d sent Port message with port %x", np.peer.IpAddr, np.peer.Port, msg.Payload)

	default:
		return fmt.Errorf("unknown message type %d from peer %s:%d", msg.MessageId, np.peer.IpAddr, np.peer.Port)
	}

	return nil
}
