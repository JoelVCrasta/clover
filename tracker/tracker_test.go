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

	err := data.Torrent("../assets/rdr2.torrent")
	if err != nil {
		t.Fatalf("Failed to initialize torrent (%v)", err)
	}

	log.Println(data.AnnounceList)

	// Connect to a trackers
	trackerManager := tracker.NewTrackerManager()
	err = trackerManager.ConnectTrackerAll(data.AnnounceList)
	if err != nil {
		t.Fatalf("Failed to connect to trackers: %v", err)
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

	peers := trackerManager.AnnounceTrackerAll(testRequest, peerId)
	t.Log("Peers:", peers)

	// Close all the connections
	trackerManager.Close()

}
