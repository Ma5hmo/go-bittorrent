package view

import (
	"client/torrent"
	"client/torrent/torrentstatus"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var torrentListView struct {
	list     *widget.List
	torrents []torrent.Torrent
	selected *torrent.Torrent
}

var c = 1

func createTorrentWidgetList(detailContainer *fyne.Container) *widget.List {
	torrentListView.list = widget.NewList(
		func() int {
			return len(torrentListView.torrents)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			t := torrentListView.torrents[i]
			o.(*widget.Label).SetText(t.Name)
		},
	)
	torrentListView.list.OnSelected = func(id widget.ListItemID) {
		t := &torrentListView.torrents[id]
		torrentListView.selected = t

		status := t.DownloadStatus
		if status == nil {
			status = &torrentstatus.TorrentStatus{DonePieces: 0, PeersAmount: 0}
		}
		// Clear old details
		detailContainer.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("Torrent Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Name:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(t.Name),
			widget.NewLabelWithStyle("Length:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d bytes", t.Length)),
			widget.NewLabelWithStyle("Pieces:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", len(t.PieceHashes))),
			widget.NewLabel(fmt.Sprintf("%d", c)),
			widget.NewLabelWithStyle("Peers:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", status.PeersAmount)),
			widget.NewLabelWithStyle("Downloaded Pieces:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", status.DonePieces)),
		}
		detailContainer.Refresh()
	}
	return torrentListView.list
}
