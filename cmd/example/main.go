package main

import (
	"log"

	torrent "github.com/JoelVCrasta"
	"github.com/JoelVCrasta/client"
	"github.com/JoelVCrasta/download"
	"github.com/JoelVCrasta/parsing"
	"github.com/JoelVCrasta/peer"
)

func main() {
	var tr parsing.Torrent
	err := tr.Torrent("assets/sc.torrent")
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

	pC, cancel, err := torrent.StartPeerDiscovery(tr.AnnounceList, tr.InfoHash, peerId)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	client := client.NewClient(pC, tr.InfoHash, peerId)
	apC := client.StartClient()

	dm := download.NewDownloadManager(tr, client)
	dm.StartDownload(apC)

}
