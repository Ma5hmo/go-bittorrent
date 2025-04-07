package connect

import "client/peer"

type Torrent struct {
	Peers       []peer.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type pieceWork struct {
	index  int
	length int
	hash   *[20]byte
}

func (t *Torrent) Download() {
	piecesLeft := make(chan *pieceWork, len(t.PieceHashes))
	for index, hash := range t.PieceHashes {
		piecesLeft <- &pieceWork{index, t.PieceLength, &hash}
	}

}
