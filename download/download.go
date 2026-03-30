package download

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/term"

	"github.com/JoelVCrasta/clover/client"
	"github.com/JoelVCrasta/clover/config"
	"github.com/JoelVCrasta/clover/message"
	"github.com/JoelVCrasta/clover/metainfo"
)

const (
	MAX_BLOCK_SIZE = 16384 // 16 KiB
	MAX_BACKLOG    = 10
)

type DownloadManager struct {
	client           *client.Client
	torrent          metainfo.Torrent
	todoPieces       []int
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
	Done        int
	Total       int
	PeerCount   int32
	TimeElapsed time.Duration
}

func NewDownloadManager(ctx context.Context, torrent metainfo.Torrent, client *client.Client) *DownloadManager {
	ctx, cancel := context.WithCancel(ctx)

	todoPieces := make([]int, len(torrent.PiecesHash))
	for i := range torrent.PiecesHash {
		todoPieces[i] = i
	}

	return &DownloadManager{
		client:           client,
		torrent:          torrent,
		todoPieces:       todoPieces,
		downloadedPieces: make([]bool, len(torrent.PiecesHash)),
		mu:               sync.Mutex{},
		ctx:              ctx,
		cancel:           cancel,
		stats: &Stats{
			Total:       len(torrent.PiecesHash),
			Done:        0,
			PeerCount:   0,
			TimeElapsed: 0,
		},
	}
}

/*
StartDownload begins the download process by distributing work to active peers.
The completed pieces are written to disk using the PieceWriter.
*/
func (dm *DownloadManager) StartDownload(apC <-chan *client.ActivePeer) {
	completedPieces := make(chan *completedPiece, 50)
	var wg sync.WaitGroup

	// start a goroutine for each active peer to download pieces
	go func() {
		for {
			select {
			case <-dm.ctx.Done():
				return

			case ap, ok := <-apC:
				if !ok {
					return
				}
				wg.Add(1)
				go func(ap *client.ActivePeer) {
					defer wg.Done()
					atomic.AddInt32(&dm.stats.PeerCount, 1)
					dm.peerDownload(ap, completedPieces)
				}(ap)
			}
		}
	}()

	// render stats in some interval
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dm.stats.TimeElapsed += 1 * time.Second
				dm.renderProgress()
			case <-dm.ctx.Done():
				return
			}
		}
	}()

	completed := false
	pw, err := NewPieceWriter(dm.torrent)
	if err != nil {
		log.Fatalf("[download] failed to create piece writer: %v", err)
		return
	}
	defer pw.CloseWriter()

	// listen for completed pieces and write them to disk
loop:
	for {
		select {
		case <-dm.ctx.Done():
			dm.client.StopClient()
			if !completed {
				dm.DownloadStatus("Stopped")
			}
			break loop

		case cp, ok := <-completedPieces:
			if !ok {
				break loop
			}

			dm.mu.Lock()
			if dm.downloadedPieces[cp.index] {
				dm.mu.Unlock()
				continue
			}
			dm.downloadedPieces[cp.index] = true
			dm.mu.Unlock()

			err := pw.WritePiece(cp)
			if err != nil {
				dm.mu.Lock()
				dm.downloadedPieces[cp.index] = false
				dm.todoPieces = append(dm.todoPieces, cp.index)
				dm.mu.Unlock()
				continue
			}

			dm.mu.Lock()
			dm.stats.Done++
			if dm.stats.Done == dm.stats.Total {
				completed = true
				dm.mu.Unlock()
				dm.cancel()
			} else {
				dm.mu.Unlock()
			}
		}
	}

	wg.Wait()

	close(completedPieces)

	if completed {
		dm.renderProgress()
		dm.DownloadStatus("Completed")
	}
}

func (dm *DownloadManager) pickPiece(ap *client.ActivePeer) (int, bool) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for i, index := range dm.todoPieces {
		if ap.Bitfield.Has(index) {
			dm.todoPieces = append(dm.todoPieces[:i], dm.todoPieces[i+1:]...)
			return index, true
		}
	}

	// end game mode
	remaining := dm.stats.Total - dm.stats.Done
	if remaining > 0 && remaining <= 5 {
		for i, done := range dm.downloadedPieces {
			if !done && ap.Bitfield.Has(i) {
				return i, true
			}
		}
	}

	return 0, false
}

func (dm *DownloadManager) returnPiece(index int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	// only return if not already downloaded
	if !dm.downloadedPieces[index] {
		// avoid duplicates in todoPieces
		for _, p := range dm.todoPieces {
			if p == index {
				return
			}
		}
		dm.todoPieces = append(dm.todoPieces, index)
	}
}

