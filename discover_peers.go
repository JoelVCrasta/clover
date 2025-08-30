package torrent

import (
	"context"
	"fmt"

	"github.com/JoelVCrasta/dht"
	"github.com/JoelVCrasta/peer"
	"github.com/JoelVCrasta/tracker"
)

func StartPeerDiscovery(announceList []string, infoHash [20]byte, peerId [20]byte) (<-chan peer.Peer, context.CancelFunc, error) {
	tm := tracker.NewTrackerManager(announceList, infoHash, peerId)
	d, err := dht.NewDHT(infoHash)
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}

	tC, err := tm.StartTracker()
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}
	dhtC, err := d.StartDHT()
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}

	pC, mergeStreamCancel := peer.MergeStream(tC, dhtC)

	cancel := func() {
		tm.StopTracker()
		d.StopDHT()
		mergeStreamCancel()
	}

	return pC, cancel, nil
}
