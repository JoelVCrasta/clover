package main

import (
	"log"
	"time"

	"github.com/JoelVCrasta/clover/parsing"
)

func main() {
	t := parsing.Torrent{}

	bencodeByteStream, err := t.LoadTorrentFile("assets/sample.torrent")
	if err != nil {
		log.Fatalln(err)
		return
	}

	err = t.Init(bencodeByteStream)
	if err != nil {
		log.Fatalln(err)
		return
	}

	log.Println("Torrent initialized successfully")
	log.Println("Announce:", t.Announce)
	log.Println("Announce List:", t.AnnounceList)
	log.Println("Created By:", t.CreatedBy)
	log.Println("Creation Date:", time.Unix(int64(t.CreationDate), 0))
	log.Println("Comment:", t.Comment)
	log.Println("Encoding:", t.Encoding)
	log.Println("Info Name:", t.Info.Name)
	log.Println("Info Length:", t.Info.Length)
	log.Println("Info Piece Length:", t.Info.PieceLength)
	log.Println("Info Pieces:", t.Info.Pieces)
	log.Println("Info Private:", t.Info.Private)
	log.Println("Info Files:", t.Info.Files)

}
