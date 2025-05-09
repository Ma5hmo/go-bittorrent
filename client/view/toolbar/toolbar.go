package toolbar

import (
	"client/common"
	"client/torrent"
	"client/torrentfile"
	"client/view/torrentlist"
	"client/view/viewutils"
	"client/viewmodel"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func New(tl *torrentlist.TorrentList) *widget.Toolbar {
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
	// 	viewutils.ShowMessage("Error opening file:\n" + err.Error())
	// 	return
	// }
	// if reader == nil {
	// 	viewutils.ShowMessage("No file selected")
	// 	return
	// }
	// defer reader.Close()
	// reader.URI().Path()

	tf, err := torrentfile.Open("../exampletorrents/debian.torrent")
	if err != nil {
		viewutils.ShowMessage("Error parsing torrent:\n" + err.Error())
		return
	}
	// }, viewutils.MainWindow).Show()

	go func() {
		t, err := torrent.New(&tf, &common.AppState.PeerID, common.AppState.Port)
		if err != nil {
			viewutils.ShowMessage("Failed to create torrent:\n" + err.Error())
			return
		}

		tl.AddTorrent(t)
	}()
}

func startTorrent(tl *torrentlist.TorrentList) {
	var fileOutput *os.File
	if tl.Selected == nil {
		viewutils.ShowMessage("No torrent is selected")
		return
	}

	onSave := func(u fyne.URIWriteCloser, err error) {
		if err != nil { // a filesystem error
			dialog.ShowError(err, viewutils.MainWindow)
			return
		}
		if u == nil { // user pressed “Cancel”
			return
		}
		path := u.URI().Path()
		u.Close()
		log.Printf("%s", path)
		fileOutput, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			dialog.ShowError(err, viewutils.MainWindow)
			return
		}
		go viewmodel.StartTorrent(tl.Selected, fileOutput)
	}

	dlg := dialog.NewFileSave(onSave, viewutils.MainWindow)
	dlg.SetFileName(tl.Selected.Name) // default name
	dlg.Show()                        // present modal
}

func stopTorrent(tl *torrentlist.TorrentList) {
	viewutils.ShowMessage("Stop Torrent")
}
