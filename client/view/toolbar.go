package view

import (
	"client/common"
	"client/torrent"
	"client/torrentfile"
	"client/view/torrentlist"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func createMainToolbar(tl *torrentlist.TorrentList) *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.FileIcon(), func() {
			openTorrent(tl)
		}),
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			startTorrent(tl)
		}),
		widget.NewToolbarAction(theme.MediaPauseIcon(), func() {
			stopTorrent(tl)
		}),
	)
}

func openTorrent(tl *torrentlist.TorrentList) {
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
	// }, mainWindow).Show()

	go func() {
		t, err := torrent.New(&tf, &common.AppState.PeerID, common.AppState.Port)
		if err != nil {
			showMessage("Failed to create torrent:\n" + err.Error())
			return
		}

		tl.AddTorrent(t)
	}()
}

func startTorrent(tl *torrentlist.TorrentList) {
	go func() {
		if tl.Selected == nil {
			return
		}
		tl.Selected.Download() // blocking
		// TODO: WRITE TO FILE
	}()
}

func stopTorrent(tl *torrentlist.TorrentList) {
	showMessage("Stop Torrent")
}
