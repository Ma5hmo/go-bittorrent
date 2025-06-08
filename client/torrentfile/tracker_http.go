package torrentfile

import (
	"client/peer"
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zeebo/bencode"
)

type bencodeTrackerResp struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func (t *TorrentFile) buildTrackerURL(peerID *[20]byte, port uint16, announce, uploaded, downloaded, event string) (string, error) {
	base, err := url.Parse(announce)
	if err != nil {
		return "", err
	}
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string((*peerID)[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{uploaded},
		"downloaded": []string{downloaded},
		"event":      []string{event},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	base.RawQuery = params.Encode()
	return base.String(), nil
}

func (t *TorrentFile) sendAnnounceHTTP(port uint16, peerID *[20]byte, announce, uploaded, downloaded, event string) ([]peer.Peer, error) {
	// Build the tracker URL
	url, err := t.buildTrackerURL(peerID, port, announce, uploaded, downloaded, event)
	if err != nil {
		return nil, err
	}
	// log.Printf("Url - %s", url)

	var zeroDialer net.Dialer
	c := &http.Client{Timeout: 15 * time.Second}
	// force ipv4
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return zeroDialer.DialContext(ctx, "tcp4", addr)
	}
	c.Transport = transport

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

	// log.Printf("Couldnt decode as binary, resolving to dict")

	// Fallback: try to decode as dictionary model
	var dictPeers []map[string]interface{}
	err = bencode.NewDecoder(strings.NewReader(trackerResp.Peers)).Decode(&dictPeers)
	// log.Printf("dictpeers - %v", dictPeers)
	if err != nil {
		// log.Printf("Couldnt decode as dict - %v, DATA=%s", err, trackerResp.Peers)
		return nil, err
	}

	return peer.UnmarshalDict(dictPeers)
}

func (t *TorrentFile) requestPeersHTTP(port uint16, peerID *[20]byte,
	announce string) ([]peer.Peer, error) {
	return t.sendAnnounceHTTP(port, peerID, announce, "0", "0", "started")
}

func (t *TorrentFile) sendSeedingAnnounceHTTP(port uint16, peerID *[20]byte, announce string,
	uploaded, downloaded uint64) error {
	_, err := t.sendAnnounceHTTP(port, peerID, announce, strconv.FormatUint(uploaded, 10), strconv.FormatUint(downloaded, 10), "")
	return err
}
