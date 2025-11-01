package torrent

import (
	"fmt"

	"github.com/JoelVCrasta/clover/client"
	"github.com/JoelVCrasta/clover/download"
	"github.com/JoelVCrasta/clover/metainfo"
	"github.com/JoelVCrasta/clover/peer"
)

func StartTorrent(inputPath string, outputPath string) error {
	fmt.Println("Reading torrent file...")
	var tr metainfo.Torrent
	err := tr.Torrent(inputPath, outputPath)
	if err != nil {
		return err
	}

	peerId, err := peer.GeneratePeerID()
	if err != nil {
		return err
	}

	fmt.Println("Searching for peers...")
	pC, cancel, err := StartPeerDiscovery(tr.AnnounceList, tr.InfoHash, peerId)
	if err != nil {
		return err
	}
	defer cancel()

	fmt.Printf("Started downloading: %s\n", tr.Info.Name)
	client := client.NewClient(pC, tr.InfoHash, peerId)
	apC := client.StartClient()

	dm := download.NewDownloadManager(tr, client)
	go StartTUI(dm.Stats)
	dm.StartDownload(apC)
	fmt.Println("Download completed successfully")
	return nil
}
