package tracker_test

import (
	"log"
	"testing"
	"time"

	"github.com/JoelVCrasta/parsing"
	"github.com/JoelVCrasta/peer"
	"github.com/JoelVCrasta/tracker"
)

func TestTracker(t *testing.T) {
	var data parsing.Torrent

	err := data.Torrent("../assets/superman.torrent")
	if err != nil {
		t.Fatalf("Failed to initialize torrent (%v)", err)
	}
	log.Println(data.AnnounceList)

	// Generate a random transaction ID
	peerId, err := peer.GeneratePeerID()
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("Peer ID:", string(peerId[:]))

	// Connect to a trackers
	trackerManager := tracker.NewTrackerManager(
		data.AnnounceList,
		data.InfoHash,
		peerId,
	)

	trackerPeerChan, err := trackerManager.Start()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	timeout := time.After(100 * time.Second)
	for {
		select {
		case peer, ok := <-trackerPeerChan:
			if !ok {
				t.Log("[tracker] peer channel closed")
				return
			}
			t.Logf("Got peer from tracker: %s:%d", peer.IpAddr, peer.Port)

		case <-timeout:
			t.Log("Timeout reached, stopping tracker manager")
			trackerManager.Stop()
			return
		}
	}
}
