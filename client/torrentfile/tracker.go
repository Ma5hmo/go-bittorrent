package torrentfile

import (
	"client/peer"
	"fmt"
	"log"
	"strings"
)

func (t *TorrentFile) RequestPeers(peerID *[20]byte, port uint16) ([]peer.Peer, error) {
	const MaxPeersAmount = 5
	var peers []peer.Peer

	for _, announce := range t.AnnounceList {
		newPeers, err := t.requestPeersFromAnnounce(announce, port, peerID)
		if err == nil {
			peers = append(peers, newPeers...)
			peers = removeDuplicates(peers)
			if len(peers) > MaxPeersAmount {
				break
			}
			log.Printf("got peers from %v - %v", announce, peers)
		} else {
			log.Printf("requesting peers from %v - %v", announce, err)
		}
	}

	if len(peers) == 0 {
		return nil, fmt.Errorf("received no peers from announces (%v)", t.AnnounceList)
	}
	return peers, nil
}

func (t *TorrentFile) requestPeersFromAnnounce(announce string, port uint16,
	peerID *[20]byte) ([]peer.Peer, error) {
	announce, isUDP := strings.CutPrefix(announce, "udp://")
	if isUDP {
		announce, _ = strings.CutSuffix(announce, "/announce")
		return t.requestPeersUDP(port, peerID, announce)
	}
	return t.requestPeersHTTP(port, peerID, announce)
}

func (t *TorrentFile) SendSeedingAnnounce(announce string, port uint16,
	peerID *[20]byte, downloaded, uploaded uint64) error {
	announce, isUDP := strings.CutPrefix(announce, "udp://")
	if isUDP {
		announce, _ = strings.CutSuffix(announce, "/announce")
		return t.sendSeedingAnnounceUDP(port, peerID, announce, downloaded, uploaded)
	}
	return t.sendSeedingAnnounceHTTP(port, peerID, announce, downloaded, uploaded)

}

func removeDuplicates(sliceList []peer.Peer) []peer.Peer {
	allKeys := make(map[[4]byte]bool)
	list := []peer.Peer{}
	for _, item := range sliceList {
		if _, value := allKeys[[4]byte(item.IP)]; !value {
			allKeys[[4]byte(item.IP)] = true
			list = append(list, item)
		}
	}
	return list
}
