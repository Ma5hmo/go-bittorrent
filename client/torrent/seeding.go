package torrent

import (
	"client/bitfield"
	"client/connection"
	"client/handshake"
	"client/message"
	"client/torrent/seedingstatus"
	"client/view/viewutils"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

// StartSeeder listens for incoming connections and seeds the torrent file to peers.
func (t *Torrent) StartSeeder() {
	defer func() { t.IsSeedingPaused = true }()

	log.Printf("[Seeder] StartSeeder called for torrent: %s", t.Name)
	// If seeding is paused, do not start
	if t.IsSeedingPaused {
		log.Printf("[Seeder] Seeding is paused for torrent: %s", t.Name)
		return
	}
	file, err := os.Open(t.Path)
	if err != nil {
		viewutils.ShowMessage("Error opening file seeding - " + err.Error())
		log.Printf("[Seeder] Error opening file for seeding: %v", err)
		return
	}
	t.Bitfield = make(bitfield.Bitfield, (len(t.PieceHashes)+7)/8)
	// Set bitfield for existing pieces
	for i := range t.PieceHashes {
		exists, err := t.checkExistingPiece(i, file)
		if err != nil {
			log.Printf("[Seeder] error reading file - %v", err)
			return
		}
		if exists {
			t.Bitfield.SetPiece(i)
		}
	}
	err = t.SendSeedingAnnounce(t.AnnounceList[0], t.Port, &t.PeerID, uint64(t.Length), 0)
	if err != nil {
		log.Printf("[Seeder] error sending seeding announce - %v", err)
		return
	}

	// Initialize seeding status if not already
	if t.SeedingStatus == nil {
		log.Printf("[Seeder] Initializing SeedingStatus for torrent: %s", t.Name)
		t.SeedingStatus = &seedingstatus.SeedingStatus{SeededBytes: 0, ActivePeers: 0}
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", t.Port))
	if err != nil {
		log.Printf("[Seeder] failed to listen on port %d: %v", t.Port, err)
		return
	}
	log.Printf("[Seeder] Seeder listening on port %d", t.Port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[Seeder] Failed to accept connection: %v", err)
			continue
		}
		log.Printf("[Seeder] Accepted connection from %v", conn.RemoteAddr())
		go t.handleSeederConn(conn, false) // IS ENCRYPTED
	}
}

func (t *Torrent) handleSeederConn(conn net.Conn, encrypted bool) {
	defer conn.Close()
	t.SeedingStatus.IncrementActivePeers()
	defer t.SeedingStatus.DecrementActivePeers()
	log.Printf("[Seeder] Connected to peer: %v", conn.RemoteAddr())
	var encConn io.ReadWriter = conn
	var err error
	if encrypted {
		log.Printf("[Seeder] Starting encrypted handshake with peer: %v", conn.RemoteAddr())
		// --- Encryption handshake: receive key/iv ---
		key := make([]byte, 32)
		iv := make([]byte, 16)
		if _, err := io.ReadFull(conn, key); err != nil {
			log.Printf("[Seeder] Failed to read key: %v", err)
			return
		}
		if _, err := io.ReadFull(conn, iv); err != nil {
			log.Printf("[Seeder] Failed to read iv: %v", err)
			return
		}
		encConn, err = connection.WrapConnWithAES(conn, key, iv)
		if err != nil {
			log.Printf("[Seeder] Failed to wrap conn with AES: %v", err)
			return
		}
	}
	// Use encConn for both reading and writing
	if !t.performHandshake(encConn) {
		log.Printf("[Seeder] performHandshake failed for peer: %v", conn.RemoteAddr())
		return
	}

	if !t.sendBitfield(encConn) {
		log.Printf("[Seeder] sendBitfield failed for peer: %v", conn.RemoteAddr())
		return
	}

	file, err := os.Open(t.Path)
	if err != nil {
		log.Printf("[Seeder] Failed to open file: %v", err)
		return
	}
	defer file.Close()

	log.Printf("[Seeder] Serving peer: %v", conn.RemoteAddr())
	t.servePeer(encConn, file)
}

func (t *Torrent) performHandshake(rw io.ReadWriter) bool {
	log.Printf("[Seeder] Performing handshake")
	hs, err := handshake.Read(rw)
	if err != nil {
		log.Printf("[Seeder] Handshake failed: %v", err)
		return false
	}
	if *hs.InfoHash != t.InfoHash {
		log.Printf("[Seeder] InfoHash mismatch: got %x, expected %x", hs.InfoHash, t.InfoHash)
		return false
	}
	resp := handshake.New(&t.InfoHash, &t.PeerID)
	_, err = rw.Write(resp.Serialize())
	if err != nil {
		log.Printf("[Seeder] Failed to send handshake: %v", err)
		return false
	}
	log.Printf("[Seeder] Handshake successful")
	return true
}

func (t *Torrent) sendBitfield(rw io.ReadWriter) bool {
	log.Printf("[Seeder] Sending bitfield")
	bitfieldMsg := &message.Message{ID: message.MsgBitfield, Payload: t.Bitfield}
	log.Printf("bitfield - %v", t.Bitfield)
	_, err := rw.Write(bitfieldMsg.Serialize())
	if err != nil {
		log.Printf("[Seeder] Failed to send bitfield: %v", err)
		return false
	}
	log.Printf("[Seeder] Bitfield sent successfully")
	return true
}

func (t *Torrent) servePeer(rw io.ReadWriter, file *os.File) {
	log.Printf("[Seeder] servePeer started")
	interested := false
	for {
		msg, err := message.Read(rw)
		if err != nil {
			if err == io.EOF {
				log.Printf("[Seeder] Peer closed connection (EOF)")
				return
			}
			log.Printf("[Seeder] Read error from peer: %v", err)
			return
		}
		if msg == nil {
			log.Printf("[Seeder] Received keep-alive from peer")
			continue // keep-alive
		}
		log.Printf("[Seeder] Received message from peer: ID=%d", msg.ID)
		t.handlePeerMessage(msg, rw, file, &interested)
	}
}

func (t *Torrent) handlePeerMessage(msg *message.Message, rw io.ReadWriter, file *os.File, interested *bool) {
	switch msg.ID {
	case message.MsgUnchoke:
		log.Printf("[Seeder] Received UNCHOKE from peer")
	case message.MsgInterested:
		log.Printf("[Seeder] Received INTERESTED from peer")
		t.handleInterested(rw, interested)
	case message.MsgRequest:
		log.Printf("[Seeder] Received REQUEST from peer")
		t.handleRequest(msg, rw, file, *interested)
	default:
		log.Printf("[Seeder] Received unknown message ID: %d", msg.ID)
	}
}

func (t *Torrent) handleInterested(rw io.ReadWriter, interested *bool) {
	*interested = true
	unchoke := &message.Message{ID: message.MsgUnchoke}
	_, err := rw.Write(unchoke.Serialize())
	if err != nil {
		log.Printf("[Seeder] Failed to send unchoke: %v", err)
	} else {
		log.Printf("[Seeder] Sent UNCHOKE to peer")
	}
}

func (t *Torrent) handleRequest(msg *message.Message, rw io.ReadWriter, file *os.File, interested bool) {
	if !interested {
		log.Printf("[Seeder] Received request from uninterested peer")
		return
	}
	if len(msg.Payload) != 12 {
		log.Printf("[Seeder] Received request with invalid payload length: %d", len(msg.Payload))
		return
	}
	index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))
	log.Printf("[Seeder] Received request: index=%d, begin=%d, length=%d", index, begin, length)
	if index < 0 || index >= len(t.PieceHashes) {
		log.Printf("[Seeder] Received request for invalid piece index: %d", index)
		return
	}
	pieceBegin := index * t.PieceLength
	pieceEnd := pieceBegin + t.PieceLength
	if pieceEnd > t.Length {
		pieceEnd = t.Length
	}
	if begin+length > pieceEnd-pieceBegin {
		log.Printf("[Seeder] Received request with invalid begin/length: begin=%d, length=%d, piece size=%d", begin, length, pieceEnd-pieceBegin)
		return
	}
	buf := make([]byte, length)
	_, err := file.ReadAt(buf, int64(pieceBegin+begin))
	if err != nil {
		log.Printf("[Seeder] Failed to read from file: %v", err)
		return
	}
	// Optionally verify hash
	if begin == 0 && length == pieceEnd-pieceBegin {
		h := sha1.Sum(buf)
		if h != t.PieceHashes[index] {
			log.Printf("[Seeder] Hash mismatch for piece %d", index)
			return
		}
	}
	// Send piece
	payload := make([]byte, 8+len(buf))
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	copy(payload[8:], buf)
	pieceMsg := &message.Message{ID: message.MsgPiece, Payload: payload}
	_, err = rw.Write(pieceMsg.Serialize())
	if err != nil {
		log.Printf("[Seeder] Failed to send piece: %v", err)
	} else {
		log.Printf("[Seeder] Sent piece: index=%d, begin=%d, length=%d", index, begin, length)
	}
	// Update seeding status
	if t.SeedingStatus != nil {
		t.SeedingStatus.IncrementSeededBytes(int64(len(buf)))
	}
}
