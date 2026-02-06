package queue

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/larkin1/wmsproject/internal/api"
)

type Commit struct {
	DeviceID string `json:"device_id"`
	Location string `json:"location"`
	Delta    int    `json:"delta"`
	ItemID   int    `json:"item_id"`
}

type Queue struct {
	api           *api.Client
	filePath      string
	checkInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

func NewQueue(apiClient *api.Client, basePath string) *Queue {
	return &Queue{
		api:           apiClient,
		filePath:      filepath.Join(basePath, "pending_commits.json"),
		checkInterval: 5 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

func (q *Queue) Start() {
	q.wg.Add(1)
	go q.worker()
}

func (q *Queue) Stop() {
	close(q.stopChan)
	q.wg.Wait()
}

func (q *Queue) SubmitCommit(deviceID, location string, delta, itemID int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	commit := Commit{
		DeviceID: deviceID,
		Location: location,
		Delta:    delta,
		ItemID:   itemID,
	}

	queue := q.loadQueue()
	queue = append(queue, commit)
	q.saveQueue(queue)

	fmt.Printf("Commit queued: %+v\n", commit)
}

func (q *Queue) worker() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopChan:
			return
		case <-ticker.C:
			if q.internetAvailable() {
				q.processQueue()
			}
		}
	}
}

func (q *Queue) internetAvailable() bool {
	// Try to connect to a reliable host
	conn, err := net.DialTimeout("tcp", "8.8.8.8:443", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (q *Queue) processQueue() {
	q.mu.Lock()
	defer q.mu.Unlock()

	queue := q.loadQueue()
	if len(queue) == 0 {
		return
	}

	fmt.Printf("Processing %d pending commits...\n", len(queue))

	var newQueue []Commit
	for _, commit := range queue {
		_, err := q.api.SendCommit(commit.DeviceID, commit.Location, commit.Delta, commit.ItemID)
		if err != nil {
			fmt.Printf("Failed to send commit: %v\n", err)
			newQueue = append(newQueue, commit)
		} else {
			fmt.Printf("Committed: %s@%s delta=%d\n", commit.Location, commit.DeviceID, commit.Delta)
		}
	}

	q.saveQueue(newQueue)
}

func (q *Queue) loadQueue() []Commit {
	if _, err := os.Stat(q.filePath); os.IsNotExist(err) {
		return []Commit{}
	}

	data, err := os.ReadFile(q.filePath)
	if err != nil {
		return []Commit{}
	}

	var commits []Commit
	json.Unmarshal(data, &commits)
	return commits
}

func (q *Queue) saveQueue(commits []Commit) {
	data, _ := json.MarshalIndent(commits, "", "  ")
	os.WriteFile(q.filePath, data, 0644)
}
