package peer

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func (p *Peer) String() string {
	ipStr := p.IP.String()
	if p.IP.To4() == nil {
		// It's an IPv6 address, wrap in brackets
		return fmt.Sprintf("[%s]:%d", ipStr, p.Port)
	}
	return fmt.Sprintf("%s:%d", ipStr, p.Port)
}

func UnmarshalBinary(peersBin []byte) ([]Peer, error) {
	const peerSize = 6 // 4 for IP, 2 for port
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		err := fmt.Errorf("received malformed peers")
		return nil, err
	}
	peers := make([]Peer, numPeers)
	for i := range numPeers {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+4])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+4 : offset+6])
	}
	return peers, nil
}

func UnmarshalDict(dicts []map[string]interface{}) ([]Peer, error) {
	peers := make([]Peer, 0, len(dicts))
	for _, entry := range dicts {
		ipStr, ok := entry["ip"].(string)
		if !ok {
			continue
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		portFloat, ok := entry["port"].(int64) // bencode may decode ints as int64
		if !ok {
			portFloat2, ok := entry["port"].(uint64) // fallback
			if ok {
				portFloat = int64(portFloat2)
			} else {
				continue
			}
		}

		peers = append(peers, Peer{
			IP:   ip,
			Port: uint16(portFloat),
		})
	}
	return peers, nil
}
