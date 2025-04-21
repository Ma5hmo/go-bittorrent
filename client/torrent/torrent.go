package torrent

import (
	"client/connection"
	"client/message"
	"client/peer"
	"client/torrentfile"
	"crypto/sha1"
	"fmt"
	"log"
	"runtime"
	"time"
)

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

type pieceResult struct {
	index int
	buf   []byte
}

type pieceStatus struct {
	downloaded int
	requested  int
	pieceIndex int
	backlog    int
	buf        []byte
	connection *connection.Connection
}

// MaxBlockSize is the largest number of bytes a request can ask for
const MaxBlockSize = 0x4000

// MaxBacklog is the number of unfulfilled requests a client can have in its pipeline
const MaxBacklog = 5

func New(tf *torrentfile.TorrentFile, peerID *[20]byte, port uint16) (*Torrent, error) {
	peers, err := tf.RequestPeers(peerID, port)
	if err != nil {
		return nil, err
	}
	return &Torrent{
		Peers:       peers,
		PeerID:      *peerID,
		PieceHashes: tf.PieceHashes,
		Length:      tf.Length,
		Name:        tf.Name,
		PieceLength: tf.PieceLength,
	}, nil
}

func (t *Torrent) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = begin + t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

func (t *Torrent) calculatePieceSize(index int) (size int) {
	begin, end := t.calculateBoundsForPiece(index)
	return end - begin
}

// Handles recieving data from the connection and updating the status as needed
func (s *pieceStatus) recieveData() error {
	msg, err := s.connection.Read()
	if err != nil {
		return err
	}
	switch msg.ID {
	case message.MsgChoke:
		s.connection.Choked = true
	case message.MsgUnchoke:
		s.connection.Choked = false
	case message.MsgHave:
		index, err := msg.ParseHave()
		if err != nil {
			return err
		}
		s.connection.Bitfield.SetPiece(index)
	case message.MsgPiece:
		lengthRecieved, err := msg.ParsePiece(s.pieceIndex, s.buf)
		if err != nil {
			return err
		}
		s.downloaded += lengthRecieved
		s.backlog--
	}
	return nil
}

func attemptDownloadPiece(c *connection.Connection, pw *pieceWork) ([]byte, error) {
	status := pieceStatus{
		connection: c,
		pieceIndex: pw.index,
		buf:        make([]byte, pw.length),
	}

	// set a deadline for this piece
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{})

	for status.downloaded < pw.length {
		if !status.connection.Choked {
			for status.backlog < MaxBacklog && status.requested < pw.length {
				reqSize := MaxBlockSize
				if status.downloaded+MaxBlockSize > pw.length {
					reqSize = pw.length - status.downloaded
				}

				err := c.SendRequest(pw.index, status.downloaded, reqSize)
				if err != nil {
					return nil, err
				}
				status.requested += reqSize
				status.backlog++
			}
		}

		err := status.recieveData()
		if err != nil {
			return nil, err
		}
	}
	return status.buf, nil
}

func checkIntegrity(pw *pieceWork, buf []byte) error {
	h := sha1.Sum(buf)
	if h != *pw.hash {
		return fmt.Errorf("index %d failed integrity check", pw.index)
	}
	return nil
}

func (t *Torrent) startDownloadWorker(peer peer.Peer, workQueue chan *pieceWork,
	resultsQueue chan *pieceResult) {
	c, err := connection.New(peer, t.PeerID, t.InfoHash)
	if err != nil {
		log.Printf("Could not handshake with %s. Disconnecting\n", peer.IP)
		return
	}
	defer c.Conn.Close()

	c.SendUnchoke()
	c.SendInterested()

	for pw := range workQueue {
		if !c.Bitfield.HasPiece(pw.index) {
			workQueue <- pw // Put piece back on the queue
			continue
		}

		// Download the piece
		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			log.Println("Exiting", err)
			workQueue <- pw // Put piece back on the queue
			return
		}

		err = checkIntegrity(pw, buf)
		if err != nil {
			log.Printf("Piece #%d failed integrity check\n", pw.index)
			workQueue <- pw // Put piece back on the queue
			continue
		}

		c.SendHave(pw.index)
		resultsQueue <- &pieceResult{pw.index, buf}
	}
}

func (t *Torrent) Download() []byte {
	// Init queues for workers to retrieve work and send results
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)
	for index, hash := range t.PieceHashes {
		length := t.calculatePieceSize(index)
		workQueue <- &pieceWork{index, length, &hash}
	}

	// Start workers
	for _, peer := range t.Peers {
		go t.startDownloadWorker(peer, workQueue, results)
	}

	// Collect results into a buffer until full
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-results
		begin, end := t.calculateBoundsForPiece(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		// TEMPORARILY
		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		numWorkers := runtime.NumGoroutine() - 1 // subtract 1 for main thread
		log.Printf("(%0.2f%%) Downloaded piece #%d from %d peers\n", percent, res.index, numWorkers)
	}
	close(workQueue)
	return buf
}
