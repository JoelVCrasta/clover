package download

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/JoelVCrasta/clover/config"
	"github.com/JoelVCrasta/clover/metainfo"
)

type PieceWriter struct {
	torrent metainfo.Torrent
	files   map[string]*os.File

	ctx    context.Context
	cancel context.CancelFunc
}

func NewPieceWriter(torrent metainfo.Torrent) (*PieceWriter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pw := &PieceWriter{
		torrent: torrent,
		files:   make(map[string]*os.File),
		ctx:     ctx,
		cancel:  cancel,
	}

	root := filepath.Join(config.Config.DownloadDirectory, torrent.Info.Name)

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

func (pw *PieceWriter) CloseWriter() {
	for _, file := range pw.files {
		_ = file.Close()
	}
	pw.cancel()

	log.Println("[download] piece writer closed")
}
