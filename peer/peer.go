package peer

import (
	"context"
	"net"
	"strconv"
)

type Peer struct {
	IpAddr net.IP
	Port   uint16
}

// MergeStream merges two channels from the 2 sources (tracker and dht) into a single channel.
func MergeStream(tC <-chan Peer, dhtC <-chan Peer, ctx context.Context) <-chan Peer {
	peerChan := make(chan Peer)

	go func() {
		defer close(peerChan)

		for {
			select {
			case peer, ok := <-tC:
				if ok {
					peerChan <- peer
				}
			case peer, ok := <-dhtC:
				if ok {
					peerChan <- peer
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return peerChan
}

func (p Peer) String() string {
	return net.JoinHostPort(p.IpAddr.String(), strconv.Itoa(int(p.Port)))
}