// peerDownload handles downloading pieces from a single active peer.
func (dm *DownloadManager) peerDownload(ap *client.ActivePeer, cp chan *completedPiece) {
	defer func() {
		atomic.AddInt32(&dm.stats.PeerCount, -1)
		ap.Disconnect()
	}()

	_ = ap.SendInterested()

	for {
		select {
		case <-dm.ctx.Done():
			return
		default:
			if ap.Conn == nil {
				return
			}

			work, ok := dm.pickPiece(ap)
			if !ok {
				err := dm.handleMessage(ap, nil)
				if err != nil {
					if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
						continue // Just a timeout, keep connection alive
					}
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Check if piece was finished during the end game
			dm.mu.Lock()
			if dm.downloadedPieces[work] {
				dm.mu.Unlock()
				continue
			}
			dm.mu.Unlock()

			if ap.IsChoked() {
				dm.returnPiece(work)
				err := dm.handleMessage(ap, nil)
				if err != nil {
					if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
						continue
					}
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

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

			err := wp.downloadPiece(ap, dm)
			if err != nil {
				dm.returnPiece(work)
				if err == io.EOF {
					return
				}

				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				}

				if err.Error() != "peer choked" {
					ap.FailedCount++
				}

				// If the peer has failed too many times, disconnect
				if ap.FailedCount >= config.Config.MaxFailedRetries {
					return
				}
				continue
			}

			// Verify one last time before sending to writer
			dm.mu.Lock()
			if dm.downloadedPieces[work] {
				dm.mu.Unlock()
				continue
			}
			dm.mu.Unlock()

			ap.SendHave(wp.index)
			cp <- &completedPiece{
				index:  wp.index,
				length: wp.length,
				buf:    wp.buf,
			}
		}
	}
}

// downloadPiece manages the download of a single piece from the peer.
func (wp *workPiece) downloadPiece(ap *client.ActivePeer, dm *DownloadManager) error {
	for wp.downloadedBytes < wp.length {
		// check if some other peer finished this piece during the end game
		dm.mu.Lock()
		if dm.downloadedPieces[wp.index] {
			dm.mu.Unlock()
			return nil
		}
		dm.mu.Unlock()

		for !ap.IsChoked() && wp.backlog < MAX_BACKLOG && wp.requestedBytes < wp.length {
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

		if ap.IsChoked() && wp.backlog == 0 {
			return fmt.Errorf("peer choked")
		}

		err := dm.handleMessage(ap, wp)
		if err != nil {
			return err
		}
	}

	if !wp.verify() {
		// log.Printf("[download] piece %d failed verification", wp.index)
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

// handleMessage reads a message from the peer and handles it accordingly.
func (dm *DownloadManager) handleMessage(ap *client.ActivePeer, wp *workPiece) error {
	ap.Conn.SetDeadline(time.Now().Add(config.Config.PieceMessageTimeout))
	defer ap.Conn.SetDeadline(time.Time{})

	msg, err := message.ReadPieceMessage(ap.Conn)
	if err != nil {
		return err
	}

	if msg == nil {
		return nil // keep-alive message
	}

	switch msg.MessageId {
	case message.ChokeId:
		ap.SetChoked(true)

	case message.UnchokeId:
		ap.SetChoked(false)

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
		if wp == nil {
			return nil
		}
		offset, block, err := msg.DecodePiece(wp.index, wp.length)
		if err != nil {
			return err
		}
		copy(wp.buf[offset:], block)
		wp.downloadedBytes += len(block)
		if wp.backlog > 0 {
			wp.backlog--
		}

	case message.PortId:
		return nil
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

func (dm *DownloadManager) DownloadStatus(status string) string {
	return fmt.Sprintf("Download %s!\n", status)
}

func (dm *DownloadManager) CancelDownload() context.CancelFunc {
	return dm.cancel
}

// renderProgress renders the stats and the downloaded piece matrix
func (dm *DownloadManager) renderProgress() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	fmt.Print("\033[H\033[2J")
	fmt.Printf("Downloading torrent in progress...\n")
	fmt.Printf("Pieces: %d/%d | Peers: %d | Time Elapsed: %s\n\n",
		dm.stats.Done, dm.stats.Total, atomic.LoadInt32(&dm.stats.PeerCount),
		dm.stats.TimeElapsed)

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 64
	}
	
	for i, done := range dm.downloadedPieces {
		if done {
			fmt.Print("\033[97m█\033[0m") // white
		} else {
			fmt.Print("\033[90m█\033[0m") // gray
		}
		if (i+1)%width == 0 {
			fmt.Println()
		}
	}
	fmt.Print("\n\nUse Ctrl+C to stop.")
}