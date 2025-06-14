package view

import (
	"client/common"
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
	grid := torrentlist.NewGrid()
	leecherList := torrentlist.New(grid)
	seedingList := torrentlist.New(grid) // Create a second list for seeding torrents

	// Create toolbar
	toolbar := toolbar.New(leecherList, seedingList)

	// Create vertical split for main list and seeding list
	seedingLabel := widget.NewLabel("Seeding")
	leechingLabel := widget.NewLabel("Leeching")

	seedingSection := container.NewBorder(seedingLabel, nil, nil, nil, container.NewVScroll(seedingList.Widgets))
	leechingSection := container.NewBorder(leechingLabel, nil, nil, nil, container.NewVScroll(leecherList.Widgets))

	listsContainer := container.NewVSplit(leechingSection, seedingSection)
	listsContainer.SetOffset(0.6) // Main list takes 70% of height

	// Create vertical container for grid and progress bar
	gridWithProgress := container.NewVBox(
		leecherList.Grid.Grid,
		leecherList.Grid.Progress,
	)

	// Create main content with lists and grid
	mainContent := container.NewHSplit(listsContainer, gridWithProgress)
	mainContent.SetOffset(0.3)

	// Add encryption toggle button in a bottom-right toolbar
	encryptionLabel := func() string {
		if common.AppState.IsTrafficAESEncrypted {
			return "AES: ON"
		}
		return "AES: OFF"
	}
	var encryptionBtn *widget.Button
	setEncryptionBtnStyle := func() {
		if common.AppState.IsTrafficAESEncrypted {
			encryptionBtn.Importance = widget.HighImportance
		} else {
			encryptionBtn.Importance = widget.MediumImportance
		}
	}

	encryptionBtn = widget.NewButton(encryptionLabel(), func() {
		common.AppState.IsTrafficAESEncrypted = !common.AppState.IsTrafficAESEncrypted
		encryptionBtn.SetText(encryptionLabel())
		setEncryptionBtnStyle()
	})
	setEncryptionBtnStyle()

	encryptionToolbar := container.NewHBox(encryptionBtn) // right-aligned, no label
	bottomBar := container.NewBorder(nil, nil, nil, encryptionToolbar)

	finalContainer := container.NewBorder(toolbar, bottomBar, nil, nil, mainContent)

	// Set window content
	viewutils.MainWindow.SetContent(finalContainer)
	viewutils.MainWindow.Show()
	viewutils.MainApp.Run()
}
