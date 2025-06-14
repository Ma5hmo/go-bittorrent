package viewutils

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var MainApp fyne.App
var MainWindow fyne.Window

func ShowMessage(message string) {
	// Use the same app as the parent window
	fyne.Do(func() {
		newWindow := MainApp.NewWindow("Message")
		dialog := widget.NewLabel(message)
		newWindow.SetContent(container.NewVBox(
			dialog,
			widget.NewButton("Close", func() {
				newWindow.Close()
			}),
		))
		newWindow.Resize(fyne.NewSize(300, 150))
		newWindow.Show()
	})
}
