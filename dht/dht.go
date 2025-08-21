package dht

import (
	"fmt"
	"log"
	"time"

	"github.com/JoelVCrasta/peer"
	"github.com/anacrolix/dht/v2"
)

type DHT struct {
	server   *dht.Server
	infoHash [20]byte
	quit     chan struct{}
}

func NewDHT(infoHash [20]byte) (*DHT, error) {
	config := dht.NewDefaultServerConfig()

	server, err := dht.NewServer(config)
	if err != nil {
		return nil, fmt.Errorf("[dht] failed to create DHT server: %w", err)
	}

	return &DHT{
		server:   server,
		infoHash: infoHash,
		quit:     make(chan struct{}),
	}, nil
}

/*
Start starts the DHT bootstrapping process and begins announcing the info hash.
It returns a channel that will receive discovered peers.
It will periodically announce the info hash every 5 minutes.
*/
func (d *DHT) Start() (<-chan peer.Peer, error) {
	if stats, err := d.server.Bootstrap(); err != nil {
		d.server.Close()
		return nil, fmt.Errorf("[dht] bootstrap failed: %w", err)
	} else {
		log.Printf("[dht] bootstrap stats: %v", stats)
	}

	peerChan := make(chan peer.Peer)

	go func() {
		defer close(peerChan)

		// announce immediately first
		d.announceOnce(peerChan)

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-d.quit:
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
			case peerChan <- peer.Peer{IpAddr: p.IP, Port: uint16(p.Port)}:
			case <-d.quit: // exit quickly if stopped
				return
			}
		}
	}
}

func (d *DHT) Stop() {
	close(d.quit)
	d.server.Close()
	log.Println("[dht] stopped")
}
