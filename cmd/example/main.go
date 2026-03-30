package main

import (
	"context"
	"log"

	torrent "github.com/JoelVCrasta/clover"
	"github.com/JoelVCrasta/clover/client"
	"github.com/JoelVCrasta/clover/download"
	"github.com/JoelVCrasta/clover/metainfo"
	"github.com/JoelVCrasta/clover/peer"
)

func main() {
	var tr metainfo.Torrent
	err := tr.Torrent("assets/sc.torrent", ".")
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

	// this is the global context for stopping the torrent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pC, err := torrent.StartPeerDiscovery(ctx, tr.AnnounceList, tr.InfoHash, peerId)
	if err != nil {
		log.Fatal(err)
	}

	client := client.NewClient(ctx, pC, tr.InfoHash, peerId)
	apC := client.StartClient()

	dm := download.NewDownloadManager(ctx, tr, client)
	dm.StartDownload(apC)

}
