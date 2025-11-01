package download

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JoelVCrasta/clover/client"
	"github.com/JoelVCrasta/clover/config"
	"github.com/JoelVCrasta/clover/message"
	"github.com/JoelVCrasta/clover/metainfo"
)

const MAX_BLOCK_SIZE = 16384 // 16 KiB
const MAX_BACKLOG = 10

type DownloadManager struct {
	client           *client.Client
	torrent          metainfo.Torrent
	workQueue        chan int
	downloadedPieces []bool

	stats *Stats

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

type workPiece struct {
	index           int
	buf             []byte
	hash            [20]byte
	length          int
	downloadedBytes int
	requestedBytes  int
	backlog         int
}

type completedPiece struct {
	index  int
	length int
	buf    []byte
}

type Stats struct {
	Done      int
	Total     int
	PeerCount int32
}

func NewDownloadManager(torrent metainfo.Torrent, client *client.Client) *DownloadManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &DownloadManager{
		client:           client,
		torrent:          torrent,
		workQueue:        make(chan int, len(torrent.PiecesHash)),
		downloadedPieces: make([]bool, len(torrent.PiecesHash)),
		mu:               sync.Mutex{},
		ctx:              ctx,
		cancel:           cancel,
		stats: &Stats{
			Total:     len(torrent.PiecesHash),
			Done:      0,
			PeerCount: 0,
		},
	}
}

/*
StartDownload begins the download process by distributing work to active peers.
The completed pieces are written to disk using the PieceWriter.
*/
func (dm *DownloadManager) StartDownload(apC <-chan *client.ActivePeer) {
	for i := range dm.torrent.PiecesHash {
		dm.workQueue <- i
	}

	completedPieces := make(chan *completedPiece, 50)
	var wg sync.WaitGroup

	// Start a goroutine for each active peer to download pieces
	go func() {
		for ap := range apC {
			wg.Add(1)
			go func(ap *client.ActivePeer) {
				defer wg.Done()
				atomic.AddInt32(&dm.stats.PeerCount, 1)
				dm.peerDownload(ap, completedPieces)
			}(ap)
		}

		wg.Wait()
	}()

	pw, err := NewPieceWriter(dm.torrent)
	if err != nil {
		log.Fatalf("[download] failed to create piece writer: %v", err)
		return
	}
	defer pw.CloseWriter()

	// Listen for completed pieces and write them to disk
	for cp := range completedPieces {
		err := pw.WritePiece(cp)
		if err != nil {
			// log.Printf("[download] failed to write piece %d: %v", cp.index, err)
			// Re-queue the piece for download
			dm.mu.Lock()
			dm.downloadedPieces[cp.index] = false
			dm.mu.Unlock()
			dm.workQueue <- cp.index
			continue
		}

		// log.Printf("[download] completed piece %d (%d/%d) (Peers: %d)", cp.index, dm.stats.Done+1, dm.stats.Total, atomic.LoadInt32(&dm.stats.PeerCount))
		dm.stats.Done++
		if dm.stats.Done == dm.stats.Total {
			// All pieces done: stop giving out more work and cancel context
			close(dm.workQueue)
			close(completedPieces)
			dm.client.StopClient()
			dm.cancel()
		}
	}
}

// peerDownload handles downloading pieces from a single active peer.
func (dm *DownloadManager) peerDownload(ap *client.ActivePeer, cp chan *completedPiece) {
	defer func() {
		atomic.AddInt32(&dm.stats.PeerCount, -1)
		ap.Disconnect()
	}()

	_ = ap.SendInterested()

	for work := range dm.workQueue {
		if ap.Conn == nil {
			return
		}

		// Check if the piece is already downloaded
		if dm.downloadedPieces[work] {
			continue
		}

		// Check if the peer has the piece
		if !ap.Bitfield.Has(work) {
			dm.workQueue <- work
			continue
		}

		// log.Printf("[download] downloading piece %d from peer %s:%d", work, ap.Peer.IpAddr, ap.Peer.Port)

		length := dm.calculatePieceLength(work)
		wp := &workPiece{
			index:           work,
			buf:             make([]byte, length),
			hash:            dm.torrent.PiecesHash[work],
			length:          length,
			downloadedBytes: 0,
			requestedBytes:  0,
			backlog:         0,
		}

		err := wp.downloadPiece(ap)
		if err != nil {
			if err == io.EOF {
				// log.Printf("[download] peer %s:%d disconnected", ap.Peer.IpAddr, ap.Peer.Port)
				return
			}

			// log.Printf("[download] error downloading piece %d from peer %s:%d: %v", work, ap.Peer.IpAddr, ap.Peer.Port, err)
			dm.workQueue <- work
			ap.FailedCount++

			// If the peer has failed too many times, disconnect
			if ap.FailedCount >= config.Config.MaxFailedRetries {
				// log.Printf("[download] Peer %s:%d has failed too many times, disconnecting", ap.Peer.IpAddr, ap.Peer.Port)
				return
			}
			continue
		}

		dm.mu.Lock()
		dm.downloadedPieces[work] = true
		dm.mu.Unlock()

		ap.SendHave(wp.index)
		cp <- &completedPiece{
			index:  wp.index,
			length: wp.length,
			buf:    wp.buf,
		}
	}

}

