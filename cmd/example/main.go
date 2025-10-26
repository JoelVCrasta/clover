package main

import (
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
