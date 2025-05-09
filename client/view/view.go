package view

import (
	"client/view/toolbar"
	"client/view/torrentlist"
	"client/view/viewutils"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func CreateMainWindow() {
	// Create the application
	viewutils.MainApp = app.NewWithID("com.itaydali.gotorrent")
	viewutils.MainWindow = viewutils.MainApp.NewWindow("GoTorrent")
	// Create a toolbar with basic actions

	// Details panel
	detailPanel := container.NewVBox(
		widget.NewLabelWithStyle("Torrent Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	// Create a list for torrents
	torrentList := torrentlist.New(detailPanel)
	tb := toolbar.New(torrentList)

	// Layout content
	mainContent := container.NewHSplit(
		torrentList.Widgets,
		container.NewVBox(
			widget.NewLabel("Details"),
			detailPanel,
		),
	)
	mainContent.SetOffset(0.3)

	// Set up the main window content
	viewutils.MainWindow.SetContent(container.NewBorder(tb, nil, nil, nil, mainContent))
	viewutils.MainWindow.Resize(fyne.NewSize(800, 600))
	viewutils.MainWindow.ShowAndRun()
}
