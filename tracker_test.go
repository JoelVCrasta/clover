package torrent_test

import (
	"log"
	"math/rand/v2"
	"testing"

	torrent "github.com/JoelVCrasta"
)

func TestTracker(t *testing.T) {
	conn, err := torrent.NewUDPTrackerConnection("opentor.net:6969")
	if err != nil {
		t.Fatalf("Failed to create UDP tracker connection: %v", err)
	}

	// Generate a random transaction ID
	peerId, err := torrent.GeneratePeerID()
	if err != nil {
		t.Error(err)
		return
	}

	// Create a new announce request
	testRequest := torrent.AnnounceRequest{
		Key:        rand.Uint32(),
		Action:     1,
		InfoHash:   [20]byte{24, 168, 151, 85, 229, 22, 22, 201, 232, 233, 4, 255, 54, 14, 54, 77, 235, 75, 9, 34},
		IpAddr:     0,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       10000,
		Event:      0,
		Numwant:    50,
	}

	response, err := conn.TrackerAnnounce(testRequest, peerId)
	if err != nil {
		t.Fatalf("Failed to send announce request: %v", err)
	}

	log.Println("Announce Response:", response)

	log.Println(response.Peers)

	peerRes, err := torrent.NewHandshake(testRequest.InfoHash, peerId, response.Peers[0].IpAddr, response.Peers[0].Port)
	if err != nil {
		t.Fatalf("Failed to create handshake: %v", err)
	}
	log.Println("Handshake Response:", peerRes)
}
