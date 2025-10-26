package torrent

import (
	"github.com/JoelVCrasta/clover/client"
	"github.com/JoelVCrasta/clover/download"
	"github.com/JoelVCrasta/clover/metainfo"
	"github.com/JoelVCrasta/clover/peer"
)

func StartTorrent(inputPath string, outputPath string) error {
	var tr metainfo.Torrent
	err := tr.Torrent(inputPath, outputPath)
	if err != nil {
		return err
	}

	peerId, err := peer.GeneratePeerID()
	if err != nil {
		return err
	}

	pC, cancel, err := StartPeerDiscovery(tr.AnnounceList, tr.InfoHash, peerId)
	if err != nil {
		return err
	}
	defer cancel()

	client := client.NewClient(pC, tr.InfoHash, peerId)
	apC := client.StartClient()

	dm := download.NewDownloadManager(tr, client)
	// go StartTUI(dm.Stats)
	dm.StartDownload(apC)

	return nil
}
