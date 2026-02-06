package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	APIURL   string `json:"api_url"`
	APIKey   string `json:"api_key"`
	DeviceID string `json:"device_id"`
}

func Load(filePath string) (*Settings, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var settings Settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func Save(filePath string, settings *Settings) error {
	dir := filepath.Dir(filePath)
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func CreateDefault(filePath string) (*Settings, error) {
	settings := &Settings{
		APIURL:   "",
		APIKey:   "",
		DeviceID: "TOUGHPAD01",
	}

	err := Save(filePath, settings)
	return settings, err
}
