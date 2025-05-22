package toolbar

import (
	"client/common"
	"client/torrent"
	"client/torrentfile"
	"client/view/torrentcreate"
	"client/view/torrentlist"
	"client/view/viewutils"
	"client/viewmodel"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	torrentList *torrentlist.TorrentList
	seedingList *torrentlist.TorrentList // Add reference to seeding list
	widget      *widget.Toolbar
}

func New(tl *torrentlist.TorrentList, seedingList *torrentlist.TorrentList) *widget.Toolbar {
	tb := &Toolbar{
		torrentList: tl,
		seedingList: seedingList,
	}

	tb.widget = widget.NewToolbar(
		widget.NewToolbarAction(theme.FileIcon(), tb.handleOpenTorrent),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), torrentcreate.HandleCreateTorrent), // Add create torrent button
		widget.NewToolbarAction(theme.DownloadIcon(), tb.handleStartTorrent),
		widget.NewToolbarAction(theme.MediaPlayIcon(), tb.handleResumeTorrent),
		widget.NewToolbarAction(theme.MediaPauseIcon(), tb.handleStopTorrent),
		widget.NewToolbarAction(theme.UploadIcon(), tb.handleSeedTorrent), // Add seed button
	)

	return tb.widget
}

func (tb *Toolbar) handleOpenTorrent() {
	// dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
	// if err != nil {
	//  viewutils.ShowMessage("Error opening file:\n" + err.Error())
	//  return
	// }
	// if reader == nil {
	//  viewutils.ShowMessage("No file selected")
	//  return
	// }
	// defer reader.Close()
	// // use reader.URI().Path() to open a torrentfile
	// )}
	tf, err := torrentfile.Open("../exampletorrents/debian.torrent")
	if err != nil {
		viewutils.ShowMessage("Error parsing torrent:\n" + err.Error())
		return
	}
	go func() {
		t, err := torrent.New(&tf, &common.AppState.PeerID, common.AppState.Port)
		if err != nil {
			viewutils.ShowMessage("Failed to create torrent:\n" + err.Error())
			return
		}

		tb.torrentList.AddTorrent(t)
	}()
}

func (tb *Toolbar) handleStartTorrent() {
	if tb.torrentList.Grid.Selected == nil {
		viewutils.ShowMessage("No torrent is selected")
		return
	}
	if tb.torrentList.Grid.Selected.Path != "" {
		viewutils.ShowMessage("Torrent had already started, resume it instead!")
		return
	}

	dlg := dialog.NewFileSave(tb.dialogFileSaveHandler, viewutils.MainWindow)
	dlg.SetFileName(tb.torrentList.Grid.Selected.Name)
	dlg.Show()
}

func (tb *Toolbar) handleResumeTorrent() {
	if tb.torrentList.Grid.Selected == nil {
		viewutils.ShowMessage("No torrent is selected")
		return
	}
	if tb.torrentList.Grid.Selected.Path != "" {
		tb.torrentList.Grid.Selected.ResumeDownload()
		tb.torrentList.ForceUpdateDetails()
		go viewmodel.StartTorrent(tb.torrentList.Grid.Selected, nil)
		return
	}
	dlg := dialog.NewFileOpen(tb.dialogFileOpenHandler, viewutils.MainWindow)
	dlg.SetFileName(tb.torrentList.Grid.Selected.Name)
	dlg.Show()
}

func (tb *Toolbar) openAndStartTorrent(path string, createIfNotExists bool) {
	flags := os.O_RDWR
	if createIfNotExists {
		flags |= os.O_CREATE
	}

	fileOutput, err := os.OpenFile(path, flags, 0666)
	if err != nil {
		dialog.ShowError(err, viewutils.MainWindow)
		return
	}
	go viewmodel.StartTorrent(tb.torrentList.Grid.Selected, fileOutput)
}

func (tb *Toolbar) dialogFileSaveHandler(u fyne.URIWriteCloser, err error) {
	if err != nil { // a filesystem error
		dialog.ShowError(err, viewutils.MainWindow)
		return
	}
	if u == nil { // user pressed "Cancel"
		return
	}
	path := u.URI().Path()
	u.Close()
	tb.openAndStartTorrent(path, true)
}

func (tb *Toolbar) dialogFileOpenHandler(u fyne.URIReadCloser, err error) {
	if err != nil { // a filesystem error
		dialog.ShowError(err, viewutils.MainWindow)
		return
	}
	if u == nil { // user pressed "Cancel"
		return
	}
	path := u.URI().Path()
	u.Close()
	tb.openAndStartTorrent(path, false)
}

func (tb *Toolbar) handleStopTorrent() {
	if tb.torrentList.Grid.Selected != nil {
		tb.torrentList.Grid.Selected.PauseDownload()
		tb.torrentList.ForceUpdateDetails()
	}
}

func (tb *Toolbar) handleSeedTorrent() {
	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			viewutils.ShowMessage("Error selecting file to seed: " + err.Error())
			return
		}
		if reader == nil {
			return
		}
		filePath := reader.URI().Path()
		reader.Close()

		// Open the torrent file and create a Torrent struct
		tf, err := torrentfile.Open(filePath)
		if err != nil {
			viewutils.ShowMessage("Error parsing torrent: " + err.Error())
			return
		}
		t, err := torrent.New(&tf, &common.AppState.PeerID, common.AppState.Port) // Use default PeerID/Port for seeding
		if err != nil {
			viewutils.ShowMessage("Failed to create torrent: " + err.Error())
			return
		}

		tb.seedingList.AddTorrent(t)

		// Start seeding in background
		go t.StartSeeder()
	}, viewutils.MainWindow)
	dlg.Show()
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
}
