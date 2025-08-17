package dht

import (
	"fmt"
	"log"
	"time"

	"github.com/anacrolix/dht/v2"
)

func Start(infoHash [20]byte ) {
	config := dht.NewDefaultServerConfig()
	server, err := dht.NewServer(config)
	if err != nil {
		log.Fatalf("[dht] Failed to create DHT server: %v", err)
	}
	defer server.Close()

 
	// This starts the bootstrapping process, which connects to known DHT nodes.
	if stats, err := server.Bootstrap(); err != nil {
		log.Fatalf("[dht] Failed to bootstrap DHT: %v", err)
	} else {
		log.Printf("[dht] DHT bootstrap stats: %v", stats)
	}

	
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
				case <- ticker.C:
					announce, err := server.AnnounceTraversal(infoHash)
					if err != nil {
						log.Printf("[dht] Failed to announce: %v", err)
						continue
					}

					for peerValues := range announce.Peers {
						for _, peer := range peerValues.Peers {
							fmt.Printf("Peer: %s:%d\n", peer.IP, peer.Port)
							
						}
					}
					announce.Close()
			}
		}
	}()
}


