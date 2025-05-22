package main

import (
	"client/common"
	"client/view"
	"os"
)

func writeToFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data) // convert [20]byte to []byte
	return err
}

// func main() {
// 	tf, err := torrentfile.Open("../exampletorrents/debian.torrent")
// 	if err != nil {
// 		log.Fatalf("opening torrent - %v", err)
// 	}
// 	log.Printf("infohash - %x", tf.InfoHash)
// 	peerId := createPeerId()
// 	port := uint16(6881)
// 	t, err := torrent.New(&tf, peerId, port)
// 	if err != nil {
// 		log.Fatalf("create torrent - %v", err)
// 	}
// 	buf := t.Download()
// 	err = writeToFile("../downloaded.bin", buf)
// 	if err != nil {
// 		log.Fatalf("writing to file - %v", err)
// 	}
// 	log.Printf("END")
// }

func main() {
	common.InitAppState()
	view.CreateMainWindow()
}
