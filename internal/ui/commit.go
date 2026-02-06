package ui

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/larkin1/wmsproject/internal/api"
	"github.com/larkin1/wmsproject/internal/queue"
)

type CommitUI struct {
	widget.BaseWidget

	scannerInput  *widget.Entry
	locationLabel *widget.Label
	deltaInput    *widget.Entry
	toggleBtn     *widget.Button
	commitBtn     *widget.Button
	changeItemBtn *widget.Button
	error         *widget.RichText

	mode      string
	location  string
	itemID    int
	locations map[string][]int
	items     map[string]int
	items_r   map[int]string

	api       *api.Client
	queue     *queue.Queue
	basePath  string
	window    fyne.Window // Store the window for dialogs
}

func NewCommitUI(apiClient *api.Client, commitQueue *queue.Queue, basePath string) *CommitUI {
	c := &CommitUI{
		api:       apiClient,
		queue:     commitQueue,
		basePath:  basePath,
		mode:      "ADD",
		items:     make(map[string]int),
		items_r:   make(map[int]string),
		locations: make(map[string][]int),
	}

	return c
}

func (c *CommitUI) loadItems() {
	log.Println("[CommitUI] loadItems() called")
	// Clear old data
	c.items = make(map[string]int)
	c.items_r = make(map[int]string)

	itemsCSV := filepath.Join(c.basePath, "items.csv")
	log.Printf("[CommitUI] Loading items from CSV: %s\n", itemsCSV)

	// Always try to fetch fresh data from API
	err := c.api.ExportItemsToCSV(itemsCSV)
	if err != nil {
		log.Printf("[CommitUI] ExportItemsToCSV error: %v (will use cached JSON)\n", err)
		// Try to load from cache instead
		c.loadItemsFromCache()
		return
	}

	log.Println("[CommitUI] ExportItemsToCSV succeeded")

	// Load from CSV (fresh from API)
	if !c.loadItemsFromCSV(itemsCSV) {
		log.Println("[CommitUI] CSV load failed, trying cache")
		c.loadItemsFromCache()
	}
}

func (c *CommitUI) loadItemsFromCSV(itemsCSV string) bool {
	file, err := os.Open(itemsCSV)
	if err != nil {
		log.Printf("[CommitUI] Cannot open items.csv: %v\n", err)
		return false
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("[CommitUI] CSV read error: %v\n", err)
		return false
	}

	log.Printf("[CommitUI] CSV has %d records (including header)\n", len(records))

	if len(records) == 0 {
		log.Println("[CommitUI] CSV is empty")
		return false
	}

	for i, record := range records {
		if i == 0 {
			log.Printf("[CommitUI] Header: %v\n", record)
			continue // skip header
		}
		if len(record) < 2 {
			log.Printf("[CommitUI] Record %d has %d fields, skipping\n", i, len(record))
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			log.Printf("[CommitUI] Cannot parse ID '%s': %v\n", record[0], err)
			continue
		}
		name := strings.TrimSpace(record[1])
		if name != "" {
			c.items[name] = id
			c.items_r[id] = name
			log.Printf("[CommitUI] Loaded item: %s (ID: %d)\n", name, id)
		}
	}

	log.Printf("[CommitUI] Total items loaded from CSV: %d\n", len(c.items))
	return len(c.items) > 0
}

func (c *CommitUI) loadItemsFromCache() {
	log.Println("[CommitUI] loadItemsFromCache() called")
	cachePath := filepath.Join(c.basePath, "items.cache.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		log.Printf("[CommitUI] Cache not found: %v\n", err)
		return
	}

	// Minimal parsing of cache JSON
	// Instead of full unmarshal, we'll use the API method that already handles this
	log.Printf("[CommitUI] Cache file exists (%d bytes), will reload items from API\n", len(data))
}

func (c *CommitUI) loadLocations() {
	log.Println("[CommitUI] loadLocations() called")
	locationsData, err := c.api.FetchLocations()
	if err != nil {
		log.Printf("[CommitUI] FetchLocations error: %v\n", err)
		return
	}

	c.locations = make(map[string][]int)
	for _, loc := range locationsData {
		c.locations[loc.LocationName] = loc.Items
		log.Printf("[CommitUI] Loaded location: %s with items %v\n", loc.LocationName, loc.Items)
	}

	log.Printf("[CommitUI] Total locations loaded: %d\n", len(c.locations))
}

