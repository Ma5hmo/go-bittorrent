package torrent

import (
	"client/peering"
	"io"
)

type TorrentStatus struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Downloaded uint64
	Left       uint64
	Uploaded   uint64
	Event      uint32
	Peers      []peering.Peer
	NumWant    uint32
	Interval   uint32
	Leechers   uint32
	Seeders    uint32
}

// func NewTorrentStatus(infoHash [20]byte, peerID [20]byte, downloaded uint64, left uint64, uploaded uint64, event uint32, numWant uint32, interval uint32, leechers uint32, seeders uint32) *TorrentStatus {
// 	return &TorrentStatus{InfoHash: infoHash, PeerID: peerID, Downloaded: downloaded, Left: left, Uploaded: uploaded, Event: event, NumWant: numWant, Interval: interval, Leechers: leechers, Seeders: seeders}
// }

func StartTorrent(file io.Reader) {
	DecodeTorrent(file)
	//.Get peers from all torrents
	// start connection with peers and recieve data
}
