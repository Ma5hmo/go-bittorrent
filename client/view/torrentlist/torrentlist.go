package torrentlist

import (
	"client/torrent"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type TorrentList struct {
	Widgets         *widget.List
	Torrents        []*torrent.Torrent
	Selected        *torrent.Torrent
	detailContainer *fyne.Container
	detailWidgets   torrentDetailWidgets
	mu              sync.RWMutex
	quit            chan struct{}
}

// torrentDetailWidgets groups the widgets to be updated dynamically
type torrentDetailWidgets struct {
	NameLabel       *widget.Label
	LengthLabel     *widget.Label
	PiecesLabel     *widget.Label
	PeersLabel      *widget.Label
	DownloadedLabel *widget.Label
	ProgressBar     *widget.ProgressBar
}

func New(detailContainer *fyne.Container) *TorrentList {
	tl := &TorrentList{
		detailContainer: detailContainer,
		quit:            make(chan struct{}),
	}
	tl.detailWidgets = torrentDetailWidgets{
		NameLabel:       widget.NewLabel(""),
		LengthLabel:     widget.NewLabel(""),
		PiecesLabel:     widget.NewLabel(""),
		PeersLabel:      widget.NewLabel(""),
		DownloadedLabel: widget.NewLabel(""),
		ProgressBar:     widget.NewProgressBar(),
	}
	tl.detailWidgets.ProgressBar.Min = 0.0
	tl.detailWidgets.ProgressBar.Max = 100.0

	detailContainer.Objects = []fyne.CanvasObject{}

	tl.Widgets = widget.NewList(
		func() int { return len(tl.Torrents) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(tl.Torrents[i].Name)
		},
	)

	tl.Widgets.OnSelected = func(id widget.ListItemID) {
		select {
		case <-tl.quit:
			// Channel already closed, do nothing
		default:
			close(tl.quit)
		}
		tl.quit = make(chan struct{})
		ticker := time.NewTicker(500 * time.Millisecond)
		tl.updateDetailWidgets(id)
		go func() {
			for {
				select {
				case <-ticker.C:
					tl.updateDetailWidgets(id)
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
		tl.detailContainer.Objects = []fyne.CanvasObject{}
		fyne.Do(func() {
			// Clear widget texts when unselected
			tl.detailWidgets.NameLabel.SetText("")
			tl.detailWidgets.LengthLabel.SetText("")
			tl.detailWidgets.PiecesLabel.SetText("")
			tl.detailWidgets.PeersLabel.SetText("")
			tl.detailWidgets.DownloadedLabel.SetText("")
			tl.detailWidgets.ProgressBar.SetValue(0)
		})
	}

	return tl
}

func (tl *TorrentList) AddTorrent(t *torrent.Torrent) {
	tl.mu.Lock()
	tl.Torrents = append(tl.Torrents, t)
	tl.mu.Unlock()
	fyne.Do(tl.Widgets.Refresh)
}

func (tl *TorrentList) GetTorrent(index int) *torrent.Torrent {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	if index < 0 || index >= len(tl.Torrents) {
		return nil
	}
	return tl.Torrents[index]
}

func (tl *TorrentList) updateDetailWidgets(index int) {
	tl.Selected = tl.GetTorrent(index)
	if tl.Selected == nil || tl.Selected.TorrentFile == nil {
		return
	}

	fyne.Do(func() {
		tl.setupDetailContainer()
		tl.updateDetailLabels()
	})
}

func (tl *TorrentList) setupDetailContainer() {
	if len(tl.detailContainer.Objects) == 0 {
		tl.detailContainer.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("Name:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			tl.detailWidgets.NameLabel,
			widget.NewLabelWithStyle("Length:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			tl.detailWidgets.LengthLabel,
			widget.NewLabelWithStyle("Pieces:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			tl.detailWidgets.PiecesLabel,
		}
	}
	if tl.Selected.DownloadStatus != nil && len(tl.detailContainer.Objects) == 6 {
		// download started and the detail container havent been updated yet
		tl.detailContainer.Objects = append(tl.detailContainer.Objects,
			widget.NewLabelWithStyle("Peers:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			tl.detailWidgets.PeersLabel,
			widget.NewLabelWithStyle("Downloaded Pieces:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			tl.detailWidgets.DownloadedLabel,
			tl.detailWidgets.ProgressBar)
	}
}

func (tl *TorrentList) updateDetailLabels() {
	percentage := tl.Selected.CalculateDownloadPercentage()
	tl.detailWidgets.NameLabel.SetText(tl.Selected.Name)
	tl.detailWidgets.LengthLabel.SetText(fmt.Sprintf("%d bytes", tl.Selected.Length))
	tl.detailWidgets.PiecesLabel.SetText(fmt.Sprintf("%d", len(tl.Selected.PieceHashes)))

	if tl.Selected.DownloadStatus != nil {
		tl.detailWidgets.PeersLabel.SetText(fmt.Sprintf("%d", tl.Selected.DownloadStatus.PeersAmount))
		tl.detailWidgets.DownloadedLabel.SetText(fmt.Sprintf("%d", tl.Selected.DownloadStatus.DonePieces))
		tl.detailWidgets.ProgressBar.SetValue(percentage)
	}
}
