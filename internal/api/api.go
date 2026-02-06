package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL  string
	APIKey   string
	Client   *http.Client
	BasePath string
}

type CommitPayload struct {
	DeviceID string `json:"device_id"`
	Location string `json:"location"`
	Delta    int    `json:"delta"`
	ItemID   int    `json:"item_id"`
}

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Location struct {
	LocationName string `json:"location"`
	Items        []int  `json:"items"`
}

// CachedItems wraps items with metadata
type CachedItems struct {
	Timestamp int64  `json:"timestamp"`
	Items     []Item `json:"items"`
}

// CachedLocations wraps locations with metadata
type CachedLocations struct {
	Timestamp int64       `json:"timestamp"`
	Locations []Location  `json:"locations"`
}

func NewClient(baseURL, apiKey, basePath string) *Client {
	return &Client{
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		APIKey:   apiKey,
		BasePath: basePath,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) getCacheFilePath(filename string) string {
	return filepath.Join(c.BasePath, filename)
}

func (c *Client) saveItemsCache(items []Item) error {
	cached := CachedItems{
		Timestamp: time.Now().Unix(),
		Items:     items,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	cachePath := c.getCacheFilePath("items.cache.json")
	log.Printf("[API] Saving items cache to: %s\n", cachePath)
	return os.WriteFile(cachePath, data, 0644)
}

func (c *Client) loadItemsCache() ([]Item, error) {
	cachePath := c.getCacheFilePath("items.cache.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		log.Printf("[API] Items cache not found: %v\n", err)
		return nil, err
	}

	var cached CachedItems
	err = json.Unmarshal(data, &cached)
	if err != nil {
		log.Printf("[API] Failed to parse items cache: %v\n", err)
		return nil, err
	}

	log.Printf("[API] Loaded items cache from %s (%d items, cached at %d)\n", cachePath, len(cached.Items), cached.Timestamp)
	return cached.Items, nil
}

func (c *Client) saveLocationsCache(locations []Location) error {
	cached := CachedLocations{
		Timestamp: time.Now().Unix(),
		Locations: locations,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	cachePath := c.getCacheFilePath("locations.cache.json")
	log.Printf("[API] Saving locations cache to: %s\n", cachePath)
	return os.WriteFile(cachePath, data, 0644)
}

func (c *Client) loadLocationsCache() ([]Location, error) {
	cachePath := c.getCacheFilePath("locations.cache.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		log.Printf("[API] Locations cache not found: %v\n", err)
		return nil, err
	}

	var cached CachedLocations
	err = json.Unmarshal(data, &cached)
	if err != nil {
		log.Printf("[API] Failed to parse locations cache: %v\n", err)
		return nil, err
	}

	log.Printf("[API] Loaded locations cache from %s (%d locations, cached at %d)\n", cachePath, len(cached.Locations), cached.Timestamp)
	return cached.Locations, nil
}

func (c *Client) Check() bool {
	req, err := http.NewRequest("GET", c.BaseURL+"/rest/v1/items?select=*&limit=1", nil)
	if err != nil {
		return false
	}

	c.setAuthHeaders(req)
	resp, err := c.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (c *Client) SendCommit(deviceID, location string, delta, itemID int) (map[string]interface{}, error) {
	payload := CommitPayload{
		DeviceID: deviceID,
		Location: location,
		Delta:    delta,
		ItemID:   itemID,
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.BaseURL+"/rest/v1/commits", bytes.NewBuffer(data))
	c.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return result, nil
}

func (c *Client) FetchItems() ([]Item, error) {
	log.Println("[API] FetchItems() called")
	req, _ := http.NewRequest("GET", c.BaseURL+"/rest/v1/items?select=*", nil)
	c.setAuthHeaders(req)

	log.Printf("[API] Making request to: %s\n", c.BaseURL+"/rest/v1/items?select=*")
	resp, err := c.Client.Do(req)
	if err != nil {
		log.Printf("[API] Request error: %v (trying cache)\n", err)
		return c.loadItemsCache()
	}
	defer resp.Body.Close()

	log.Printf("[API] Response status: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[API] Response body: %s\n", string(body))

	var items []Item
	err = json.Unmarshal(body, &items)
	if err != nil {
		log.Printf("[API] JSON unmarshal error: %v (trying cache)\n", err)
		return c.loadItemsCache()
	}

	if resp.StatusCode >= 400 {
		log.Printf("[API] HTTP error %d (trying cache)\n", resp.StatusCode)
		return c.loadItemsCache()
	}

	// Success: save to cache
	if len(items) > 0 {
		c.saveItemsCache(items)
	}

	log.Printf("[API] Parsed %d items\n", len(items))
	for i, item := range items {
		log.Printf("[API] Item %d: ID=%d, Name=%s\n", i, item.ID, item.Name)
	}

	return items, nil
}

func (c *Client) FetchLocations() ([]Location, error) {
	log.Println("[API] FetchLocations() called")
	req, _ := http.NewRequest("GET", c.BaseURL+"/rest/v1/locations?select=*", nil)
	c.setAuthHeaders(req)

	log.Printf("[API] Making request to: %s\n", c.BaseURL+"/rest/v1/locations?select=*")
	resp, err := c.Client.Do(req)
	if err != nil {
		log.Printf("[API] Request error: %v (trying cache)\n", err)
		return c.loadLocationsCache()
	}
	defer resp.Body.Close()

	log.Printf("[API] Response status: %d\n", resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[API] Response body: %s\n", string(body))

	var locations []Location
	err = json.Unmarshal(body, &locations)
	if err != nil {
		log.Printf("[API] JSON unmarshal error: %v (trying cache)\n", err)
		return c.loadLocationsCache()
	}

	if resp.StatusCode >= 400 {
		log.Printf("[API] HTTP error %d (trying cache)\n", resp.StatusCode)
		return c.loadLocationsCache()
	}

	// Success: save to cache
	if len(locations) > 0 {
		c.saveLocationsCache(locations)
	}

	log.Printf("[API] Parsed %d locations\n", len(locations))
	for i, loc := range locations {
		log.Printf("[API] Location %d: %s with %d items\n", i, loc.LocationName, len(loc.Items))
	}

	return locations, nil
}

func (c *Client) ExportItemsToCSV(filePath string) error {
	log.Println("[API] ExportItemsToCSV() called")
	items, err := c.FetchItems()
	if err != nil {
		log.Printf("[API] FetchItems error: %v\n", err)
		return err
	}

	log.Printf("[API] Exporting %d items to CSV\n", len(items))

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"id", "name"})

	for _, item := range items {
		writer.Write([]string{fmt.Sprintf("%d", item.ID), item.Name})
	}

	writer.Flush()
	log.Printf("[API] CSV export complete: %s\n", filePath)
	return nil
}

func (c *Client) ExportLocationsToCSV(filePath string) error {
	log.Println("[API] ExportLocationsToCSV() called")
	locations, err := c.FetchLocations()
	if err != nil {
		log.Printf("[API] FetchLocations error: %v\n", err)
		return err
	}

	log.Printf("[API] Exporting %d locations to CSV\n", len(locations))

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"location", "items"})

	for _, loc := range locations {
		itemsStr := fmt.Sprintf("%v", loc.Items)
		writer.Write([]string{loc.LocationName, itemsStr})
	}

	writer.Flush()
	log.Printf("[API] CSV export complete: %s\n", filePath)
	return nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("apikey", c.APIKey)
}
