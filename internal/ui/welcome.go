package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type WelcomeScreen struct {
	widget.BaseWidget
	onScreenChange func(string)
}

func NewWelcomeScreen(onScreenChange func(string)) *WelcomeScreen {
	w := &WelcomeScreen{
		onScreenChange: onScreenChange,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *WelcomeScreen) CreateRenderer() fyne.WidgetRenderer {
	addBtn := widget.NewButton("Add/Remove Stock", func() {
		w.onScreenChange("commit")
	})
	addBtn.Importance = widget.HighImportance

	exitBtn := widget.NewButton("Exit", func() {
		fyne.CurrentApp().Quit()
	})

	title := widget.NewLabel("Warehouse Management System")
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	subtitle := widget.NewLabel("Select an option below")
	subtitle.Alignment = fyne.TextAlignCenter

	vbox := container.NewVBox(
		title,
		subtitle,
		widget.NewSeparator(),
		addBtn,
		exitBtn,
	)

	centered := container.NewCenter(vbox)
	return widget.NewSimpleRenderer(centered)
}
