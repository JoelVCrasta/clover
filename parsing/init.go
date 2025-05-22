package parsing

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
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

	// InfoHash is the SHA-1 hash of the info dictionary
	InfoHash [20]byte

	// PiecesHash is the SHA-1 hash of each piece in the torrent
	PiecesHash [][20]byte

	// Check if the torrent is a multi-file torrent
	IsMultiFile bool
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
	MD5sum []byte
}

/*
Init initializes the Torrent struct by loading a torrent file from the specified path.
It returns an error if any required fields are missing or if the decoding fails.
*/
func (t *Torrent) Init(filePath string) error {
	bencodeByteStream, err := t.loadTorrentFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load torrent file: %v", err)
	}

	err = t.populateTorrent(bencodeByteStream)
	if err != nil {
		return fmt.Errorf("failed to populate torrent: %v", err)
	}

	return nil
}

/*
LoadTorrentFile loads a torrent file from the specified path.
It returns the bencoded byte stream and an error if any occurs.
*/
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

/*
Init initializes the Torrent struct by decoding the bencoded byte stream.
It takes a byte slice as input and populates the Torrent struct fields.
It returns an error if any required fields are missing or if the decoding fails.
*/
func (t *Torrent) populateTorrent(bencodeByteStream []byte) error {
	decoded, err := t.Decode(bencodeByteStream)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	torrent, ok := decoded.(map[string]any)
	if !ok {
		return fmt.Errorf("failed to assert decoded data to map[string]any")
	}

	// Required: announce
	if announce, ok := torrent["announce"].([]byte); ok {
		t.Announce = string(announce)
	} else {
		return fmt.Errorf("missing required field: announce")
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
		return fmt.Errorf("missing required field: info.piece length")
	}

	// Required: pieces
	if pieces, ok := info["pieces"].([]byte); ok {
		t.Info.Pieces = pieces
	} else {
		return fmt.Errorf("missing required field: info.pieces")
	}

	// Optional: private
	if private, ok := info["private"].(int); ok {
		t.Info.Private = private
	}

	// Handle single-file OR multi-file
	if length, ok := info["length"].(int); ok {
		// Single-file mode
		t.Info.Length = length
	} else if files, ok := info["files"].([]any); ok {
		// Multi-file mode
		for _, file := range files {
			var f File
			if fileInfo, ok := file.(map[string]any); ok {
				// Required: file length
				if length, ok := fileInfo["length"].(int); ok {
					f.Length = length
				} else {
					return fmt.Errorf("missing required field: file length")
				}

				// Required: file path
				if path, ok := fileInfo["path"].([]any); ok {
					for _, p := range path {
						if str, ok := p.([]byte); ok {
							f.Path = append(f.Path, string(str))
						} else {
							return fmt.Errorf("invalid file path entry")
						}
					}
				} else {
					return fmt.Errorf("missing required field: file path")
				}

				// Optional: md5sum
				if md5sum, ok := fileInfo["md5sum"].([]byte); ok {
					f.MD5sum = md5sum
				}

				t.Info.Files = append(t.Info.Files, f)
			}
		}

		t.IsMultiFile = true
	} else {
		return fmt.Errorf("missing required field: either info.length or info.files")
	}

	// Optional: announce-list
	if announceList, ok := torrent["announce-list"].([]any); ok {
		for _, tier := range announceList {
			if tierList, ok := tier.([]any); ok {
				var trackers []string

				for _, item := range tierList {
					if str, ok := item.([]byte); ok {
						trackers = append(trackers, string(str))
					}
				}
				if len(trackers) > 0 {
					t.AnnounceList = append(t.AnnounceList, trackers)
				}
			}
		}
	}

	// Optional: created by
	if createdBy, ok := torrent["created by"].([]byte); ok {
		t.CreatedBy = string(createdBy)
	}

	// Optional: creation date
	if creationDate, ok := torrent["creation date"].(int); ok {
		t.CreationDate = creationDate
	}

	// Optional: comment
	if comment, ok := torrent["comment"].([]byte); ok {
		t.Comment = string(comment)
	}

	// Optional: encoding
	if encoding, ok := torrent["encoding"].([]byte); ok {
		t.Encoding = string(encoding)
	}

	// Encode the info dictionary to bencoded byte stream
	infoEncoded, err := t.Encode(torrent["info"])
	if err != nil {
		return err
	}
	infoHash := t.HashInfoDirectory(infoEncoded)
	t.InfoHash = infoHash

	// Split the pieces into an array of 20-byte hashes
	piecesHash, err := t.SplitPieces(t.Info.Pieces)
	if err != nil {
		return err
	}
	t.PiecesHash = piecesHash

	log.Println("Announce: ", t.Announce)
	//log.Println("Announce list: ", t.AnnounceList)
	log.Println("infoHash: ", t.InfoHash)
	fmt.Println("Hex Info Hash: ", hex.EncodeToString(t.InfoHash[:]))

	return nil
}