// downloadPiece manages the download of a single piece from the peer.
func (wp *workPiece) downloadPiece(ap *client.ActivePeer) error {
	for wp.downloadedBytes < wp.length {
		for !ap.Choked && wp.backlog < MAX_BACKLOG && wp.requestedBytes < wp.length {
			blockSize := MAX_BLOCK_SIZE
			remaining := wp.length - wp.requestedBytes
			if remaining < blockSize {
				blockSize = remaining
			}
			if err := ap.SendRequest(wp.index, wp.requestedBytes, blockSize); err != nil {
				return err
			}
			wp.backlog++
			wp.requestedBytes += blockSize
		}

		err := wp.read(ap)
		if err != nil {
			return err
		}
	}

	if !wp.verify() {
		return fmt.Errorf("piece %d failed verification", wp.index)
	}
	return nil
}

/*
calculatePieceLength calculates the length of a piece based on its index.
It returns the specifies piece length, if its the last piece, it returns the remaining length.
*/
func (dm *DownloadManager) calculatePieceLength(index int) int {
	if index < len(dm.torrent.PiecesHash)-1 {
		return dm.torrent.Info.PieceLength
	}

	lastPieceLength := dm.torrent.Info.Length % dm.torrent.Info.PieceLength
	if lastPieceLength == 0 {
		return dm.torrent.Info.PieceLength
	}
	return lastPieceLength
}

// verify checks if the piece's sha1 hash matches the expected hash.
func (wp *workPiece) verify() bool {
	pieceHash := sha1.Sum(wp.buf)
	return bytes.Equal(pieceHash[:], wp.hash[:])
}

// read reads a message from the peer and handles it accordingly.
func (wp *workPiece) read(ap *client.ActivePeer) error {
	ap.Conn.SetDeadline(time.Now().Add(config.Config.PieceMessageTimeout))
	defer ap.Conn.SetDeadline(time.Time{})

	msg, err := message.ReadPieceMessage(ap.Conn)
	if err != nil {
		return err
	}

	if msg == nil {
		return nil // Keep-alive message
	}

	switch msg.MessageId {
	case message.ChokeId:
		// log.Printf("peer %s:%d choked us", ap.Peer.IpAddr, ap.Peer.Port)
		ap.Choked = true

	case message.UnchokeId:
		// log.Printf("peer %s:%d unchoked us", ap.Peer.IpAddr, ap.Peer.Port)
		ap.Choked = false

	case message.HaveId:
		index, err := msg.DecodeHave()
		if err != nil {
			return err
		}
		ap.Bitfield.Set(index)

	case message.BitfieldId:
		bf, err := msg.DecodeBitfield()
		if err != nil {
			return err
		}
		ap.Bitfield = bf

	case message.PieceId:
		offset, block, err := msg.DecodePiece(wp.index, wp.length)
		if err != nil {
			return err
		}
		copy(wp.buf[offset:], block)
		wp.downloadedBytes += len(block)
		if wp.backlog > 0 {
			wp.backlog--
		}
		// log.Printf("[block] piece %d: downloaded %d/%d bytes", wp.index, wp.downloadedBytes, wp.length)

	case message.PortId:
		return nil

	default:
		// log.Printf("unknown message id %d from peer %s:%d", msg.MessageId, ap.Peer.IpAddr, ap.Peer.Port)
	}

	return nil
}

func (dm *DownloadManager) Stats() *Stats {
	return &Stats{
		Done:      dm.stats.Done,
		Total:     dm.stats.Total,
		PeerCount: atomic.LoadInt32(&dm.stats.PeerCount),
	}
}
