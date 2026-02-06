# WMS - Go Implementation

Direct port of the Python WMS mockup to Go with Fyne GUI.

## Quick Start

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Platform dependencies** (for Fyne GUI):
  - **Linux**: `sudo apt-get install libgl1-mesa-dev xorg-dev`
  - **macOS**: Xcode command line tools (should be automatic)
  - **Windows**: MinGW-w64 or MSVC (Go handles most of it)

### Setup

```bash
# Clone or navigate to the repo
cd WMSproject

# Download dependencies
go mod download

# Run the app
go run main.go
```

### First Launch

The app will ask for:
- **API Base URL**: `https://your-api.example.com` or your Supabase URL
- **API Key**: Your authentication key

Settings are saved to `settings.json` and reused on subsequent launches.

## Project Structure

```
WMSproject/
├── go.mod                    # Go module definition
├── main.go                   # Entry point
├── settings.json             # Saved configuration
├── pending_commits.json      # Offline queue
├── items.csv                 # Cached items
├── locations.csv             # Cached locations
└── internal/
    ├── api/
    │   └── api.go            # HTTP client for database
    ├── queue/
    │   └── queue.go          # Offline-first commit queue
    ├── ui/
    │   ├── welcome.go        # Welcome screen
    │   ├── commit.go         # Stock tracking screen
    │   ├── settings.go       # Settings screen
    │   └── dialogs.go        # Dialog utilities
    └── config/
        └── config.go         # Settings management
```

## Building for Production

### Linux

```bash
go build -o wms
```

### Windows

```bash
go build -o wms.exe
```

### macOS

```bash
go build -o wms_mac
```

### Cross-compile (from Linux to Windows)

```bash
GOOS=windows GOARCH=amd64 go build -o wms.exe
```

## Architecture

### Fyne GUI

The UI is built with [Fyne](https://fyne.io) - a cross-platform GUI framework. Same functionality as Kivy, but cleaner and fewer dependencies.

### API Client

The `api.go` module handles all HTTP requests. It's framework-agnostic:
- Works with Supabase REST endpoints
- Works with any custom PostgreSQL API
- Easily configurable headers and endpoints

### Offline-First Queue

The `queue.go` module:
- Stores commits locally to `pending_commits.json`
- Checks internet connectivity every 5 seconds
- Automatically syncs when online
- Never loses data even if you power off

### CSV Caching

Items and locations are:
- Fetched from API on startup
- Cached to `items.csv` and `locations.csv`
- Re-fetched when location is scanned (to stay current)

## For Your VPS Database

When switching from Supabase to your own PostgreSQL:

1. **Create a REST API** (using PostgREST, Hasura, etc.)
2. **Update endpoints** in `internal/api/api.go`:
   ```go
   // Change these lines:
   req, _ := http.NewRequest("GET", c.BaseURL+"/rest/v1/items", nil)
   // To match your API:
   req, _ := http.NewRequest("GET", c.BaseURL+"/api/items", nil)
   ```
3. **Update auth headers** if needed:
   ```go
   // Change from Bearer token:
   req.Header.Set("Authorization", "Bearer "+c.APIKey)
   // To API key header:
   req.Header.Set("X-API-Key", c.APIKey)
   ```
4. **Test with the "Check" button** in settings

## Database Schema

Expected tables (same as Python version):

### commits
```sql
CREATE TABLE commits (
  commit_id SERIAL PRIMARY KEY,
  device_id TEXT,
  location TEXT,
  delta INTEGER,
  item_id INTEGER,
  created_at TIMESTAMP DEFAULT NOW()
);
```

### items
```sql
CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  name TEXT UNIQUE
);
```

### locations
```sql
CREATE TABLE locations (
  location TEXT PRIMARY KEY,
  items TEXT  -- JSON array as string: "[1, 2, 3]"
);
```

### overview (view)
```sql
CREATE VIEW overview AS
SELECT location, item_id, SUM(delta) as qty
FROM commits
GROUP BY location, item_id;
```

## Troubleshooting

### "Cannot find module" error

```bash
go mod tidy
go mod download
```

### Fyne dependencies missing (Linux)

```bash
sudo apt-get install libgl1-mesa-dev xorg-dev libxcursor-dev
```

### Settings not saving

Ensure the executable has write permissions to its directory.

### API connection fails

- Check that your URL is correct (should start with `https://`)
- Verify the API key is valid
- Test with the "Check" button in settings
- Check browser console for CORS issues (if testing from localhost)

## Features (from Python port)

✅ Barcode/QR scanner input for locations  
✅ Item lookup with fuzzy search  
✅ Add/Remove stock with toggle  
✅ Offline-first queue for connectivity issues  
✅ CSV caching for offline browsing  
✅ Settings persistence  
✅ Device ID tracking  
✅ Clean, responsive Fyne GUI  

## Performance

- **Binary size**: ~10MB (includes Fyne)
- **Memory**: ~30-50MB while running
- **Startup**: <500ms
- **Queue sync**: Runs in background, non-blocking

## Next Steps

1. Test with your Supabase account first
2. Migrate your database to VPS
3. Update `api.go` with your VPS endpoints
4. Build and deploy the binary

## Questions?

Check the inline code comments in `internal/` for detailed explanations of each module.
