package connection

import (
	"bytes"
	"client/bitfield"
	"client/handshake"
	"client/message"
	"client/peer"
	"fmt"
	"net"
	"time"
)

// Connection represents a client connection.
type Connection struct {
	Conn     net.Conn
	Choked   bool
	Bitfield bitfield.Bitfield
	peer     peer.Peer
	infoHash [20]byte
	peerID   [20]byte
}

func completeHandshake(conn net.Conn, infohash, peerID [20]byte) (*handshake.Handshake, error) {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	req := handshake.New(infohash, peerID)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := handshake.Read(conn)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(res.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", res.InfoHash, infohash)
	}
	return res, nil
}

func recvBitfield(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	msg, err := message.Read(conn)
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

func New(peer peer.Peer, peerID, infoHash [20]byte) (*Connection, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}

	bf, err := recvBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Connection{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

func (c *Connection) Read() (*message.Message, error) {
	return message.Read(c.Conn)
}

func (c *Connection) SendUnchoke() error {
	msg := message.Message{ID: message.MsgUnchoke}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendRequest(index, begin, length int) error {
	msg := message.FormatRequest(index, begin, length)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendHave(index int) error {
	msg := message.FormatHave(index)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}
