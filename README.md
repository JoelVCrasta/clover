# Clover

Clover (🍀) is a lightweight, fast, leech-only torrent client written in Go. It implements the core BitTorrent protocol to download torrents directly from the terminal. It handles everything from parsing .torrent files and discovering peers to managing concurrent piece downloads.



## Features


- Torrent parsing - Implements an encoder and decoder for parsing bencode encoded .torrent files.
- Dual peer discovery - Finds peers via both UDP trackers and the DHT network, merging them into a single stream.
- Concurrent downloads - Manages multiple peer connections to download pieces simultaneously. 
- Clean CLI stats - Real-time stats showing a progress bar, percentage completed, pieces downloaded, active peer count, and time elapsed.


## Getting Started
 
### Prerequisites
 
- Go 
 
### Installation
 
```bash
git clone https://github.com/JoelVCrasta/clover.git
cd clover
go build ./cmd/clover
```

Make sure to move the clover binary to your system or user bin folder so it works from anywhere.

**In Linux:**
```bash
sudo mv ./clover /usr/local/bin/
```

## Usage/Examples

### Usage
 
```bash
go run ./cmd/clover/main.go -i <path-to-torrent-file> -o <output-directory>

or

clover -i <path-to-torrent-file> -o <output-directory>
```
 
### Examples
 
```bash
go run ./cmd/clover/main.go -i ~/downloads/ubuntu.torrent -o ~/downloads/

or 

clover -i ~/downloads/ubuntu.torrent -o ~/downloads/
```

```bash
go run ./cmd/clover/main.go -i ~/downloads/ubuntu.torrent

or 

clover -i ~/downloads/ubuntu.torrent
```
If the output flag is not provided, then it will download to the ~/Downloads directory.

## Project Structure

.
├── client
│   ├── bitfield.go
│   └── client.go
├── cmd
│   ├── clover
│   │   └── main.go
│   └── example
│       └── main.go
├── config
│   └── config.go
├── dht
│   └── dht.go
├── download
│   ├── download.go
│   └── save.go
├── handshake
│   └── handshake.go
├── message
│   └── message.go
├── metainfo
│   ├── decode.go
│   ├── encode.go
│   ├── hash.go
│   └── torrentfile.go
├── peer
│   ├── peer.go
│   └── peer_id.go
├─── tracker
│   ├── scrape.go
│   ├── tracker.go
│   └── tracker_test.go
├── torrent.go
├── discover_peers.go
├── go.mod
└── go.sum