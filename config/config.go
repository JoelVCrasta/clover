package config

import "time"

type GlobalConfig struct {
	MinPeers               int
	Port                   uint16
	TrackerConnectTimeout  time.Duration
	PeerHandshakeTimeout   time.Duration
	PieceMessageTimeout    time.Duration
	DefaultTrackerInterval uint32
	DownloadDirectory      string
	MaxTrackerConnections  int
	MaxFailedRetries       int
	PeerId                 [20]byte
}

var Config = GlobalConfig{
	MinPeers:               10,
	Port:                   6881,
	TrackerConnectTimeout:  5 * time.Second,
	PeerHandshakeTimeout:   20 * time.Second,
	PieceMessageTimeout:    30 * time.Second,
	DefaultTrackerInterval: 1800, // 20 minutes
	DownloadDirectory:      "./Downloads",
	MaxTrackerConnections:  20,
	MaxFailedRetries:       3,
}
