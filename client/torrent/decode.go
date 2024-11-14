package torrent

import (
	"crypto/sha1"
	"io"

	"github.com/zeebo/bencode"
)

// So far only gets the announce domains and the infoHash, can be changed easily
func DecodeTorrent(file io.Reader) (announceList [][]string, infoHash [20]byte, err error) {
	var metainfo struct {
		AnnounceList [][]string         `bencode:"announce-list"`
		Info         bencode.RawMessage `bencode:"info"`
	}
	err = bencode.NewDecoder(file).Decode(&metainfo)
	if err != nil || len(metainfo.Info) == 0 {
		return
	}
	announceList = metainfo.AnnounceList
	infoHash = sha1.Sum(metainfo.Info)
	return
}
