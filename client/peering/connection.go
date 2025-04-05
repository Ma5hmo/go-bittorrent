package peering

import (
	"fmt"
	"net"
	"time"
)

// A Handshake is a special message that a peer uses to identify itself
type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	PeerID   [20]byte
}

// Serialize serializes the handshake to a buffer
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8)) // 8 reserved bytes
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

func (h *Handshake) Send(peer Peer) error {

	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Println("Connected to", peer)

	message := h.Serialize()

	_, err = conn.Write(message)
	if err != nil {
		fmt.Println("Error sending data:", err)
		return err
	}
	// fmt.Println("Message sent to client:", message)

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading data:", err)
		return err
	}

	fmt.Println("Client responded to handshake:", buffer[:n])
	// Start requesting pieces...
	return nil
}
