package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
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
	Path         string // Path to the actual file to seed (not bencoded)
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
	}
	return t, nil
}

// CreateFromFile creates a TorrentFile from a file and metadata
func CreateFromFile(filePath, announce, torrentName, description string, pieceLength int) (*TorrentFile, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	length := int(fileInfo.Size())
	name := torrentName
	var pieceHashes [][20]byte
	buf := make([]byte, pieceLength)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hash := sha1.Sum(buf[:n])
			pieceHashes = append(pieceHashes, hash)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	tf := TorrentFile{
		AnnounceList: []string{announce},
		PieceHashes:  pieceHashes,
		PieceLength:  pieceLength,
		Length:       length,
		Name:         name,
		Path:         filePath,
	}
	// Generate infohash
	infoDict := map[string]interface{}{
		"name":         name,
		"length":       length,
		"piece length": pieceLength,
		"pieces":       tf.piecesString(),
		"description":  description,
	}
	infoBencode, err := bencode.EncodeBytes(infoDict)
	if err != nil {
		return nil, err
	}
	tf.InfoHash = sha1.Sum(infoBencode)
	return &tf, nil
}

// SaveToFile bencodes and saves the TorrentFile to disk
func (tf *TorrentFile) SaveToFile(path string) error {
	infoDict := map[string]interface{}{
		"name":         tf.Name,
		"length":       tf.Length,
		"piece length": tf.PieceLength,
		"pieces":       tf.piecesString(),
	}
	torrent := map[string]interface{}{
		"announce": tf.AnnounceList[0],
		"info":     infoDict,
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return bencode.NewEncoder(out).Encode(torrent)
}

// piecesString returns the concatenated piece hashes as a string
func (tf *TorrentFile) piecesString() string {
	pieces := make([]byte, 0, len(tf.PieceHashes)*20)
	for _, h := range tf.PieceHashes {
		pieces = append(pieces, h[:]...)
	}
	return string(pieces)
}
