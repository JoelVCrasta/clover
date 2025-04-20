package torrent

import (
	"log"

	"github.com/JoelVCrasta/parsing"
)

func Run() {
	var torrent parsing.Torrent

	err := torrent.Init("assets/sample.torrent")
	if err != nil {
		log.Fatalf("Failed to initialize torrent: %v", err)
	}
}
