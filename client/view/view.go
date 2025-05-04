package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var mainApp fyne.App
var mainWindow fyne.Window

func CreateMainWindow() {
	// Create the application
	mainApp = app.New()
	mainWindow = mainApp.NewWindow("GoTorrent")
	// Create a toolbar with basic actions
	toolbar := createMainToolbar()

	// Details panel
	detailPanel := container.NewVBox(
		widget.NewLabelWithStyle("Torrent Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	// Create a list for torrents
	torrentWidgetList := createTorrentWidgetList(detailPanel)

	// Layout content
	mainContent := container.NewHSplit(
		torrentWidgetList,
		container.NewVBox(
			widget.NewLabel("Details"),
			detailPanel,
		),
	)
	mainContent.SetOffset(0.3)

	// Set up the main window content
	mainWindow.SetContent(container.NewBorder(toolbar, nil, nil, nil, mainContent))
	mainWindow.Resize(fyne.NewSize(800, 600))
	mainWindow.ShowAndRun()
}

func showMessage(message string) {
	// Use the same app as the parent window
	newWindow := mainApp.NewWindow("Message")
	dialog := widget.NewLabel(message)
	newWindow.SetContent(container.NewVBox(
		dialog,
		widget.NewButton("Close", func() {
			newWindow.Close()
		}),
	))
	newWindow.Resize(fyne.NewSize(300, 150))
	newWindow.Show()
}
