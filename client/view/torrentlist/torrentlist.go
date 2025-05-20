package torrentlist

import (
	"client/torrent"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type TorrentList struct {
	Widgets  *widget.List
	Grid     *fyne.Container
	Progress *widget.ProgressBar
	Torrents []*torrent.Torrent
	Selected *torrent.Torrent
	mu       sync.RWMutex
	quit     chan struct{}
	// Store labels for updating
	labels struct {
		Name       *widget.Label
		Length     *widget.Label
		Pieces     *widget.Label
		Status     *widget.Label
		Peers      *widget.Label
		Downloaded *widget.Label
	}
}

func New() *TorrentList {
	tl := &TorrentList{
		quit: make(chan struct{}),
	}

	// Initialize the list widget
	tl.Widgets = widget.NewList(
		func() int { return len(tl.Torrents) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(tl.Torrents[i].Name)
		},
	)

	// Create progress bar
	tl.Progress = widget.NewProgressBar()
	tl.Progress.Hide()

	tl.Widgets.OnSelected = func(id widget.ListItemID) {
		select {
		case <-tl.quit:
			// Channel already closed, do nothing
		default:
			close(tl.quit)
		}
		tl.quit = make(chan struct{})
		ticker := time.NewTicker(500 * time.Millisecond)
		tl.Selected = tl.Torrents[id]
		tl.Progress.Show()
		tl.updateLabels()
		go func() {
			for {
				select {
				case <-ticker.C:
					fyne.Do(tl.updateLabels)
				case <-tl.quit:
					ticker.Stop()
					return
				}
			}
		}()
	}

	tl.Widgets.OnUnselected = func(id widget.ListItemID) {
		select {
		case <-tl.quit:
			// Channel already closed, do nothing
		default:
			close(tl.quit)
		}
		tl.Selected = nil
		tl.Progress.Hide()
		fyne.Do(tl.updateLabels)
	}

	// Create initial grid
	tl.updateGrid()

	return tl
}

func (tl *TorrentList) updateGrid() {
	if tl.Grid != nil {
		tl.Grid.Refresh()
	}

	// Create header labels
	headers := []string{"Name", "Length", "Pieces", "Status", "Peers", "Downloaded"}
	var allLabels []fyne.CanvasObject

	// For each field
	for _, header := range headers {
		// Add header label
		headerLabel := widget.NewLabelWithStyle(header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		allLabels = append(allLabels, headerLabel)

		// Create value label
		valueLabel := widget.NewLabel("No torrent selected")
		switch header {
		case "Name":
			tl.labels.Name = valueLabel
		case "Length":
			tl.labels.Length = valueLabel
		case "Pieces":
			tl.labels.Pieces = valueLabel
		case "Status":
			tl.labels.Status = valueLabel
		case "Peers":
			tl.labels.Peers = valueLabel
		case "Downloaded":
			tl.labels.Downloaded = valueLabel
		}
		allLabels = append(allLabels, valueLabel)
	}

	// Create grid with 2 columns (field name and value)
	tl.Grid = container.NewGridWithColumns(2, allLabels...)
	tl.Grid.Refresh()
}

func (tl *TorrentList) updateLabels() {
	if tl.Selected == nil {
		tl.labels.Name.SetText("No torrent selected")
		tl.labels.Length.SetText("No torrent selected")
		tl.labels.Pieces.SetText("No torrent selected")
		tl.labels.Status.SetText("No torrent selected")
		tl.labels.Peers.SetText("No torrent selected")
		tl.labels.Downloaded.SetText("No torrent selected")
		tl.Grid.Refresh()
		return
	}

	// Update each label based on the selected torrent
	tl.labels.Name.SetText(tl.Selected.Name)
	tl.labels.Length.SetText(fmt.Sprintf("%d bytes", tl.Selected.Length))
	tl.labels.Pieces.SetText(fmt.Sprintf("%d", len(tl.Selected.PieceHashes)))

	// Status
	if tl.Selected.DownloadStatus == nil {
		tl.labels.Status.SetText("Not Started")
		tl.labels.Status.Importance = widget.MediumImportance
	} else if tl.Selected.Paused {
		tl.labels.Status.SetText("⏸️ PAUSED")
		tl.labels.Status.Importance = widget.HighImportance
	} else {
		tl.labels.Status.SetText("▶️ DOWNLOADING")
		tl.labels.Status.Importance = widget.MediumImportance
	}

	// Peers
	if tl.Selected.DownloadStatus == nil {
		tl.labels.Peers.SetText("0")
	} else {
		tl.labels.Peers.SetText(fmt.Sprintf("%d", tl.Selected.DownloadStatus.PeersAmount))
	}

	// Downloaded
	if tl.Selected.DownloadStatus == nil {
		tl.labels.Downloaded.SetText("0")
	} else {
		tl.labels.Downloaded.SetText(fmt.Sprintf("%d/%d", tl.Selected.DownloadStatus.DonePieces, len(tl.Selected.PieceHashes)))
	}

	// Update progress bar
	if tl.Selected.DownloadStatus != nil {
		tl.Progress.SetValue(tl.Selected.CalculateDownloadPercentage() / 100.0)
	} else {
		tl.Progress.SetValue(0)
	}

	tl.Grid.Refresh()
}

func (tl *TorrentList) AddTorrent(t *torrent.Torrent) {
	tl.mu.Lock()
	tl.Torrents = append(tl.Torrents, t)
	tl.mu.Unlock()
	fyne.Do(tl.Widgets.Refresh)
}

func (tl *TorrentList) ForceUpdateDetails() {
	fyne.Do(tl.updateLabels)
}

// TODO:
// 1. pause button
// 2. notification after download finished
// 3. seeding
