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

type workPiece struct {
	index  int
	buf    []byte
	hash   [20]byte
	length int

	downloadedBytes int

	peer *client.ActivePeer
}

type completedPiece struct {
	index int
	buf   []byte
}

func NewDownloadManager(c *client.Client) *DownloadManager {
	return &DownloadManager{
		Client:     c,
		PieceQueue: make(chan int, len(c.Torrent.PiecesHash)),
		Downloaded: make(map[int]bool),
		Requested:  make(map[int]bool),
	}
}

func NewWorkPiece(index int, peer *client.ActivePeer, hash [20]byte, length int) *workPiece {
	return &workPiece{
		index:           index,
		buf:             make([]byte, length),
		hash:            hash,
		length:          length,
		peer:            peer,
		downloadedBytes: 0,
	}
}

func (dm *DownloadManager) Download() {
	// Fill the piece queue with piece indices
	for i := range dm.Client.Torrent.PiecesHash {
		dm.PieceQueue <- i
	}

	completedPieces := make(chan *completedPiece)
	var wg sync.WaitGroup

	// Start downloading pieces from peers
	for _, peer := range dm.Client.ActivePeers {
		wg.Add(1)
		go func(p *client.ActivePeer) {
			defer wg.Done()
			dm.peerDownload(p, completedPieces)
		}(peer)
	}

	go func() {
		wg.Wait()
		close(completedPieces)
	}()

	for piece := range completedPieces {
		log.Printf("Completed piece %d (len=%d)", piece.index, len(piece.buf))
	}
}

func (dm *DownloadManager) peerDownload(peer *client.ActivePeer, completedPieces chan *completedPiece) {
	log.Printf("Starting download from peer %s:%d", peer.IpAddr, peer.Port)

	_ = peer.SendInterested()

	// Process pieces from the piece queue
	for work := range dm.PieceQueue {
		if !peer.Bitfield.Has(work) {
			dm.PieceQueue <- work // Requeue if peer does not have the piece
			continue
		}

		if dm.Downloaded[work] {
			continue // Skip already downloaded pieces
		}

		dm.Mutex.Lock()
		if dm.Requested[work] {
			dm.Mutex.Unlock()
			continue // Skip already requested pieces
		}
		dm.Requested[work] = true
		dm.Mutex.Unlock()

		log.Printf("Downloading piece %d from peer %s:%d", work, peer.IpAddr, peer.Port)

		// Calculate the length of the piece and create a WorkPiece for piece download
		length := dm.calculatePieceLength(work)
		piece := NewWorkPiece(work, peer, dm.Client.Torrent.PiecesHash[work], length)

		err := piece.downloadPiece(&dm.Mutex)
		if err != nil {
			log.Print(err.Error())
			dm.PieceQueue <- work // Requeue the piece if download failed
			continue
		}

		if !piece.verify() {
			log.Printf("Piece %d from peer %s:%d failed hash verification", work, peer.IpAddr, peer.Port)
			dm.PieceQueue <- work // Requeue the piece if verification failed
			continue
		}

		log.Printf("Downloaded piece %d from peer %s:%d", work, peer.IpAddr, peer.Port)
		dm.Mutex.Lock()
		dm.Downloaded[work] = true // Mark the piece as downloaded
		dm.Requested[work] = false // Mark the piece as no longer requested
		dm.Mutex.Unlock()

		// Send the completed piece to the channel
		completedPieces <- &completedPiece{
			index: work,
			buf:   piece.buf,
		}
	}
}

/*
downloadPiece downloads a piece from a peer.
It sends requests to the peer until the entire piece is downloaded.
If the peer is choked, it will wait until the peer unchokes before continuing.
*/
func (wp *workPiece) downloadPiece(mu *sync.Mutex) error {
	for wp.downloadedBytes < wp.length {
		if !wp.peer.Choked {
			// Calculate the size of the next request
			remainingBytes := wp.length - wp.downloadedBytes
			requestSize := min(remainingBytes, MAX_BLOCK_SIZE)

			err := wp.peer.SendRequest(wp.index, wp.downloadedBytes, requestSize)
			if err != nil {
				return err
			}
		}

		// Read the piece data from the peer
		err := wp.read(mu)
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
func (wp *workPiece) verify() bool {
	pieceHash := sha1.Sum(wp.buf)
	return bytes.Equal(pieceHash[:], wp.hash[:])
}

/*
read reads messages from the peer connection.
It processes different message types such as Choke, Unchoke, Have, Bitfield, Piece, and Port (for now only leeching)
It updates the peer's state based on the received messages.
*/
func (wp *workPiece) read(mu *sync.Mutex) error {
	_ = wp.peer.Conn.SetDeadline(time.Now().Add(config.Config.PieceMessageTimeout)) // Set a deadline for reading messages

	msg, err := message.ReadMessage(wp.peer.Conn)
	if err != nil {
		// Peer connection closed
		if err == io.EOF {
			wp.peer.Conn.Close()
			return fmt.Errorf("connection closed by peer %s:%d", wp.peer.IpAddr, wp.peer.Port)
		}

		return err
	}

	_ = wp.peer.Conn.SetDeadline(time.Time{}) // Clear the deadline after reading a message

	// KeepAlive message
	if msg == nil {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	switch msg.MessageId {
	case message.ChokeId:
		log.Printf("Peer %s:%d choked us", wp.peer.IpAddr, wp.peer.Port)
		wp.peer.Choked = true

	case message.UnchokeId:
		log.Printf("Peer %s:%d unchoked us", wp.peer.IpAddr, wp.peer.Port)
		wp.peer.Choked = false

	case message.HaveId:
		pieceIndex, err := msg.DecodeHave()
		if err != nil {
			return err
		}
		wp.peer.Bitfield.Set(pieceIndex)

	case message.BitfieldId:
		bitfield, err := msg.DecodeBitfield()
		if err != nil {
			return err
		}
		wp.peer.Bitfield = bitfield

	case message.PieceId:
		offset, block, err := msg.DecodePiece(wp.index, wp.length)
		if err != nil {
			return err
		}

		log.Printf("Received block %d", offset)
		copy(wp.buf[offset:], block)
		wp.downloadedBytes += len(block)

	case message.PortId:
		log.Printf("Peer %s:%d sent Port message with port %x", wp.peer.IpAddr, wp.peer.Port, msg.Payload)

	default:
		return fmt.Errorf("unknown message type %d from peer %s:%d", msg.MessageId, wp.peer.IpAddr, wp.peer.Port)
	}

	return nil
}
