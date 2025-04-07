package peering

import (
	"fmt"
	"io"
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

func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	pstrlen := int(lengthBuf[0])

	if pstrlen == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peerID [20]byte

	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peerID[:], handshakeBuf[pstrlen+8+20:])

	h := Handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	return &h, nil
}

// func (h *Handshake) Send(peer peering.Peer) error {

// 	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
// 	if err != nil {
// 		return err
// 	}
// 	defer conn.Close()
// 	fmt.Println("Connected to", peer)

// 	message := h.Serialize()

// 	_, err = conn.Write(message)
// 	if err != nil {
// 		fmt.Println("Error sending data:", err)
// 		return err
// 	}
// 	// fmt.Println("Message sent to client:", message)

// 	buffer := make([]byte, 1024)
// 	n, err := conn.Read(buffer)
// 	if err != nil {
// 		fmt.Println("Error reading data:", err)
// 		return err
// 	}

// 	fmt.Println("Client responded to handshake:", buffer[:n])
// 	// Start requesting pieces...
// 	return nil
// }
