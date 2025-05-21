package connection

import (
	"bytes"
	"client/bitfield"
	"client/handshake"
	"client/message"
	"client/peer"
	"fmt"
	"io"
	"net"
	"time"
)

// Connection represents a client connection.
type Connection struct {
	Conn     net.Conn      // Underlying TCP connection
	EncConn  io.ReadWriter // Encrypted connection for protocol communication
	Choked   bool
	Bitfield bitfield.Bitfield
	peer     peer.Peer
	infoHash *[20]byte
	peerID   *[20]byte
}

func completeHandshake(rw io.ReadWriter, infohash, peerID *[20]byte) (*handshake.Handshake, error) {
	// Use ReadWriter for handshake
	req := handshake.New(infohash, peerID)
	_, err := rw.Write(req.Serialize())
	if err != nil {
		return nil, err
	}
	// Read handshake response
	resp, err := handshake.Read(rw)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(resp.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", resp.InfoHash, infohash)
	}
	return resp, nil
}

func recvBitfield(rw io.ReadWriter) (bitfield.Bitfield, error) {
	msg, err := message.Read(rw)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		err := fmt.Errorf("expected bitfield but got %v", msg)
		return nil, err
	}
	if msg.ID != message.MsgBitfield {
		err := fmt.Errorf("expected bitfield but got ID %d", msg.ID)
		return nil, err
	}
	return msg.Payload, nil
}

func New(peer peer.Peer, peerID *[20]byte, infoHash *[20]byte) (*Connection, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	// --- Encryption handshake: send key/iv ---
	key, iv, err := GenerateRandomKeyIV()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Write(key); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err := conn.Write(iv); err != nil {
		conn.Close()
		return nil, err
	}
	encConn, err := WrapConnWithAES(conn, key, iv)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// Use bufRW for handshake and bitfield
	if _, err = completeHandshake(encConn, infoHash, peerID); err != nil {
		conn.Close()
		return nil, err
	}

	bf, err := recvBitfield(encConn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Connection{
		Conn:     conn,
		EncConn:  encConn,
		Choked:   true,
		Bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

func (c *Connection) Read() (*message.Message, error) {
	return message.Read(c.EncConn)
}

func (c *Connection) SendUnchoke() error {
	msg := message.Message{ID: message.MsgUnchoke}
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendRequest(index, begin, length int) error {
	msg := message.FormatRequest(index, begin, length)
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendHave(index int) error {
	msg := message.FormatHave(index)
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}
