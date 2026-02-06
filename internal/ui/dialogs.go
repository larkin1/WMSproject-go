package ui

import (
	"fyne.io/fyne/v2"
)

// Placeholder for dialog utilities
// Can be expanded for item search dialogs, etc.

type ItemSearchDialog struct {
	onSelect func(name string)
}

func NewItemSearchDialog(onSelect func(name string)) *ItemSearchDialog {
	return &ItemSearchDialog{
		onSelect: onSelect,
	}
}

func (d *ItemSearchDialog) Show(parent fyne.Window) {
	// Placeholder
	// Would implement fuzzy search and selection
}
