package view

import (
	"client/view/toolbar"
	"client/view/torrentlist"
	"client/view/viewutils"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func CreateMainWindow() {
	// Create the application
	viewutils.MainApp = app.NewWithID("com.itaydali.gotorrent")
	viewutils.MainWindow = viewutils.MainApp.NewWindow("GoTorrent")

	// Create torrent list
	leecherList := torrentlist.New()
	seedingList := torrentlist.New() // Create a second list for seeding torrents

	// Create toolbar
	toolbar := toolbar.New(leecherList)

	// Create vertical split for main list and seeding list
	seedingLabel := widget.NewLabel("Seeding")
	leechingLabel := widget.NewLabel("Leeching")

	seedingSection := container.NewVBox(seedingLabel, seedingList.Widgets)
	leechingSection := container.NewVBox(leechingLabel, leecherList.Widgets)

	listsContainer := container.NewVSplit(leechingSection, seedingSection)
	listsContainer.SetOffset(0.7) // Main list takes 70% of height

	// Create vertical container for grid and progress bar
	gridWithProgress := container.NewVBox(
		leecherList.Grid,
		leecherList.Progress,
	)

	// Create main content with lists and grid
	mainContent := container.NewHSplit(listsContainer, gridWithProgress)
	mainContent.SetOffset(0.3)

	// Create main container with toolbar and content
	mainContainer := container.NewBorder(toolbar, nil, nil, nil, mainContent)

	// Set window content
	viewutils.MainWindow.SetContent(mainContainer)
	viewutils.MainWindow.Show()
	viewutils.MainApp.Run()
}
