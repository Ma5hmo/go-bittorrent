package view

import (
	"client/common"
	"client/torrent"
	"client/torrentfile"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func createMainToolbar() *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.FileIcon(), onClickOpenFile),
		widget.NewToolbarAction(theme.MediaPlayIcon(), onClickStartTorrent),
		widget.NewToolbarAction(theme.MediaPauseIcon(), onClickStopTorrent),
	)
}

func onClickOpenFile() {
	// Add torrent action
	// dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
	// if err != nil {
	// 	showMessage("Error opening file:\n" + err.Error())
	// 	return
	// }
	// if reader == nil {
	// 	showMessage("No file selected")
	// 	return
	// }
	// defer reader.Close()
	// reader.URI().Path()
	tf, err := torrentfile.Open("../exampletorrents/debian.torrent")
	if err != nil {
		showMessage("Error parsing torrent:\n" + err.Error())
		return
	}

	t, err := torrent.New(&tf, &common.AppState.PeerID, common.AppState.Port)
	if err != nil {
		showMessage("Failed to create torrent:\n" + err.Error())
		return
	}

	torrentListView.torrents = append(torrentListView.torrents, *t)
	torrentListView.list.Refresh()
	// }, mainWindow).Show()
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

func onClickStartTorrent() {
	go func() {
		if torrentListView.selected == nil {
			return
		}
		torrentListView.selected.Download()
	}()
}

func onClickStopTorrent() {
	showMessage("Stop Torrent")
}
