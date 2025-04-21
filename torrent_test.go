package torrent_test

import (
	"testing"

	torrent "github.com/JoelVCrasta"
)

func TestMain(t *testing.T) {
	torrent.Run()

	// peerId, err := torrent.GeneratePeerID()
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	// fmt.Println("Generated Peer ID:", peerId)
}
