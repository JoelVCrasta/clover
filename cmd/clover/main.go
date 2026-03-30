package main

import (
	"flag"
	"fmt"
	"os"

	torrent "github.com/JoelVCrasta/clover"
)

func main() {
	input := flag.String("i", "", "Path to the .torrent file")
	output := flag.String("o", "", "Path to the download directory (Default: ~/Downloads)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "clover is a lightweight, fast leech only torrent client.\n")
		fmt.Fprintf(os.Stderr, "Usage: clover -i <torrentfile> -o <outputdir>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *input == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *output == "." {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		*output = cwd
	}

	err := torrent.StartTorrent(*input, *output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
