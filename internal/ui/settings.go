package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/larkin1/wmsproject/internal/api"
)

type SettingsUI struct {
	widget.BaseWidget

	urlInput   *widget.Entry
	keyInput   *widget.Entry
	submitBtn  *widget.Button
	errLabel   *widget.RichText

	onSubmit func(url, key string)
	basePath string
}

func NewSettingsUI(onSubmit func(url, key string), basePath string) *SettingsUI {
	return &SettingsUI{
		onSubmit: onSubmit,
		basePath: basePath,
	}
}

func (s *SettingsUI) checkCredentials(url, key string) bool {
	client := api.NewClient(url, key, s.basePath)
	return client.Check()
}

func (s *SettingsUI) submit() {
	url := s.urlInput.Text
	key := s.keyInput.Text

	if url == "" || key == "" {
		s.setError("URL and key cannot be empty")
		return
	}

	// Auto-prefix https if needed
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}

	s.setError("Checking credentials...")

	if !s.checkCredentials(url, key) {
		s.setError("Invalid credentials or cannot connect")
		return
	}

	s.setError("")
	s.onSubmit(url, key)
}

func (s *SettingsUI) setError(msg string) {
	if msg == "" {
		s.errLabel.ParseMarkdown("")
	} else {
		s.errLabel.ParseMarkdown(fmt.Sprintf("**%s**", msg))
	}
}

func (s *SettingsUI) CreateRenderer() fyne.WidgetRenderer {
	s.urlInput = widget.NewEntry()
	s.urlInput.SetPlaceHolder("API Base URL (e.g., https://your-api.example.com)")
	s.urlInput.OnSubmitted = func(text string) {
		// Focus removed - Fyne v2 doesn't support Entry.Focus()
	}

	s.keyInput = widget.NewEntry()
	s.keyInput.SetPlaceHolder("API Key")
	s.keyInput.Password = true
	s.keyInput.OnSubmitted = func(text string) {
		s.submit()
	}

	s.submitBtn = widget.NewButton("Submit", func() {
		s.submit()
	})
	s.submitBtn.Importance = widget.HighImportance

	s.errLabel = widget.NewRichTextFromMarkdown("")

	// Use container.NewCenter for centered labels instead of NewLabelWithAlignment
	title := widget.NewLabel("Warehouse Management System")
	subtitle := widget.NewLabel("Initial Configuration")

	vbox := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewLabel(""),
		widget.NewLabel("API Configuration:"),
		s.urlInput,
		s.keyInput,
		s.submitBtn,
		s.errLabel,
	)

	return widget.NewSimpleRenderer(container.NewCenter(vbox))
}
