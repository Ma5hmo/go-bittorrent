package torrentfile

import (
	"client/peer"
	"fmt"
	"strings"
)

func (t *TorrentFile) RequestPeers(peerID *[20]byte, port uint16) ([]peer.Peer, error) {
	var peers []peer.Peer

	for _, announce := range t.AnnounceList {
		newPeers, err := t.requestPeersFromAnnounce(announce, port, peerID)
		if err == nil {
			peers = append(peers, newPeers...)
		}
	}

	if len(peers) == 0 {
		return nil, fmt.Errorf("received no peers from announces (%v)", t.AnnounceList)
	}
	return peers, nil
}

func (t *TorrentFile) requestPeersFromAnnounce(announce string, port uint16, peerID *[20]byte) ([]peer.Peer, error) {
	if strings.HasPrefix(announce, "udp:") {
		return t.requestPeersUDP(port, peerID, announce)
	}
	return t.requestPeersHTTP(port, peerID, announce)
}
