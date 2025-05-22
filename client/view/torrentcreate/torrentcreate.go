package torrentcreate

import (
	"client/torrentfile"
	"client/view/viewutils"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func HandleCreateTorrent() {
	var filePath string
	var announce string
	var torrentName string
	var description string
	var pieceLengthStr string
	chooseFileButton := widget.NewButton("Choose File", nil)
	form := widget.NewForm(
		widget.NewFormItem("File to Share", chooseFileButton),
		widget.NewFormItem("Announce URL", widget.NewEntry()),
		widget.NewFormItem("Torrent Name", widget.NewEntry()),
		widget.NewFormItem("Description", widget.NewEntry()),
		widget.NewFormItem("Piece Length (KB)", widget.NewEntry()),
	)

	// Set default value for piece length
	if entry, ok := form.Items[4].Widget.(*widget.Entry); ok {
		entry.SetText("256")
	}
	// Set default value for the announce URL
	if entry, ok := form.Items[1].Widget.(*widget.Entry); ok {
		entry.SetText("http://localhost:8080/announce")
	}

	chooseFileButton.OnTapped = func() {
		dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				viewutils.ShowMessage("Error selecting file: " + err.Error())
				return
			}
			if reader == nil {
				return
			}
			filePath = reader.URI().Path()
			torrentName = reader.URI().Name()
			reader.Close()
			if entry, ok := form.Items[2].Widget.(*widget.Entry); ok {
				entry.SetText(torrentName)
			}
			chooseFileButton.SetText(filePath)
		}, viewutils.MainWindow).Show()
	}

	form.OnSubmit = func() {
		announce = form.Items[1].Widget.(*widget.Entry).Text
		torrentName = form.Items[2].Widget.(*widget.Entry).Text
		description = form.Items[3].Widget.(*widget.Entry).Text
		pieceLengthStr = form.Items[4].Widget.(*widget.Entry).Text

		if filePath == "" || announce == "" || torrentName == "" || pieceLengthStr == "" {
			viewutils.ShowMessage("Please fill all required fields and select a file.")
			return
		}
		// Check announce URL scheme
		if !(len(announce) > 6 && (announce[:6] == "udp://")) &&
			!(len(announce) > 7 && (announce[:7] == "http://")) &&
			!(len(announce) > 8 && (announce[:8] == "https://")) {
			viewutils.ShowMessage("Announce URL must start with udp://, http://, or https://")
			return
		}
		pieceLengthKB, err := strconv.Atoi(pieceLengthStr)
		if err != nil || pieceLengthKB <= 0 {
			viewutils.ShowMessage("Invalid piece length. Please enter a positive integer.")
			return
		}
		pieceLength := pieceLengthKB * 1024

		dlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				viewutils.ShowMessage("Error saving torrent: " + err.Error())
				return
			}
			if writer == nil {
				return
			}
			torrentPath := writer.URI().Path()
			go func() {
				err := createAndSaveTorrent(filePath, announce, torrentName, description, pieceLength, torrentPath)
				if err != nil {
					viewutils.ShowMessage("Failed to create torrent: " + err.Error())
				} else {
					viewutils.ShowMessage("Torrent file created successfully!\n" + torrentPath)
				}
				writer.Close()
			}()
		}, viewutils.MainWindow)
		dlg.SetFileName(torrentName + ".torrent")
		dlg.Show()
	}

	w := fyne.CurrentApp().NewWindow("Create Torrent")
	w.SetContent(container.NewVBox(
		widget.NewLabel("Create a new Torrent File"),
		form,
	))
	w.Resize(fyne.NewSize(400, 350))
	w.Show()
}

// createAndSaveTorrent creates and saves a .torrent file with metadata and custom piece length using TorrentFile struct
func createAndSaveTorrent(filePath, announce, torrentName, description string, pieceLength int, torrentPath string) error {
	// Build TorrentFile struct
	tf, err := torrentfile.CreateFromFile(filePath, announce, torrentName, description, pieceLength)
	if err != nil {
		return err
	}
	return tf.SaveToFile(torrentPath)
}
