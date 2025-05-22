package torrent

import (
	"client/connection"
	"client/handshake"
	"client/message"
	"client/torrent/seedingstatus"
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
	// Initialize seeding status if not already
	if t.SeedingStatus == nil {
		t.SeedingStatus = &seedingstatus.SeedingStatus{SeededBytes: 0, ActivePeers: 0}
	}
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", t.Port))
	if err != nil {
		log.Printf("failed to listen on port %d: %v", t.Port, err)
		return
	}
	log.Printf("Seeder listening on port %d", t.Port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go t.handleSeederConn(conn)
	}
}

func (t *Torrent) handleSeederConn(conn net.Conn) {
	defer conn.Close()
	t.SeedingStatus.IncrementActivePeers()
	defer t.SeedingStatus.DecrementActivePeers()

	// --- Encryption handshake: receive key/iv ---
	key := make([]byte, 32)
	iv := make([]byte, 16)
	if _, err := io.ReadFull(conn, key); err != nil {
		log.Printf("Failed to read key: %v", err)
		return
	}
	if _, err := io.ReadFull(conn, iv); err != nil {
		log.Printf("Failed to read iv: %v", err)
		return
	}
	encConn, err := connection.WrapConnWithAES(conn, key, iv)
	if err != nil {
		log.Printf("Failed to wrap conn with AES: %v", err)
		return
	}
	// Use encConn for both reading and writing
	if !t.performHandshake(encConn) {
		log.Printf("performHandshake failed for peer")
		return
	}

	if !t.sendBitfield(encConn) {
		log.Printf("sendBitfield failed for peer")
		return
	}

	file, err := os.Open(t.Path)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	t.servePeer(encConn, file)
}

func (t *Torrent) performHandshake(rw io.ReadWriter) bool {
	hs, err := handshake.Read(rw)
	if err != nil {
		log.Printf("Handshake failed: %v", err)
		return false
	}
	if *hs.InfoHash != t.InfoHash {
		log.Printf("InfoHash mismatch: got %x, expected %x", hs.InfoHash, t.InfoHash)
		return false
	}
	resp := handshake.New(&t.InfoHash, &t.PeerID)
	_, err = rw.Write(resp.Serialize())
	if err != nil {
		log.Printf("Failed to send handshake: %v", err)
		return false
	}
	return true
}

func (t *Torrent) sendBitfield(rw io.ReadWriter) bool {
	bitfieldMsg := &message.Message{ID: message.MsgBitfield, Payload: t.Bitfield}
	_, err := rw.Write(bitfieldMsg.Serialize())
	if err != nil {
		log.Printf("Failed to send bitfield: %v", err)
		return false
	}
	return true
}

func (t *Torrent) servePeer(rw io.ReadWriter, file *os.File) {
	interested := false
	for {
		msg, err := message.Read(rw)
		if err != nil {
			if err == io.EOF {
				log.Printf("Peer closed connection (EOF)")
				return
			}
			log.Printf("Read error from peer: %v", err)
			return
		}
		if msg == nil {
			continue // keep-alive
		}
		t.handlePeerMessage(msg, rw, file, &interested)
	}
}

func (t *Torrent) handlePeerMessage(msg *message.Message, rw io.ReadWriter, file *os.File, interested *bool) {
	switch msg.ID {
	case message.MsgInterested:
		t.handleInterested(rw, interested)
	case message.MsgRequest:
		t.handleRequest(msg, rw, file, *interested)
	}
}

func (t *Torrent) handleInterested(rw io.ReadWriter, interested *bool) {
	*interested = true
	unchoke := &message.Message{ID: message.MsgUnchoke}
	_, err := rw.Write(unchoke.Serialize())
	if err != nil {
		log.Printf("Failed to send unchoke: %v", err)
	}
}

func (t *Torrent) handleRequest(msg *message.Message, rw io.ReadWriter, file *os.File, interested bool) {
	if !interested {
		log.Printf("Received request from uninterested peer")
		return
	}
	if len(msg.Payload) != 12 {
		log.Printf("Received request with invalid payload length: %d", len(msg.Payload))
		return
	}
	index := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	length := int(binary.BigEndian.Uint32(msg.Payload[8:12]))
	if index < 0 || index >= len(t.PieceHashes) {
		log.Printf("Received request for invalid piece index: %d", index)
		return
	}
	pieceBegin := index * t.PieceLength
	pieceEnd := pieceBegin + t.PieceLength
	if pieceEnd > t.Length {
		pieceEnd = t.Length
	}
	if begin+length > pieceEnd-pieceBegin {
		log.Printf("Received request with invalid begin/length: begin=%d, length=%d, piece size=%d", begin, length, pieceEnd-pieceBegin)
		return
	}
	buf := make([]byte, length)
	_, err := file.ReadAt(buf, int64(pieceBegin+begin))
	if err != nil {
		log.Printf("Failed to read from file: %v", err)
		return
	}
	// Optionally verify hash
	if begin == 0 && length == pieceEnd-pieceBegin {
		h := sha1.Sum(buf)
		if h != t.PieceHashes[index] {
			log.Printf("Hash mismatch for piece %d", index)
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
		log.Printf("Failed to send piece: %v", err)
	}
	// Update seeding status
	if t.SeedingStatus != nil {
		t.SeedingStatus.IncrementSeededBytes(int64(len(buf)))
	}
}
