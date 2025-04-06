package torrentfile

import (
	"client/peer"
	"strings"
)

func (t *TorrentFile) RequestPeers(peerID *[20]byte, port uint16) ([]peer.Peer, error) {
	if strings.HasPrefix(t.Announce, "udp:") {
		return t.requestPeersUDP(port, peerID)
	}
	return t.requestPeersHTTP(peerID, port)
}
