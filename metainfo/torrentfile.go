package metainfo

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

type Torrent struct {
	Announce     string
	AnnounceList []string
	CreatedBy    string
	CreationDate int
	Comment      string
	Info         Info
	InfoHash     [20]byte
	PiecesHash   [][20]byte
	IsMultiFile  bool
}

type Info struct {
	Name        string
	Length      int
	PieceLength int
	Pieces      []byte
	Files       []File
}

type File struct {
	Length int
	Path   string
	Offset int
}

// Torrent loads and parses a torrent file from the specified path, populating the Torrent struct fields.
func (t *Torrent) Torrent(filePath string) error {
	bencodeByteStream, err := t.loadTorrentFile(filePath)
	if err != nil {
		return fmt.Errorf("[torrent] %v", err)
	}

	err = t.populateTorrent(bencodeByteStream)
	if err != nil {
		return fmt.Errorf("[torrent] %v", err)
	}

	return nil
}

// loadTorrentFile reads the torrent file from the specified path and returns its content as a byte slice.
func (t Torrent) loadTorrentFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	becodedData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return becodedData, nil
}

// populateTorrent decodes the bencoded byte stream and populates the Torrent struct fields.
func (t *Torrent) populateTorrent(bencodeByteStream []byte) error {
	decoded, err := BencodeUnmarshall(bencodeByteStream)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	torrent, ok := decoded.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid torrent file format")
	}

	// Optional: announce
	if announce, ok := torrent["announce"].([]byte); ok {
		t.Announce = string(announce)
	}

	// Required: info
	info, ok := torrent["info"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing required field: info")
	}

	// Required: name
	if name, ok := info["name"].([]byte); ok {
		t.Info.Name = string(name)
	} else {
		return fmt.Errorf("missing required field: info.name")
	}

	// Required: piece length
	if pieceLength, ok := info["piece length"].(int); ok {
		t.Info.PieceLength = pieceLength
	} else {
		return fmt.Errorf("missing required field: info.piecelength")
	}

	// Required: pieces
	if pieces, ok := info["pieces"].([]byte); ok {
		t.Info.Pieces = pieces
	} else {
		return fmt.Errorf("missing required field: info.pieces")
	}

	// Handle single-file OR multi-file
	if length, ok := info["length"].(int); ok {
		// Single-file mode
		t.Info.Length = length
	} else if files, ok := info["files"].([]any); ok {
		// Multi-file mode
		offset := 0

		for _, file := range files {
			var f File

			if fileInfo, ok := file.(map[string]any); ok {
				// Required: length
				if length, ok := fileInfo["length"].(int); ok {
					f.Length = length
				} else {
					return fmt.Errorf("missing required field: filelength")
				}

				var pathParts []string

				// Required: paths
				if path, ok := fileInfo["path"].([]any); ok {
					for _, p := range path {
						if str, ok := p.([]byte); ok {
							pathParts = append(pathParts, string(str))
						} else {
							return fmt.Errorf("invalid file path entry")
						}
					}
				} else {
					return fmt.Errorf("missing required field: filepath")
				}

				f.Path = filepath.Join(pathParts...)
				f.Offset = offset
				offset += f.Length
				t.Info.Files = append(t.Info.Files, f)
			}
		}

		t.Info.Length = t.computeTotalLength()
		t.IsMultiFile = true
	} else {
		return fmt.Errorf("missing required field: either info.length or info.files")
	}

	// Required: announce-list
	if announceList, ok := torrent["announce-list"].([]any); ok {
		for _, tier := range announceList {
			if tracker, ok := tier.([]any); ok {
				for _, item := range tracker {
					if str, ok := item.([]byte); ok {
						parsed, err := url.Parse(string(str))
						if err != nil {
							return fmt.Errorf("invalid announce-list entry: %v", err)
						}

						if parsed.Scheme == "udp" {
							t.AnnounceList = append(t.AnnounceList, parsed.Host)
						}
					} else {
						return fmt.Errorf("invalid announce-list entry")
					}
				}
			} else {
				return fmt.Errorf("invalid announce-list format")
			}
		}
	}

	// Optional fields
	if createdBy, ok := torrent["created by"].([]byte); ok {
		t.CreatedBy = string(createdBy)
	}
	if creationDate, ok := torrent["creation date"].(int); ok {
		t.CreationDate = creationDate
	}
	if comment, ok := torrent["comment"].([]byte); ok {
		t.Comment = string(comment)
	}

	// Calculate the info hash
	infoEncoded, err := BencodeMarshall(torrent["info"])
	if err != nil {
		return err
	}
	infoHash := hashInfoDirectory(infoEncoded)
	t.InfoHash = infoHash

	// Split pieces into 20 byte SHA1 hashes
	piecesHash, err := splitPieces(t.Info.Pieces)
	if err != nil {
		return err
	}
	t.PiecesHash = piecesHash

	return nil
}

func (t Torrent) computeTotalLength() int {
	totalLength := 0

	for _, file := range t.Info.Files {
		totalLength += file.Length
	}
	return totalLength
}
