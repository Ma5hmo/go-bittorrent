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
	dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			viewutils.ShowMessage("Error opening file:\n" + err.Error())
			return
		}
		if reader == nil {
			viewutils.ShowMessage("No file selected")
			return
		}
		path := reader.URI().Path()
		reader.Close()
		tf, err := torrentfile.Open(path) //"../exampletorrents/debian.torrent")
		// use reader.URI().Path() to open a torrentfile
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
	}, viewutils.MainWindow).Show()
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
	if tb.torrentList.Grid.Selected.Path != "" && !tb.torrentList.Grid.Selected.IsSeedingPaused {
		tb.torrentList.Grid.Selected.ResumeDownload()
		tb.torrentList.ForceUpdateDetails()
		go viewmodel.StartTorrent(tb.torrentList.Grid.Selected, nil)
		return
	}
	// If this is a seeding torrent and is paused, start seeding
	if tb.torrentList.Grid.Selected.IsSeedingPaused {
		tb.torrentList.Grid.Selected.IsSeedingPaused = false
		go tb.torrentList.Grid.Selected.StartSeeder()
		tb.torrentList.ForceUpdateDetails()
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
	// Buttons to open file dialogs
	torrentBtn := widget.NewButton("Choose .torrent", nil)
	torrentBtn.OnTapped = func() {
		dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			torrentBtn.SetText(reader.URI().Path())
			reader.Close()
		}, viewutils.MainWindow)
		dlg.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
		dlg.Show()
	}
	fileBtn := widget.NewButton("Choose File", nil)
	fileBtn.OnTapped = func() {
		dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			fileBtn.SetText(reader.URI().Path())
			reader.Close()
		}, viewutils.MainWindow)
		dlg.Show()
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Torrent file", Widget: torrentBtn},
			{Text: "File to seed", Widget: fileBtn},
		},
		OnSubmit: func() {
			torrentPath := torrentBtn.Text
			filePath := fileBtn.Text
			if torrentPath == "Choose .torrent" || filePath == "Choose File" {
				viewutils.ShowMessage("Please select both a .torrent file and a file to seed.")
				return
			}
			tf, err := torrentfile.Open(torrentPath)
			if err != nil {
				viewutils.ShowMessage("Error parsing torrent: " + err.Error())
				return
			}
			t, err := torrent.New(&tf, &common.AppState.PeerID, common.AppState.Port)
			t.Path = filePath // Save the actual file path
			if err != nil {
				viewutils.ShowMessage("Failed to create torrent: " + err.Error())
				return
			}
			t.IsSeedingPaused = true // Mark as paused for seeding
			tb.seedingList.AddTorrent(t)
			// Do NOT start seeding here; wait for resume
			viewutils.ShowMessage("Torrent added to seeding list. Press Resume to start seeding.")
		},
	}
	form.Resize(fyne.NewSize(800, 430))
	dlg := dialog.NewCustom("Seed Torrent", "Close", form, viewutils.MainWindow)
	dlg.Resize(fyne.NewSize(1000, 430))
	dlg.Show()
}
