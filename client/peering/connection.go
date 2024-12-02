package peering

import (
	"bytes"
	"client/tracker"
	"encoding/binary"
	"fmt"
	"net"
)

// type PeerHandshakeMessage struct {
// 	pstrlen uint8
// 	// Protocol String here in length pstrlen
// 	reserved  [8]byte
// 	info_hash [20]byte
// 	peer_id   [20]byte
// }

func PeerHandshake(peer tracker.Peer, info_hash [20]byte) error {
	tcpAddr := &net.TCPAddr{
		IP:   net.IPv4(peer.IP[0], peer.IP[1], peer.IP[2], peer.IP[3]),
		Port: int(peer.Port),
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Println("Connected to", tcpAddr)

	protocolString := []byte("BitTorrent protocol") // string to determine Bittorrent protocol v1.0
	peerId := tracker.GetPeerID()

	// Set up the handshake message
	message := bytes.NewBuffer(make([]byte, 0, 49+len(protocolString)))
	binary.Write(message, binary.BigEndian, uint8(len(protocolString))) // pstrlen
	binary.Write(message, binary.BigEndian, protocolString)             // pstr
	binary.Write(message, binary.BigEndian, [8]byte{0})                 // reserved bytes
	binary.Write(message, binary.BigEndian, info_hash)                  // info hash of the torrent
	binary.Write(message, binary.BigEndian, peerId)                     // this client's peer ID

	_, err = conn.Write(message.Bytes())
	if err != nil {
		fmt.Println("Error sending data:", err)
		return err
	}
	fmt.Println("Message sent to client:", message.Bytes())

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
