package torrentfile

import (
	"bytes"
	"client/peer"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"time"
)

type connectRequestUDP struct {
	Magic         uint64
	Action        uint32
	TransactionID uint32
}
type connectResponseUDP struct {
	Action        uint32
	TransactionID uint32
	ConnectionID  uint64
}

type announceRequestUDP struct {
	ConnectionID  uint64   // Recieved from the connect request
	Action        uint32   // Announce action (1)
	TransactionID uint32   // Randomally generated
	InfoHash      [20]byte // SHA1 hash of the info entry in the torrent
	PeerID        [20]byte // This client's ID (configured at start)
	Downloaded    uint64   // Bytes downloaded from the torrnet so far
	Left          uint64   // Total bytes left to download from the torrent
	Uploaded      uint64   // Bytes uploaded to this torrent
	Event         uint32   // Event enum (none, completed, started, stopped)
	IPAddress     [4]byte  // This client's IP address converted to 4 bytes
	Key           uint32   // Key to signify this client from others, in case of ip change
	NumWant       uint32   // How many peers to get
	Port          uint16   // This client's port to listen on during the Bittorrent transfer
}
type announceResponseHeaderUDP struct {
	Action        uint32
	TransactionID uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	// ... The IP addresses and ports are the next part of the response
}

func newConnectRequestUDP() connectRequestUDP {
	return connectRequestUDP{Magic: 0x41727101980, Action: 0, TransactionID: rand.Uint32()}
}

func newAnnounceRequestUDP(connectionID uint64, infoHash [20]byte, peerID [20]byte,
	downloaded uint64, left uint64, uploaded uint64, event uint32,
	ipAddress [4]byte, port uint16, numWant uint32) announceRequestUDP {
	return announceRequestUDP{
		ConnectionID: connectionID, InfoHash: infoHash, PeerID: peerID,
		Downloaded: downloaded, Left: left, Uploaded: uploaded, Event: event,
		IPAddress: ipAddress, Port: port, NumWant: numWant, Key: rand.Uint32(),
		TransactionID: rand.Uint32(), Action: 1}
}

func sendConnectUDP(conn *net.UDPConn) (connectionID uint64, err error) {
	reqObject := newConnectRequestUDP()
	err = binary.Write(conn, binary.BigEndian, reqObject)
	if err != nil {
		return
	}
	log.Println("Sent out connect request: ", reqObject)

	resObject := new(connectResponseUDP)
	err = binary.Read(conn, binary.BigEndian, resObject)
	if err != nil {
		return
	}
	log.Println("Got connect response: ", resObject)

	if resObject.TransactionID != reqObject.TransactionID || resObject.Action != 0 {
		err = errors.New("invalid response from tracker")
		return
	}
	connectionID = resObject.ConnectionID
	return
}

func sendAnnounceUDP(conn *net.UDPConn, connectionID uint64, infoHash *[20]byte,
	port uint16, peerID *[20]byte) (peers []peer.Peer, err error) {
	const BUFF_SIZE = 1024
	const HEADER_LENGTH = 20
	const PEERS_RETURNED = (BUFF_SIZE - HEADER_LENGTH) / 6

	req := newAnnounceRequestUDP(connectionID, *infoHash, *peerID, 0, 1000, 0, 2,
		[4]byte{0}, port, PEERS_RETURNED)

	err = binary.Write(conn, binary.BigEndian, req)
	if err != nil {
		return
	}
	log.Println("Sent out announce request: ", req)

	resBytes := bytes.NewBuffer(make([]byte, BUFF_SIZE))
	bytesRead, err := conn.Read(resBytes.Bytes())
	if err != nil || bytesRead == HEADER_LENGTH {
		return
	}
	log.Printf("Recieved announce response: %v", resBytes.Bytes())
	if bytesRead < HEADER_LENGTH {
		err = fmt.Errorf("unexpected response length of announce response - %v < %v", bytesRead, HEADER_LENGTH)
		return
	}

	resObject := new(announceResponseHeaderUDP)
	binary.Read(resBytes, binary.BigEndian, resObject)

	if resObject.Action != 1 || resObject.TransactionID != req.TransactionID {
		err = errors.New("error getting announce response")
		return
	}

	peers = make([]peer.Peer, (bytesRead-HEADER_LENGTH)/6)
	err = binary.Read(resBytes, binary.BigEndian, &peers)
	return
}

func (t *TorrentFile) requestPeersUDP(port uint16, peerID *[20]byte,
	announce string) (peers []peer.Peer, err error) {
	var announceAddress string
	if len(announce) > len("udp://") {
		announceAddress = announce[len("udp://"):]
	} else {
		announceAddress = announce
	}
	raddr, err := net.ResolveUDPAddr("udp", announceAddress)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return
	}
	log.Printf("Dialed to %v", raddr)
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return
	}
	connectionID, err := sendConnectUDP(conn)
	if err != nil {
		return
	}
	peers, err = sendAnnounceUDP(conn, connectionID, &t.InfoHash, port, peerID)
	if err != nil {
		return
	}
	return
}
