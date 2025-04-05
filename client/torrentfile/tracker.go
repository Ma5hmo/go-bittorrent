package torrentfile

import (
	"client/peering"
	"strings"
)

func (t *TorrentFile) RequestPeers(peerID *[20]byte, port uint16) ([]peering.Peer, error) {
	if strings.HasPrefix(t.Announce, "udp:") {
		return t.requestPeersUDP(port, peerID)
	}
	return t.requestPeersHTTP(peerID, port)
}
