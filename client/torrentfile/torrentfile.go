package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/zeebo/bencode"
)

// TorrentFile encodes the metadata from a .torrent file
type TorrentFile struct {
	AnnounceList []string
	InfoHash     [20]byte
	PieceHashes  [][20]byte
	PieceLength  int
	Length       int
	Name         string
	Started      bool
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce     string             `bencode:"announce"`
	AnnounceList [][]string         `bencode:"announce-list"`
	InfoRaw      bencode.RawMessage `bencode:"info"`
}

// Open parses a torrent file
func Open(path string) (TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return TorrentFile{}, err
	}
	defer file.Close()

	bto := bencodeTorrent{}
	err = bencode.NewDecoder(file).Decode(&bto)
	if err != nil {
		return TorrentFile{}, err
	}
	return bto.toTorrentFile()
}

func (b *bencodeTorrent) hash() [20]byte {
	return sha1.Sum([]byte(b.InfoRaw))
}

func (b *bencodeTorrent) getInfo() (*bencodeInfo, error) {
	bi := &bencodeInfo{}
	err := bencode.NewDecoder(bytes.NewReader(b.InfoRaw)).Decode(bi)
	if err != nil {
		return nil, err
	}
	return bi, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash := bto.hash()
	info, err := bto.getInfo()
	if err != nil {
		return TorrentFile{}, err
	}

	pieceHashes, err := info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}

	fixedAnnounceList := []string{bto.Announce}
	for _, arr := range bto.AnnounceList {
		if len(arr) == 1 {
			fixedAnnounceList = append(fixedAnnounceList, arr[0])
		}
	}
	t := TorrentFile{
		AnnounceList: fixedAnnounceList,
		InfoHash:     infoHash,
		PieceHashes:  pieceHashes,
		PieceLength:  info.PieceLength,
		Length:       info.Length,
		Name:         info.Name,
		Started:      false,
	}
	return t, nil
}
