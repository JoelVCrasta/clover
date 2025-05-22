package torrent_test

import (
	"log"
	"math/rand/v2"
	"testing"

	torrent "github.com/JoelVCrasta"
)

func TestTracker(t *testing.T) {

	conn, err := torrent.NewUDPTrackerConnection("tracker.opentrackr.org:1337") // opentor.net:6969
	if err != nil {
		t.Fatalf("Failed to create UDP tracker connection: %v", err)
	}
	log.Println("Connection established with tracker")

	// Generate a random transaction ID
	peerId, err := torrent.GeneratePeerID()
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("Generated Peer ID:", peerId)
	log.Println("Peer ID string:", string(peerId[:]))

	// Create a new announce request
	testRequest := torrent.AnnounceRequest{
		Key:        rand.Uint32(),
		InfoHash:   [20]byte{53, 253, 194, 188, 211, 197, 66, 97, 84, 53, 232, 149, 165, 210, 43, 91, 143, 236, 112, 25},
		IpAddr:     0,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       1000,
		Event:      0,
		Numwant:    50,
	}

	res, err := conn.TrackerAnnounce(testRequest, peerId)
	if err != nil {
		t.Fatalf("Failed to send announce request: %v", err)
	}

	log.Printf("Action: %d, Transaction ID: %d, Interval: %d, Leechers: %d, Seeders: %d\n", res.Action, res.TransactionId, res.Interval, res.Leechers, res.Seeders)
	log.Println("Peers:", res.Peers)

	conn.Close()

	// Send a handshake request to the first peer
	// peerRes, err := torrent.NewHandshake(testRequest.InfoHash, peerId, uint32(1142975849), uint16(29555))
	// if err != nil {
	// 	t.Fatalf("Failed to create handshake: %v", err)
	// }
	// log.Println("Handshake Response:", peerRes)
}