func (c *CommitUI) onScanned(text string) {
	log.Printf("[CommitUI] onScanned: '%s'\n", text)
	c.location = strings.TrimSpace(text)
	c.loadLocations()

	if itemIDs, ok := c.locations[c.location]; ok {
		log.Printf("[CommitUI] Location found with items: %v\n", itemIDs)
		// Location exists
		if len(itemIDs) == 0 {
			c.setError("Location has no items")
			return
		}

		if len(itemIDs) > 1 {
			log.Println("[CommitUI] Multiple items, showing dialog")
			c.showItemSelectDialog(itemIDs)
			return
		} else if len(itemIDs) == 1 {
			c.itemID = itemIDs[0]
			log.Printf("[CommitUI] Single item, auto-selected: %d\n", c.itemID)
		}
	} else {
		// Location doesn't exist - automatically show item picker
		log.Printf("[CommitUI] Location '%s' not found, showing item picker\n", c.location)
		c.setError(fmt.Sprintf("New location '%s' - select an item below:", c.location))
		c.itemID = 0
		c.showItemSearch()
		return
	}

	c.updateLocationLabel()
}

func (c *CommitUI) updateLocationLabel() {
	if c.location != "" {
		itemName := c.items_r[c.itemID]
		if itemName == "" {
			itemName = fmt.Sprintf("ID: %d", c.itemID)
		}
		c.locationLabel.SetText(fmt.Sprintf("Location: %s\nItem: %s", c.location, itemName))
		c.setError("")
	}
}

func (c *CommitUI) toggleMode() {
	if c.mode == "ADD" {
		c.mode = "SUB"
	} else {
		c.mode = "ADD"
	}
	c.toggleBtn.SetText("Mode: " + c.mode)
}

func (c *CommitUI) commit() {
	if c.location == "" || c.itemID == 0 {
		c.setError("No location or item selected")
		return
	}

	qty, err := strconv.Atoi(c.deltaInput.Text)
	if err != nil {
		c.setError("Invalid number")
		return
	}

	if c.mode == "SUB" {
		qty = -qty
	}

	log.Printf("[CommitUI] Submitting commit: location=%s, itemID=%d, qty=%d\n", c.location, c.itemID, qty)
	c.queue.SubmitCommit("TOUGHPAD01", c.location, qty, c.itemID)
	c.deltaInput.SetText("")
	c.setError("")
}

func (c *CommitUI) setError(msg string) {
	log.Printf("[CommitUI] setError: %s\n", msg)
	if msg == "" {
		c.error.ParseMarkdown("")
	} else {
		c.error.ParseMarkdown("**Status:** " + msg)
	}
}

// fuzzyMatch checks if query matches name with fuzzy matching
func (c *CommitUI) fuzzyMatch(query, name string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	name = strings.ToLower(name)

	if query == "" {
		return true // empty query matches everything
	}

	// Exact match
	if strings.Contains(name, query) {
		return true
	}

	// Fuzzy: check if all query chars appear in name in order
	queryIdx := 0
	for _, char := range name {
		if queryIdx < len(query) && char == rune(query[queryIdx]) {
			queryIdx++
		}
	}
	return queryIdx == len(query)
}

func (c *CommitUI) showItemSelectDialog(itemIDs []int) {
	log.Printf("[CommitUI] showItemSelectDialog called with %d items\n", len(itemIDs))
	// Create options for the select widget
	options := make([]string, len(itemIDs))
	itemMap := make(map[string]int)

	for i, id := range itemIDs {
		name := c.items_r[id]
		if name == "" {
			name = fmt.Sprintf("ID: %d", id)
		}
		options[i] = name
		itemMap[name] = id
		log.Printf("[CommitUI] Dialog option %d: %s (ID: %d)\n", i, name, id)
	}

	// Create the select widget
	selectWidget := widget.NewSelect(options, func(value string) {
		log.Printf("[CommitUI] Item selected from dialog: %s\n", value)
		if id, ok := itemMap[value]; ok {
			c.itemID = id
			c.updateLocationLabel() // Update label after selection
		}
	})
	selectWidget.PlaceHolder = "Select item..."
	if len(options) > 0 {
		selectWidget.SetSelected(options[0])
		c.itemID = itemMap[options[0]]
	}

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Multiple items found at this location. Select one:"),
		selectWidget,
	)

	log.Printf("[CommitUI] Creating dialog, window is nil: %v\n", c.window == nil)
	dlg := dialog.NewCustom("Select Item", "OK", form, c.window)
	dlg.SetOnClosed(func() {
		log.Println("[CommitUI] Item select dialog closed")
		c.updateLocationLabel() // Update label when dialog closes
	})
	dlg.Show()
	log.Println("[CommitUI] Dialog shown")
}

