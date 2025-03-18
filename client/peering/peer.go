package peering

import "math/rand/v2"

type Peer struct {
	IP   [4]byte
	Port uint16
}

func GetPeerID() (peerID [20]byte) {
	peerID = [20]byte{} // create a random peerId (temporarily)
	for i := 0; i < 20; i++ {
		peerID[i] = byte(rand.UintN(256))
	}
	return
}
