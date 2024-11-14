package main

import (
	"client/torrent"
	"client/tracker"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("../exampletorrents/rdr.torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	announces, infoHash, err := torrent.DecodeTorrent(file)
	if err != nil {
		fmt.Println("Error decoding torrent file:", err)
	}

	fmt.Println("Info Hash: ", hex.EncodeToString(infoHash[:]))

	fmt.Println("Announcing trackers:", announces)

	for _, url := range announces {
		strURL := url[0]

		fmt.Println("URL: ", strURL)
		if strings.HasPrefix(strURL, "http:") {
			peers, err := tracker.SendAnnounceHTTP(strURL, string(infoHash[:]))

			if err != nil {
				fmt.Println("Error GETting ", strURL, ": ", err)
			} else {
				fmt.Println("Peers from: ", strURL, ": ", peers)
			}
		} else if strings.HasPrefix(strURL, "udp:") {
			endOfURL := strings.LastIndex(strURL, "/")
			var udpAddr string
			if endOfURL > 6 {
				udpAddr = strURL[6:endOfURL]
			} else {
				udpAddr = strURL[6:]
			}
			peers, err := tracker.SendUDPRequest(udpAddr, infoHash)

			if err != nil {
				fmt.Println("UDP Error from", udpAddr, ": ", err)
			} else {
				fmt.Println("Peers from", strURL, ": ", peers)
			}
		}
	}
}
