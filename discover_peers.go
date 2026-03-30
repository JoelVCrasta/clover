package torrent

import (
	"context"

	"github.com/JoelVCrasta/clover/dht"
	"github.com/JoelVCrasta/clover/peer"
	"github.com/JoelVCrasta/clover/tracker"
)

// StartPeerDiscovery is used start the trackers and dht to seach for peers
// and merge them into a single channel
func StartPeerDiscovery(ctx context.Context, announceList []string, infoHash [20]byte, peerId [20]byte) (<-chan peer.Peer, error) {
	tm := tracker.NewTrackerManager(ctx, announceList, infoHash, peerId)
	d, err := dht.NewDHT(ctx, infoHash)
	if err != nil {
		return nil, err
	}

	tC, err := tm.StartTracker()
	if err != nil {
		return nil, err
	}
	dhtC, err := d.StartDHT()
	if err != nil {
		return nil, err
	}

	pC := peer.MergeStream(ctx, tC, dhtC)

	go func() {
		<-ctx.Done()
		tm.StopTracker()
		d.StopDHT()
	}()

	return pC, nil
}
