package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/larkin1/wmsproject/internal/api"
	"github.com/larkin1/wmsproject/internal/queue"
	"github.com/larkin1/wmsproject/internal/ui"
)

var (
	basePath     string
	settingsPath string
	appAPI       *api.Client
	commitQueue  *queue.Queue
	mainWindow   fyne.Window
	fyneApp      fyne.App
)

func init() {
	// This will be overridden in main() with proper Fyne storage
	if exe, err := os.Executable(); err == nil {
		basePath = filepath.Dir(exe)
	} else {
		basePath, _ = os.Getwd()
	}
	settingsPath = filepath.Join(basePath, "settings.json")
}

func getStoragePath() string {
	if fyneApp == nil {
		return basePath
	}
	// Use Fyne's storage root for mobile compatibility
	uri := fyneApp.Storage().RootURI()
	if uri.Scheme() == "file" {
		path := uri.Path()
		log.Printf("[Main] Using Fyne storage path: %s\n", path)
		return path
	}
	return basePath
}

func loadSettings() (bool, error) {
	// Update paths using Fyne storage
	basePath = getStoragePath()
	settingsPath = filepath.Join(basePath, "settings.json")

	log.Printf("[Main] Loading settings from: %s\n", settingsPath)

	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		log.Println("[Main] Settings file not found, creating empty")
		// Create empty settings
		emptySettings := map[string]string{
			"api_url": "",
			"api_key": "",
		}
		data, _ := json.MarshalIndent(emptySettings, "", "  ")
		err := os.WriteFile(settingsPath, data, 0644)
		if err != nil {
			log.Printf("[Main] Failed to write settings: %v\n", err)
		}
		return false, nil
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		log.Printf("[Main] Failed to read settings: %v\n", err)
		return false, err
	}

	var settings map[string]string
	err = json.Unmarshal(data, &settings)
	if err != nil {
		log.Printf("[Main] Failed to parse settings: %v\n", err)
		return false, err
	}

	log.Printf("[Main] Settings loaded: api_url=%s\n", settings["api_url"])

	if settings["api_url"] == "" || settings["api_key"] == "" {
		log.Println("[Main] Settings incomplete")
		return false, nil
	}

	appAPI = api.NewClient(settings["api_url"], settings["api_key"], basePath)
	commitQueue = queue.NewQueue(appAPI, basePath)
	commitQueue.Start()

	log.Println("[Main] API client and queue initialized")
	return true, nil
}

func switchScreen(screenName string) {
	log.Printf("[Main] Switching to screen: %s\n", screenName)
	switch screenName {
	case "commit":
		commitUI := ui.NewCommitUI(appAPI, commitQueue, basePath)
		commitUI.SetWindow(mainWindow)
		mainWindow.SetContent(commitUI)
	case "welcome":
		mainWindow.SetContent(makeApp())
	default:
		mainWindow.SetContent(makeApp())
	}
}

func main() {
	log.Println("[Main] Starting WMS app")
	a := app.NewWithID("com.velocidrone.velocidrone")
	fyneApp = a

	w := a.NewWindow("WMS - Warehouse Management System")
	w.Resize(fyne.NewSize(600, 800))
	mainWindow = w

	// Initialize storage path
	basePath = getStoragePath()
	log.Printf("[Main] Base path: %s\n", basePath)

	// Ensure directory exists
	os.MkdirAll(basePath, 0755)

	hasSettings, _ := loadSettings()

	if !hasSettings {
		log.Println("[Main] No settings found, showing settings screen")
		// Show settings screen
		settingsUI := ui.NewSettingsUI(func(apiURL, apiKey string) {
			log.Printf("[Main] Settings saved: %s\n", apiURL)
			appAPI = api.NewClient(apiURL, apiKey, basePath)
			commitQueue = queue.NewQueue(appAPI, basePath)
			commitQueue.Start()

			// Save settings
			settings := map[string]string{
				"api_url": apiURL,
				"api_key": apiKey,
			}
			data, _ := json.MarshalIndent(settings, "", "  ")
			err := os.WriteFile(settingsPath, data, 0644)
			if err != nil {
				log.Printf("[Main] Failed to save settings: %v\n", err)
			}

			// Show welcome screen
			w.SetContent(makeApp())
		}, basePath)

		w.SetContent(settingsUI)
	} else {
		log.Println("[Main] Settings found, showing welcome screen")
		w.SetContent(makeApp())
	}

	w.ShowAndRun()

	if commitQueue != nil {
		log.Println("[Main] Stopping queue")
		commitQueue.Stop()
	}
}

func makeApp() fyne.CanvasObject {
	return container.NewVBox(
		ui.NewWelcomeScreen(switchScreen),
	)
}
