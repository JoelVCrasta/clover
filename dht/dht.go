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

func NewDHT(infoHash [20]byte) (*DHT, error) {
	config := dht.NewDefaultServerConfig()

	server, err := dht.NewServer(config)
	if err != nil {
		return nil, fmt.Errorf("[dht] failed to create DHT server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

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
	if stats, err := d.server.Bootstrap(); err != nil {
		d.server.Close()
		return nil, fmt.Errorf("[dht] bootstrap failed: %w", err)
	} else {
		log.Printf("[dht] bootstrap stats: %v", stats)
	}

	peerChan := make(chan peer.Peer, 500)

	go func() {
		defer close(peerChan)

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
			select {
			case <-d.ctx.Done(): // exit quickly if stopped
				return
			default:
				peer := peer.Peer{
					IpAddr: p.IP,
					Port:   uint16(p.Port),
				}
				if peer.IpAddr == nil || peer.IpAddr.IsUnspecified() {
					continue // skip invalid IPs
				}

				peerChan <- peer
			}
		}
	}
}

func (d *DHT) StopDHT() {
	d.cancel()
	d.server.Close()
	log.Println("[dht] stopped")
}
