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
func MergeStream(ctx context.Context, tC <-chan Peer, dhtC <-chan Peer) <-chan Peer {
	peerChan := make(chan Peer, 1000)

	go func() {
		defer close(peerChan)

		for tC != nil || dhtC != nil {
			select {
			case peer, ok := <-tC:
				if !ok {
					tC = nil
					continue
				}
				select {
				case peerChan <- peer:
				case <-ctx.Done():
					return
				}

			case peer, ok := <-dhtC:
				if !ok {
					dhtC = nil
					continue
				}
				select {
				case peerChan <- peer:
				case <-ctx.Done():
					return
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
