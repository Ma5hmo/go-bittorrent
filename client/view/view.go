package view

import (
	"client/view/toolbar"
	"client/view/torrentlist"
	"client/view/viewutils"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

func CreateMainWindow() {
	// Create the application
	viewutils.MainApp = app.NewWithID("com.itaydali.gotorrent")
	viewutils.MainWindow = viewutils.MainApp.NewWindow("GoTorrent")

	// Create torrent list
	torrentList := torrentlist.New()

	// Create toolbar
	toolbar := toolbar.New(torrentList)

	// Create main content with list and grid
	mainContent := container.NewHSplit(torrentList.Widgets, torrentList.Grid)
	mainContent.SetOffset(0.3)

	// Create vertical container for grid and progress bar
	gridWithProgress := container.NewVBox(
		torrentList.Grid,
		torrentList.Progress,
	)

	// Update main content to use the new container
	mainContent = container.NewHSplit(torrentList.Widgets, gridWithProgress)
	mainContent.SetOffset(0.3)

	// Create main container with toolbar and content
	mainContainer := container.NewBorder(toolbar, nil, nil, nil, mainContent)

	// Set window content
	viewutils.MainWindow.SetContent(mainContainer)
	viewutils.MainWindow.Show()
	viewutils.MainApp.Run()
}
