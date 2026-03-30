package dht

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/JoelVCrasta/clover/peer"
	"github.com/anacrolix/dht/v2"
)

type DHT struct {
	server   *dht.Server
	infoHash [20]byte
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewDHT(ctx context.Context, infoHash [20]byte) (*DHT, error) {
	config := dht.NewDefaultServerConfig()

	server, err := dht.NewServer(config)
	if err != nil {
		return nil, fmt.Errorf("[dht] failed to create DHT server: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	return &DHT{
		server:   server,
		infoHash: infoHash,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

/*
Start starts the DHT bootstrapping process and begins announcing the info hash.
It returns a channel that will receive discovered peers.
It will periodically announce the info hash every 5 minutes.
*/
func (d *DHT) StartDHT() (<-chan peer.Peer, error) {
	peerChan := make(chan peer.Peer, 500)

	go func() {
		defer close(peerChan)

		if _, err := d.server.Bootstrap(); err != nil {
			log.Printf("[dht] bootstrap failed: %v", err)
		}

		// announce immediately first
		d.announceOnce(peerChan)

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				return
			case <-ticker.C:
				d.announceOnce(peerChan)
			}
		}
	}()

	return peerChan, nil
}

// announceOnce performs a single announce to the DHT and sends discovered peers to the channel.
func (d *DHT) announceOnce(peerChan chan<- peer.Peer) {
	announce, err := d.server.AnnounceTraversal(d.infoHash)
	if err != nil {
		log.Printf("[dht] announce failed: %v", err)
		return
	}
	defer announce.Close()

	for peerValues := range announce.Peers {
		for _, p := range peerValues.Peers {
			p_peer := peer.Peer{
				IpAddr: p.IP,
				Port:   uint16(p.Port),
			}
			if p_peer.IpAddr == nil || p_peer.IpAddr.IsUnspecified() {
				continue // skip invalid IPs
			}

			select {
			case peerChan <- p_peer:
			case <-d.ctx.Done():
				return
			}
		}
	}
}

func (d *DHT) StopDHT() {
	d.cancel()
	d.server.Close()
}
