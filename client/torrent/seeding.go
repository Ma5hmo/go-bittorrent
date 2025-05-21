package torrent

import (
	"client/connection"
	"client/handshake"
	"client/message"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

// StartSeeder listens for incoming connections and seeds the torrent file to peers.
func (t *Torrent) StartSeeder() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", t.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", t.Port, err)
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
	index := int(uint32(msg.Payload[0])<<24 | uint32(msg.Payload[1])<<16 | uint32(msg.Payload[2])<<8 | uint32(msg.Payload[3]))
	begin := int(uint32(msg.Payload[4])<<24 | uint32(msg.Payload[5])<<16 | uint32(msg.Payload[6])<<8 | uint32(msg.Payload[7]))
	length := int(uint32(msg.Payload[8])<<24 | uint32(msg.Payload[9])<<16 | uint32(msg.Payload[10])<<8 | uint32(msg.Payload[11]))
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
	// index
	payload[0] = byte(index >> 24)
	payload[1] = byte(index >> 16)
	payload[2] = byte(index >> 8)
	payload[3] = byte(index)
	// begin
	payload[4] = byte(begin >> 24)
	payload[5] = byte(begin >> 16)
	payload[6] = byte(begin >> 8)
	payload[7] = byte(begin)
	copy(payload[8:], buf)
	pieceMsg := &message.Message{ID: message.MsgPiece, Payload: payload}
	_, err = rw.Write(pieceMsg.Serialize())
	if err != nil {
		log.Printf("Failed to send piece: %v", err)
	}
}
