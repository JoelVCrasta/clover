package download

import (
	"fmt"
	"os"
	"path"

	"github.com/JoelVCrasta/clover/config"
	"github.com/JoelVCrasta/clover/metainfo"
)

type FileRegion struct {
	Path   string
	Length int
	Start  int
	End    int
	File   *os.File
}

func createSaveFile(torrent metainfo.Torrent) (*os.File, error) {
	downloadDir := config.Config.DownloadDirectory

	// Multifile torrent
	if torrent.IsMultiFile {
		path := path.Join(downloadDir, torrent.Info.Name)
		err := os.MkdirAll(path, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create download directory: %w", err)
		}

		return nil, nil
	}

	// Single file torrent
	path := path.Join(downloadDir, torrent.Info.Name)
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create save file: %w", err)
	}

	return file, nil
}
