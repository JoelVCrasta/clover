package tracker_test

import (
	"testing"
	"time"

	"github.com/JoelVCrasta/clover/metainfo"
	"github.com/JoelVCrasta/clover/peer"
	"github.com/JoelVCrasta/clover/tracker"
)

func TestTracker(t *testing.T) {
	var data metainfo.Torrent

	if err := data.Torrent("../assets/bot.torrent", "."); err != nil {
		t.Fatalf("Failed to initialize torrent (%v)", err)
	}

	peerId, err := peer.GeneratePeerID()
	if err != nil {
		t.Fatal(err)
	}

	trackerManager := tracker.NewTrackerManager(
		data.AnnounceList,
		data.InfoHash,
		peerId,
	)

	trackerPeerChan, err := trackerManager.StartTracker()
	if err != nil {
		t.Fatal(err)
	}

	timeout := time.After(30 * time.Second)
	gotPeers := 0

	for {
		select {
		case p, ok := <-trackerPeerChan:
			if !ok {
				if gotPeers == 0 {
					t.Fatal("Tracker channel closed with no peers received")
				}
				trackerManager.StopTracker()
				return
			}
			gotPeers++
			t.Logf("Got peer from tracker: %s", p.String())
		case <-timeout:
			if gotPeers == 0 {
				t.Fatal("Timeout reached, no peers received from tracker")
			}
			t.Logf("Received %d peers before timeout", gotPeers)
			trackerManager.StopTracker()
			return
		}
	}
}
