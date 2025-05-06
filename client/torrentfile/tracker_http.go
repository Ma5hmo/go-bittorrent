package torrentfile

import (
	"bytes"
	"client/peer"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/zeebo/bencode"
)

type bencodeTrackerResp struct {
	Interval int                `bencode:"interval"`
	Peers    bencode.RawMessage `bencode:"peers"`
}

func (t *TorrentFile) buildTrackerURL(peerID *[20]byte, port uint16, announce string) (string, error) {
	base, err := url.Parse(announce)
	if err != nil {
		return "", err
	}
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string((*peerID)[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}

func (t *TorrentFile) requestPeersHTTP(port uint16, peerID *[20]byte, announce string) ([]peer.Peer, error) {
	// Build the tracker URL
	url, err := t.buildTrackerURL(peerID, port, announce)
	if err != nil {
		return nil, err
	}
	log.Printf("Url - %s", url)
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	trackerResp := bencodeTrackerResp{}
	err = bencode.NewDecoder(resp.Body).Decode(&trackerResp)
	if err != nil {
		return nil, err
	}

	peers, err := peer.UnmarshalBinary([]byte(trackerResp.Peers))
	if err == nil {
		return peers, nil
	}

	log.Printf("Couldnt decode as binary, resolving to dict")

	// Fallback: try to decode as dictionary model
	var dictPeers []map[string]interface{}
	err = bencode.NewDecoder(bytes.NewReader(trackerResp.Peers)).Decode(&dictPeers)
	log.Printf("dictpeers - %v", dictPeers)
	if err != nil {
		log.Printf("Couldnt decode as dict - %v", err)
		return nil, err
	}

	return peer.UnmarshalDict(dictPeers)
}
