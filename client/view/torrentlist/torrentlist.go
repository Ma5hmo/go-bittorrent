package torrentlist

import (
	"client/torrent"
	"client/torrent/torrentstatus"
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
	mu              sync.RWMutex
	quit            chan struct{}
}

func New(detailContainer *fyne.Container) *TorrentList {
	tl := new(TorrentList)
	tl.quit = make(chan struct{})
	tl.detailContainer = detailContainer

	tl.Widgets = widget.NewList(
		func() int {
			return len(tl.Torrents)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			t := tl.Torrents[i]
			o.(*widget.Label).SetText(t.Name)
		},
	)

	tl.Widgets.OnSelected = func(id widget.ListItemID) {
		ticker := time.NewTicker(500 * time.Millisecond)
		go func() {
			for {
				select {
				case <-ticker.C:
					tl.showTorrentOnContainer(id)
				case <-tl.quit:
					ticker.Stop()
					return
				}
			}
		}()
	}
	tl.Widgets.OnUnselected = func(id widget.ListItemID) {
		close(tl.quit)
		// make it go unselected (selected = nil and update gui)
		tl.Selected = nil
		tl.detailContainer.Objects = []fyne.CanvasObject{}
	}
	return tl
}

func (tl *TorrentList) AddTorrent(t *torrent.Torrent) {
	tl.mu.Lock()
	tl.Torrents = append(tl.Torrents, t)
	tl.mu.Unlock()

	fyne.Do(func() {
		tl.Widgets.Refresh()
	})
}

func (tl *TorrentList) GetTorrent(index int) *torrent.Torrent {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	if index < 0 || index >= len(tl.Torrents) {
		return nil
	}
	return tl.Torrents[index]
}

func (tl *TorrentList) showTorrentOnContainer(index int) {
	tl.Selected = tl.GetTorrent(index)
	if tl.Selected == nil || tl.Selected.TorrentFile == nil {
		fyne.DoAndWait(func() {
			tl.detailContainer.Refresh()
		})
		return
	}
	status := tl.Selected.DownloadStatus
	if status == nil {
		status = &torrentstatus.TorrentStatus{DonePieces: 0, PeersAmount: 0}
	}
	fyne.DoAndWait(func() {
		// Clear old details
		tl.detailContainer.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("Name:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(tl.Selected.Name),
			widget.NewLabelWithStyle("Length:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d bytes", tl.Selected.Length)),
			widget.NewLabelWithStyle("Pieces:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", len(tl.Selected.PieceHashes))),
			widget.NewLabelWithStyle("Peers:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", status.PeersAmount)),
			widget.NewLabelWithStyle("Downloaded Pieces:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", status.DonePieces)),
		}
		tl.detailContainer.Refresh()
	})
}
