package config

import "time"

type GlobalConfig struct {
	MinPeers              int
	Port                  uint16
	TrackerConnectTimeout time.Duration
	PeerHandshakeTimeout  time.Duration
	PieceMessageTimeout   time.Duration
	DownloadDirectory     string
	MaxConnections        int
	PeerId                [20]byte
}

var Config = GlobalConfig{
	MinPeers:              10,
	Port:                  6881,
	TrackerConnectTimeout: 5 * time.Second,
	PeerHandshakeTimeout:  3 * time.Second,
	PieceMessageTimeout:   25 * time.Second,
	DownloadDirectory:     "./Downloads",
	MaxConnections:        100,
}
