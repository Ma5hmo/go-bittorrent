package viewmodel

import (
	"client/torrent"
	"log"
	"os"
)

func StartTorrent(t *torrent.Torrent, fileOutput *os.File) {
	if fileOutput != nil {
		defer fileOutput.Close()
	}
	err := t.StartDownload(fileOutput)
	if err != nil {
		log.Printf("error starting download - %s", err.Error())
	}

	// update it to being done or smth
}
