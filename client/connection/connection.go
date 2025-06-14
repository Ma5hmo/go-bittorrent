package connection

import (
	"bytes"
	"client/bitfield"
	"client/handshake"
	"client/message"
	"client/peer"
	"client/protocolconn"
	"fmt"
	"net"
	"time"
)

// Connection represents a client connection.
type Connection struct {
	Conn     net.Conn                   // Underlying TCP connection
	EncConn  *protocolconn.ProtocolConn // Encrypted connection for protocol communication
	Choked   bool
	Bitfield bitfield.Bitfield
	peer     peer.Peer
	infoHash *[20]byte
	peerID   *[20]byte
}

func completeHandshake(rw *protocolconn.ProtocolConn, infohash, peerID *[20]byte) (*handshake.Handshake, error) {
	// Use ReadWriter for handshake
	req := handshake.New(infohash, peerID)
	// log.Printf("created handshake - %v", req)
	pstrlen := []byte{byte(len(req.Pstr))}
	_, err := rw.RawReadWriter.Write(pstrlen)
	if err != nil {
		return nil, err
	}
	// log.Printf("sent pstrlen - %v", pstrlen)
	_, err = rw.EncryptedWriter.Write(req.Serialize()[1:]) // Skip pstrlen byte
	if err != nil {
		return nil, err
	}
	// log.Printf("sent rest of handshake - %v", req.Serialize()[1:])

	// Read handshake response
	resp, err := handshake.Read(rw)
	if err != nil {
		return nil, err
	}
	// log.Printf("resp handshake - %v", resp.Serialize())

	if !bytes.Equal(resp.InfoHash[:], infohash[:]) {
		return nil, fmt.Errorf("expected infohash %x but got %x", resp.InfoHash, infohash)
	}
	return resp, nil
}

func recvBitfield(rw *protocolconn.ProtocolConn) (bitfield.Bitfield, error) {
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

func New(peer peer.Peer, peerID *[20]byte, infoHash *[20]byte, encrypted bool) (*Connection, error) {
	// log.Printf("[Connection] Attempting to connect to peer: %s", peer.String())
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		// log.Printf("[Connection] Failed to connect to peer: %s, error: %v", peer.String(), err)
		return nil, err
	}
	var encConn *protocolconn.ProtocolConn = &protocolconn.ProtocolConn{
		EncryptedReader: conn,
		EncryptedWriter: conn,
		RawReadWriter:   conn,
	}
	if encrypted {
		// log.Printf("[Connection] Starting encryption handshake with peer: %s", peer.String())
		// --- Encryption handshake: send key/iv ---
		key, iv, err := protocolconn.GenerateRandomKeyIV()
		if err != nil {
			// log.Printf("[Connection] Failed to generate key/iv: %v", err)
			conn.Close()
			return nil, err
		}
		if _, err := conn.Write(key); err != nil {
			// log.Printf("[Connection] Failed to send key: %v", err)
			conn.Close()
			return nil, err
		}
		if _, err := conn.Write(iv); err != nil {
			// log.Printf("[Connection] Failed to send iv: %v", err)
			conn.Close()
			return nil, err
		}
		encConn, err = protocolconn.New(conn, key, key, iv, iv)
		if err != nil {
			// log.Printf("[Connection] Failed to wrap conn with AES: %v", err)
			conn.Close()
			return nil, err
		}
	}
	// Use bufRW for handshake and bitfield
	// log.Printf("[Connection] Performing handshake with peer: %s", peer.String())
	if _, err = completeHandshake(encConn, infoHash, peerID); err != nil {
		// log.Printf("[Connection] Handshake failed with peer: %s, error: %v", peer.String(), err)
		conn.Close()
		return nil, err
	}

	// log.Printf("[Connection] Receiving bitfield from peer: %s", peer.String())
	bf, err := recvBitfield(encConn)
	if err != nil {
		// log.Printf("[Connection] Failed to receive bitfield from peer: %s, error: %v", peer.String(), err)
		conn.Close()
		return nil, err
	}

	// log.Printf("[Connection] Connection established with peer: %s", peer.String())
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
	// // log.Printf("[Connection] Reading message from peer: %s", c.peer.String())
	return message.Read(c.EncConn)
}

func (c *Connection) SendUnchoke() error {
	// log.Printf("[Connection] Sending UNCHOKE to peer: %s", c.peer.String())
	msg := message.Message{ID: message.MsgUnchoke}
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendInterested() error {
	// log.Printf("[Connection] Sending INTERESTED to peer: %s", c.peer.String())
	msg := message.Message{ID: message.MsgInterested}
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendRequest(index, begin, length int) error {
	// log.Printf("[Connection] Sending REQUEST to peer: %s (index=%d, begin=%d, length=%d)", c.peer.String(), index, begin, length)
	msg := message.FormatRequest(index, begin, length)
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}

func (c *Connection) SendHave(index int) error {
	// log.Printf("[Connection] Sending HAVE to peer: %s (index=%d)", c.peer.String(), index)
	msg := message.FormatHave(index)
	_, err := c.EncConn.Write(msg.Serialize())
	return err
}
