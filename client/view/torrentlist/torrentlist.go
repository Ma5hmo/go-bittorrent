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
	Torrents []*torrent.Torrent
	mu       sync.RWMutex
	Grid     *Grid
}

type Grid struct {
	Grid         *fyne.Container
	quit         chan struct{}
	Selected     *torrent.Torrent
	torrentLists []*TorrentList

	Progress *widget.ProgressBar
	// Store labels for updating
	labels struct {
		Name             *widget.Label
		Length           *widget.Label
		Pieces           *widget.Label
		Status           *widget.Label
		Peers            *widget.Label
		DownloadedHeader *widget.Label
		Downloaded       *widget.Label
	}
}

func NewGrid() *Grid {
	g := &Grid{quit: make(chan struct{})}
	g.Progress = widget.NewProgressBar()
	g.Progress.Hide()
	g.updateGrid()
	return g
}

func New(grid *Grid) *TorrentList {
	tl := &TorrentList{}

	// Initialize the list widget
	tl.Widgets = widget.NewList(
		func() int { return len(tl.Torrents) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(tl.Torrents[i].Name)
		},
	)
	tl.Grid = grid
	grid.addTorrentList(tl)

	tl.Widgets.OnSelected = func(id widget.ListItemID) {
		tl.Grid.OnSelect(tl.Torrents[id], tl)
	}

	tl.Widgets.OnUnselected = func(id widget.ListItemID) {
		tl.Grid.OnUnselected()
	}

	// Create initial grid
	tl.Grid.updateGrid()

	return tl
}

func (g *Grid) addTorrentList(tl *TorrentList) {
	g.torrentLists = append(g.torrentLists, tl)
}

func (g *Grid) OnSelect(selected *torrent.Torrent, selectedList *TorrentList) {
	for _, tl := range g.torrentLists {
		if tl != selectedList {
			tl.Widgets.UnselectAll()
		}
	}
	select {
	case <-g.quit:
		// Channel already closed, do nothing
	default:
		close(g.quit)
	}
	g.quit = make(chan struct{})
	ticker := time.NewTicker(500 * time.Millisecond)
	g.Selected = selected
	g.Progress.Show()
	g.updateLabels()
	go func() {
		for {
			select {
			case <-ticker.C:
				fyne.Do(g.updateLabels)
			case <-g.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (g *Grid) OnUnselected() {
	select {
	case <-g.quit:
		// Channel already closed, do nothing
	default:
		close(g.quit)
	}
	g.Selected = nil
	g.Progress.Hide()
	fyne.Do(g.updateLabels)
}

func (g *Grid) updateGrid() {
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
			g.labels.Name = valueLabel
		case "Length":
			g.labels.Length = valueLabel
		case "Pieces":
			g.labels.Pieces = valueLabel
		case "Status":
			g.labels.Status = valueLabel
		case "Peers":
			g.labels.Peers = valueLabel
		case "Downloaded":
			g.labels.Downloaded = valueLabel
			g.labels.DownloadedHeader = headerLabel // so it can be changed to sent bytes
		}
		allLabels = append(allLabels, valueLabel)
	}

	// Create grid with 2 columns (field name and value)
	g.Grid = container.NewGridWithColumns(2, allLabels...)
	g.Grid.Refresh()
}

func (g *Grid) updateLabels() {
	if g.Selected == nil {
		g.labels.Name.SetText("No torrent selected")
		g.labels.Length.SetText("No torrent selected")
		g.labels.Pieces.SetText("No torrent selected")
		g.labels.Status.SetText("No torrent selected")
		g.labels.Peers.SetText("No torrent selected")
		g.labels.Downloaded.SetText("No torrent selected")
		g.Grid.Refresh()
		return
	}

	// Update each label based on the selected torrent
	// Name, Length, Pieces
	g.labels.Name.SetText(g.Selected.Name)
	g.labels.Length.SetText(fmt.Sprintf("%d bytes", g.Selected.Length))
	g.labels.Pieces.SetText(fmt.Sprintf("%d", len(g.Selected.PieceHashes)))

	// Seeding or Downloading status
	if g.Selected.SeedingStatus != nil {
		// Seeding mode
		g.labels.Status.SetText("🌱 SEEDING")
		g.labels.Status.Importance = widget.HighImportance
		peers := g.Selected.SeedingStatus.GetActivePeers()
		seeded := g.Selected.SeedingStatus.GetSeededBytes()
		g.labels.Peers.SetText(fmt.Sprintf("%d", peers))
		g.labels.Downloaded.SetText(fmt.Sprintf("%d bytes", seeded))
		g.labels.DownloadedHeader.SetText("Seeded")
		g.Progress.SetValue(1) // Seeding is always 100%
		g.Progress.Hide()
	} else if g.Selected.DownloadStatus != nil {
		if g.Selected.Paused {
			// Paused
			g.labels.Status.SetText("⏸️ PAUSED")
			g.labels.Status.Importance = widget.HighImportance
		} else {
			// Downloading
			g.labels.Status.SetText("▶️ DOWNLOADING")
			g.labels.Status.Importance = widget.MediumImportance
		}
		g.labels.Peers.SetText(fmt.Sprintf("%d", g.Selected.DownloadStatus.PeersAmount))
		g.labels.DownloadedHeader.SetText("Downloaded")
		g.labels.Downloaded.SetText(fmt.Sprintf("%d/%d", g.Selected.DownloadStatus.DonePieces, len(g.Selected.PieceHashes)))
		g.Progress.SetValue(g.Selected.CalculateDownloadPercentage() / 100.0)
		g.Progress.Show()
	} else {
		// Neither seeding nor downloading (not started or unknown state)
		g.labels.Status.SetText("Not Started")
		g.labels.Status.Importance = widget.MediumImportance
		g.labels.Peers.SetText("0")
		g.labels.DownloadedHeader.SetText("Downloaded")
		g.labels.Downloaded.SetText("0")
		g.Progress.Hide()
	}

	g.Grid.Refresh()
}

func (tl *TorrentList) AddTorrent(t *torrent.Torrent) {
	tl.mu.Lock()
	tl.Torrents = append(tl.Torrents, t)
	tl.mu.Unlock()
	fyne.Do(tl.Widgets.Refresh)
}

func (tl *TorrentList) ForceUpdateDetails() {
	fyne.Do(tl.Grid.updateLabels)
}
