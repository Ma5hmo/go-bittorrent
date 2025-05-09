package viewmodel

import (
	"client/torrent"
	"log"
	"os"
)

func StartTorrent(t *torrent.Torrent, fileOutput *os.File) {
	defer fileOutput.Close()
	err := t.Download(fileOutput)
	if err != nil {
		log.Printf("error - %s", err.Error())
	}

	// update it to being done or smth
}
