package main

import (
	"flag"
	"fmt"
	"os"

	torrent "github.com/JoelVCrasta/clover"
)

func main() {
	input := flag.String("i", "", "Path to the .torrent file")
	output := flag.String("o", ".", "Output directory to save the downloaded files")

	flag.Parse()

	if *input == "" {
		fmt.Println("Usage: clover -i <torrentfile> -o <outputdir>")
		os.Exit(1)
	}

	err := torrent.StartTorrent(*input, *output)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}
