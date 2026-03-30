package torrent

import (
	"context"
	"fmt"
	"os"
	"os/signal"

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

	// this is the global context for stopping the torrent
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Println("Searching for peers...")
	pC, err := StartPeerDiscovery(ctx, tr.AnnounceList, tr.InfoHash, peerId)
	if err != nil {
		return err
	}

	fmt.Println("Started download...")
	client := client.NewClient(ctx, pC, tr.InfoHash, peerId)
	apC := client.StartClient()

	dm := download.NewDownloadManager(ctx, tr, client)

	// go StartTUI(dm)
	dm.StartDownload(apC)

	return nil
}
