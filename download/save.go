package download

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/JoelVCrasta/clover/config"
	"github.com/JoelVCrasta/clover/metainfo"
)

var root_ string

type PieceWriter struct {
	torrent metainfo.Torrent
	files   map[string]*os.File
}

func NewPieceWriter(torrent metainfo.Torrent) (*PieceWriter, error) {
	pw := &PieceWriter{
		torrent: torrent,
		files:   make(map[string]*os.File),
	}

	if torrent.OutputPath != "" {
		root_ = torrent.OutputPath
	} else {
		root_ = config.Config.DownloadDirectory
	}

	root := filepath.Join(root_, torrent.Info.Name)

	cleanup := func() {
		pw.CloseWriter()
		_ = os.RemoveAll(root)
	}
	defer func() {
		if cleanup != nil {
			cleanup()
		}
	}()

	if torrent.IsMultiFile {
		err := os.MkdirAll(root, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create root dir: %v", err)
		}

		for _, file := range torrent.Info.Files {
			fullPath := filepath.Join(root, file.Path)
			dir := filepath.Dir(fullPath)

			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return nil, fmt.Errorf("failed to create subdir: %v", err)
			}

			f, err := os.Create(fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create file: %v", err)
			}

			if err := f.Truncate(int64(file.Length)); err != nil {
				f.Close()
				return nil, fmt.Errorf("failed to preallocate file: %v", err)
			}

			pw.files[fullPath] = f
		}
	} else {
		dir := filepath.Dir(root)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create download dir: %v", err)
		}

		file, err := os.Create(root)
		if err != nil {
			return nil, fmt.Errorf("failed to create file: %v", err)
		}

		if err := file.Truncate(int64(torrent.Info.Length)); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to preallocate file: %v", err)
		}

		pw.files[root] = file
	}

	cleanup = nil
	return pw, nil
}

func (pw *PieceWriter) WritePiece(cp *completedPiece) error {
	pieceStart := cp.index * pw.torrent.Info.PieceLength
	pieceLen := cp.length
	pieceEnd := pieceStart + pieceLen
	bufOffset := 0

	if pw.torrent.IsMultiFile {
		for _, file := range pw.torrent.Info.Files {
			fileStart := file.Offset
			fileEnd := fileStart + file.Length

			// If piece does not overlap with file, skip
			if pieceEnd <= fileStart || pieceStart >= fileEnd {
				continue
			}

			// Piece overlaps with file
			writeStart := max(pieceStart, fileStart)
			writeEnd := min(pieceEnd, fileEnd)
			writeLen := writeEnd - writeStart

			filePath := filepath.Join(root_, pw.torrent.Info.Name, file.Path)
			f := pw.files[filePath]
			if f == nil {
				return fmt.Errorf("file not found for path: %s", filePath)
			}

			fileOffset := writeStart - fileStart
			_, err := f.WriteAt(cp.buf[bufOffset:bufOffset+writeLen], int64(fileOffset))
			if err != nil {
				return fmt.Errorf("failed to write to file: %v", err)
			}

			bufOffset += writeLen
			if bufOffset >= pieceLen {
				break
			}
		}
	} else {
		filePath := filepath.Join(root_, pw.torrent.Info.Name)
		f := pw.files[filePath]
		if f == nil {
			return fmt.Errorf("file not found for path: %s", filePath)
		}

		_, err := f.WriteAt(cp.buf, int64(pieceStart))
		if err != nil {
			return fmt.Errorf("failed to write to file: %v", err)
		}
	}

	return nil
}

func (pw *PieceWriter) CloseWriter() {
	for _, file := range pw.files {
		_ = file.Close()
	}
	log.Println("[download] piece writer closed")
}
