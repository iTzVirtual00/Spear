package config

import (
	"io/fs"
)

type ShareParams struct {
	ListenAddress string   // e.g. "0.0.0.0:9000"
	MaxDownloads  int      // 0 = unlimited
	ExpiresIn     int      // seconds; 0 = no expiration
	AllowContacts []string // list of allowed contact IDs
	NatPunch      bool     // enable NAT punchthrough
	//Files         []*utils.Path // files to share
	// New: virtual filesystem to serve
	MountFS fs.FS
}

type DownloadParams struct {
	PeerAddress string   // e.g. "host:port"
	Files       []string // specific files to download (empty = all)
	OutDir      string   // destination directory
	WithID      string   // use this identity to connect
	FromID      string   // other peer identifier (empty=no check)
	NatPunch    bool     // use NAT punchthrough
}

type ReceiveParams struct {
	Port     int      // port to listen on
	Single   bool     // accept a single file then exit
	Batch    bool     // keep accepting files until stopped
	Dest     string   // destination directory for incoming files
	AllowIDs []string // allowed sender IDs
	NatPunch bool     // enable NAT punchthrough advertisement
}

type SendParams struct {
	PeerAddress string // e.g. "host:port"
	Port        int    // if splitting host and port
	FromID      string // sender identifier for access control
	NatPunch    bool   // use NAT punchthrough
}
