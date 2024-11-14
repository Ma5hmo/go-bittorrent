package tracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	"github.com/zeebo/bencode"
)

var httpClient = http.Client{
	Timeout: 2 * time.Second,
}

func SendAnnounceHTTP(urlStr string, infoHash string) (peers []Peer, err error) {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return
	}
	peerID := getPeerID()

	q := req.URL.Query()
	q.Add("info_hash", infoHash)
	q.Add("peer_id", string(peerID[:]))
	q.Add("port", "6881")
	q.Add("uploaded", "0")
	q.Add("downloaded", "0")
	q.Add("left", "10000")
	q.Add("compact", "1")
	q.Add("event", "started")
	req.URL.RawQuery = q.Encode()
	fmt.Println("sending ", req.URL)

	res, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	// Can add more fields here to get
	var decodedRes struct {
		Peers bencode.RawMessage `bencode:"peers"`
	}
	err = bencode.NewDecoder(res.Body).Decode(&decodedRes)
	if err != nil {
		return
	}

	if len(decodedRes.Peers) > 0 {
		peers = make([]Peer, len(decodedRes.Peers)/6)
		err = binary.Read(bytes.NewBuffer(decodedRes.Peers), binary.BigEndian, &peers)
	}
	return
}