func (c *CommitUI) showItemSearch() {
	log.Println("[CommitUI] showItemSearch called")
	// Ensure items are loaded
	c.loadItems()

	// Build sorted list of item names
	var itemNames []string
	for name := range c.items {
		itemNames = append(itemNames, name)
	}
	sort.Strings(itemNames)

	log.Printf("[CommitUI] showItemSearch: found %d items\n", len(itemNames))

	if len(itemNames) == 0 {
		log.Println("[CommitUI] No items loaded!")
		c.setError("No items loaded from database")
		return
	}

	// Create search entry
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Type to search...")

	// Create select widget (will be filtered)
	selectWidget := widget.NewSelect(itemNames, func(value string) {
		log.Printf("[CommitUI] Item selected from search: %s\n", value)
		if id, ok := c.items[value]; ok {
			c.itemID = id
			log.Printf("[CommitUI] Item ID set to: %d\n", c.itemID)
			c.updateLocationLabel()
		}
	})
	selectWidget.PlaceHolder = "Search results..."

	// Update select options when search changes
	searchEntry.OnChanged = func(s string) {
		var filtered []string
		for _, name := range itemNames {
			if c.fuzzyMatch(s, name) {
				filtered = append(filtered, name)
			}
		}
		log.Printf("[CommitUI] Search '%s' filtered to %d items\n", s, len(filtered))
		selectWidget.Options = filtered
		if len(filtered) > 0 {
			selectWidget.SetSelected(filtered[0])
		}
	}

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Search and select an item:"),
		searchEntry,
		selectWidget,
	)

	log.Printf("[CommitUI] Creating item search dialog, window is nil: %v\n", c.window == nil)
	dlg := dialog.NewCustom("Select Item", "OK", form, c.window)
	dlg.SetOnClosed(func() {
		log.Println("[CommitUI] Item search dialog closed")
		c.updateLocationLabel() // Update label when dialog closes
	})
	dlg.Show()
	log.Println("[CommitUI] Item search dialog shown")
}

func (c *CommitUI) CreateRenderer() fyne.WidgetRenderer {
	log.Println("[CommitUI] CreateRenderer called")
	// Load data when renderer is created
	c.loadItems()
	c.loadLocations()

	c.scannerInput = widget.NewEntry()
	c.scannerInput.SetPlaceHolder("Scan location code...")
	c.scannerInput.OnSubmitted = func(s string) {
		c.onScanned(s)
		c.scannerInput.SetText("")
	}

	c.locationLabel = widget.NewLabel("Location: (waiting for scan)")

	c.deltaInput = widget.NewEntry()
	c.deltaInput.SetPlaceHolder("Enter quantity")

	c.toggleBtn = widget.NewButton("Mode: ADD", func() {
		c.toggleMode()
	})

	c.commitBtn = widget.NewButton("Commit", func() {
		c.commit()
	})

	c.changeItemBtn = widget.NewButton("Change Item", func() {
		log.Println("[CommitUI] Change Item button clicked")
		c.showItemSearch()
	})

	c.error = widget.NewRichTextFromMarkdown("")

	buttons := container.NewHBox(
		c.toggleBtn,
		c.commitBtn,
		c.changeItemBtn,
	)

	vbox := container.NewVBox(
		c.scannerInput,
		c.locationLabel,
		c.deltaInput,
		buttons,
		c.error,
	)

	log.Println("[CommitUI] Renderer created successfully")
	return widget.NewSimpleRenderer(vbox)
}

// SetWindow allows main to pass the window reference
func (c *CommitUI) SetWindow(w fyne.Window) {
	log.Printf("[CommitUI] SetWindow called, window is nil: %v\n", w == nil)
	c.window = w
}
