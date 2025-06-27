package tracker_test

import (
	"log"
	"math/rand/v2"
	"testing"

	"github.com/JoelVCrasta/parsing"
	"github.com/JoelVCrasta/peer"
	"github.com/JoelVCrasta/tracker"
)

func TestTracker(t *testing.T) {
	var data parsing.Torrent

	err := data.Torrent("../assets/arch.torrent")
	if err != nil {
		t.Fatalf("Failed to initialize torrent (%v)", err)
	}

	log.Println(data.AnnounceList)

	// Connect to a tracker
	var conn *tracker.Connection
	for _, announce := range data.AnnounceList {
		tempConn, err := tracker.ConnectTracker(announce) // opentor.net:6969
		if err != nil {
			t.Errorf("Failed to create UDP tracker connection: %v", err)
			continue
		}

		conn = tempConn
		t.Log("Connection established with tracker")
		break
	}

	// Generate a random transaction ID
	peerId, err := peer.GeneratePeerID()
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("Peer ID:", string(peerId[:]))

	// Create a new announce request
	testRequest := tracker.AnnounceRequest{
		Key:        rand.Uint32(),
		InfoHash:   data.InfoHash,
		IpAddr:     0,
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       1000,
		Event:      0,
		Numwant:    50,
	}

	res, err := conn.AnnounceTracker(testRequest, peerId)
	if err != nil {
		t.Fatalf("Failed to send announce request: %v", err)
	}

	t.Logf("Action: %d, Transaction ID: %d, Interval: %d, Leechers: %d, Seeders: %d\n", res.Action, res.TransactionId, res.Interval, res.Leechers, res.Seeders)
	t.Log("Peers:", res.Peers)

	// Close all the connections
	conn.Close()
}
