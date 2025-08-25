package main

import (
	"context"
	"log"

	"github.com/JoelVCrasta/client"
	"github.com/JoelVCrasta/dht"
	"github.com/JoelVCrasta/download"
	"github.com/JoelVCrasta/parsing"
	"github.com/JoelVCrasta/peer"
	"github.com/JoelVCrasta/tracker"
)

func main() {
	var tr parsing.Torrent
	err := tr.Torrent("assets/arch.torrent")
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println(tr.AnnounceList)
	}

	// Generate a random transaction ID
	peerId, err := peer.GeneratePeerID()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Peer ID:", string(peerId[:]))

	tm := tracker.NewTrackerManager(tr.AnnounceList, tr.InfoHash, peerId)
	dht, err := dht.NewDHT(tr.InfoHash)
	if err != nil {
		log.Fatal(err)
	}

	tC, err := tm.StartTracker()
	if err != nil {
		log.Fatal(err)
	}
	dhtC, err := dht.Start()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pC := peer.MergeStream(tC, dhtC, ctx)

	client := client.NewClient(pC, tr.InfoHash, peerId)
	apC := client.StartClient()

	dm := download.NewDownloadManager(tr, client)
	dm.StartDownload(apC)

}
