package parsing

import (
	"fmt"
	"io"
	"os"
)

type Torrent struct {
	Announce     string
	AnnounceList [][]string
	CreatedBy    string
	CreationDate int
	Comment      string
	Encoding     string
	Info         Info
}

type Info struct {
	Name        string
	Length      int
	PieceLength int
	Pieces      []byte
	Private     int
	Files       []File
}

type File struct {
	Length int
	Path   []string
}

func (t *Torrent) Load(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	becodedData, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	_ = becodedData
	return nil
}
