package torrent

import (
	"client/bitfield"
	"client/common"
	"client/connection"
	"client/message"
	"client/peer"
	"client/torrent/seedingstatus"
	"client/torrent/torrentstatus"
	"client/torrentfile"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type Torrent struct {
	*torrentfile.TorrentFile
	DownloadStatus  *torrentstatus.TorrentStatus
	SeedingStatus   *seedingstatus.SeedingStatus // <-- Add this pointer
	Peers           []peer.Peer
	PeerID          [20]byte
	Port            uint16
	Paused          bool
	IsSeedingPaused bool              // true if seeding is paused, false if active
	Bitfield        bitfield.Bitfield // Bitfield representing downloaded pieces
	// Retrieved from TorrentFile:
	// InfoHash       [20]byte
	// PieceHashes    [][20]byte
	// PieceLength    int
	// Length         int
	// Name           string
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
	return &Torrent{
		TorrentFile:    tf,
		DownloadStatus: nil,
		Peers:          nil,
		PeerID:         *peerID,
		Port:           port,
		Bitfield:       nil,
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
	// read returns nil for a keep alive message
	if msg == nil {
		return nil
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
	state := pieceStatus{
		connection: c,
		pieceIndex: pw.index,
		buf:        make([]byte, pw.length),
	}

	// Setting a deadline helps get unresponsive peers unstuck.
	// 30 seconds is more than enough time to download a 262 KB piece
	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{}) // Disable the deadline

	for state.downloaded < pw.length {
		// If unchoked, send requests until we have enough unfulfilled requests
		if !state.connection.Choked {
			for state.backlog < MaxBacklog && state.requested < pw.length {
				blockSize := MaxBlockSize
				// Last block might be shorter than the typical block
				if pw.length-state.requested < blockSize {
					blockSize = pw.length - state.requested
				}

				err := c.SendRequest(pw.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}

		err := state.recieveData()
		if err != nil {
			return nil, err
		}
	}

	return state.buf, nil
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
	log.Printf("[DownloadWorker] Starting download worker for peer: %s", peer.String())
	c, err := connection.New(peer, &t.PeerID, &t.InfoHash, common.AppState.IsTrafficAESEncrypted) // IS ENCRYPTED
	if err != nil {
		log.Printf("[DownloadWorker] Could not handshake with %s - %s", peer.IP, err)
		t.DownloadStatus.DecrementPeersAmount()
		return
	}
	defer c.Conn.Close()

	c.SendUnchoke()
	c.SendInterested()
	for pw := range workQueue {
		// Check if download is paused
		if t.Paused {
			log.Printf("[DownloadWorker] Download paused, putting piece %d back on queue", pw.index)
			workQueue <- pw // Put piece back on the queue
			return
		}

		if !c.Bitfield.HasPiece(pw.index) {
			// // log.Printf("[DownloadWorker] Peer %s does not have piece %d, putting back on queue", peer.String(), pw.index)
			workQueue <- pw // Put piece back on the queue
			continue
		}

		// log.Printf("[DownloadWorker] Attempting to download piece %d from peer %s", pw.index, peer.String())
		// Download the piece
		buf, err := attemptDownloadPiece(c, pw)
		if err != nil {
			log.Printf("[DownloadWorker] Exiting worker for peer %s: %v", peer.String(), err)
			workQueue <- pw // Put piece back on the queue
			t.DownloadStatus.DecrementPeersAmount()
			return
		}

		err = checkIntegrity(pw, buf)
		if err != nil {
			// log.Printf("[DownloadWorker] Piece #%d failed integrity check", pw.index)
			workQueue <- pw // Put piece back on the queue
			continue
		}

		// log.Printf("[DownloadWorker] Downloaded and verified piece %d from peer %s", pw.index, peer.String())
		c.SendHave(pw.index)
		t.Bitfield.SetPiece(pw.index) // Update bitfield when piece is downloaded
		resultsQueue <- &pieceResult{pw.index, buf}
	}
	log.Printf("[DownloadWorker] Worker for peer %s finished", peer.String())
	t.DownloadStatus.DecrementPeersAmount()
}

// checkExistingPiece verifies if a piece already exists in the file and is valid
func (t *Torrent) checkExistingPiece(index int, file *os.File) (bool, error) {
	begin, end := t.calculateBoundsForPiece(index)
	pieceSize := end - begin

	// Read the piece from the file
	buf := make([]byte, pieceSize)
	_, err := file.ReadAt(buf, int64(begin))
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, err
	}

	// Verify the piece hash
	h := sha1.Sum(buf)
	return h == t.PieceHashes[index], nil
}

// scanExistingPieces checks which pieces are already downloaded and valid
func (t *Torrent) scanExistingPieces(file *os.File) (int, error) {
	donePieces := 0
	for i := range t.PieceHashes {
		exists, err := t.checkExistingPiece(i, file)
		if err != nil {
			return donePieces, err
		}
		if exists {
			donePieces++
		}
	}
	return donePieces, nil
}

func (t *Torrent) StartDownload(output *os.File) error {
	log.Printf("[Torrent] StartDownload called for %s", t.Name)
	var err error

	if t.DownloadStatus != nil {
		if t.DownloadStatus.DonePieces == len(t.PieceHashes) {
			log.Printf("[Torrent] file is already done")
			return nil
		}
		if t.Paused {
			t.Paused = false
		}
	}

	if output != nil {
		t.Path = output.Name()
	} else if t.Path != "" {
		output, err = os.OpenFile(t.Path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("[Torrent] Error opening file for download: %v", err)
			return err
		}
	} else {
		log.Printf("[Torrent] No file is presented to StartDownload")
		return fmt.Errorf("no file is presented to StartDownload")
	}
	defer output.Close()

	log.Printf("[Torrent] Initializing download status")
	t.DownloadStatus = &torrentstatus.TorrentStatus{DonePieces: 0, PeersAmount: 0}

	log.Printf("[Torrent] Initializing bitfield")
	t.Bitfield = make(bitfield.Bitfield, (len(t.PieceHashes)+7)/8)

	// Check for existing pieces
	log.Printf("[Torrent] Scanning for existing pieces in file")
	existingPieces, err := t.scanExistingPieces(output)
	if err != nil {
		log.Printf("[Torrent] error scanning existing pieces: %v", err)
		return fmt.Errorf("error scanning existing pieces: %v", err)
	}
	t.DownloadStatus.DonePieces = existingPieces

	// Set bitfield for existing pieces
	for i := range t.PieceHashes {
		exists, err := t.checkExistingPiece(i, output)
		if err != nil {
			log.Printf("[Torrent] Error checking existing piece %d: %v", i, err)
			return err
		}
		if exists {
			t.Bitfield.SetPiece(i)
		}
	}

	// If all pieces are already downloaded, we're done
	if existingPieces == len(t.PieceHashes) {
		log.Println("[Torrent] All pieces already downloaded!")
		return nil
	}

	// Get peers
	log.Printf("[Torrent] Requesting peers for download")
	t.Peers, err = t.RequestPeers(&t.PeerID, t.Port)
	if err != nil {
		log.Printf("[Torrent] Error requesting peers: %v", err)
		return err
	}
	t.DownloadStatus.PeersAmount = len(t.Peers)

	// Init queues for workers to retrieve work and send results
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)
	t.Paused = false

	// Only queue pieces that haven't been downloaded yet
	for index, hash := range t.PieceHashes {
		exists, err := t.checkExistingPiece(index, output)
		if err != nil {
			log.Printf("[Torrent] Error checking existing piece %d: %v", index, err)
			return err
		}
		if !exists {
			length := t.calculatePieceSize(index)
			workQueue <- &pieceWork{index, length, &hash}
		}
	}

	log.Printf("[Torrent] Starting download workers for %d peers", len(t.Peers))
	// Start workers
	for _, peer := range t.Peers {
		go t.startDownloadWorker(peer, workQueue, results)
	}

	// Collect results into a buffer until full
	for t.DownloadStatus.DonePieces < len(t.PieceHashes) {
		if t.Paused {
			log.Printf("[Torrent] Download paused, returning")
			return nil
		}

		res := <-results
		begin, end := t.calculateBoundsForPiece(res.index)

		t.DownloadStatus.IncrementDonePieces()
		// percent := t.CalculateDownloadPercentage()

		// log.Printf("[Torrent] (%0.2f%%) Downloaded piece #%d from %d peers", percent, res.index, t.DownloadStatus.GetPeersAmount())
		if _, err := output.WriteAt(res.buf[:end-begin], int64(begin)); err != nil {
			log.Printf("[Torrent] Error writing piece %d to file: %v", res.index, err)
			return err
		}
	}
	close(workQueue)
	log.Printf("[Torrent] Download complete for %s", t.Name)
	return nil
}

func (t *Torrent) PauseDownload() error {
	t.Paused = true
	return nil
}

func (t *Torrent) ResumeDownload() error {
	var err error
	t.Peers, err = t.RequestPeers(&t.PeerID, t.Port)
	if err != nil {
		log.Printf("[Torrent] Error requesting peers: %v", err)
		return err
	}
	t.Paused = false
	return nil
}

func (t *Torrent) CalculateDownloadPercentage() float64 {
	if t.DownloadStatus == nil {
		return 0
	}
	return float64(t.DownloadStatus.GetDonePieces()) / float64(len(t.PieceHashes)) * 100
}
