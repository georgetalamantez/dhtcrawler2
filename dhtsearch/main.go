package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ostafen/clover/v2/document"
	"go.etcd.io/bbolt"
)

type FileInfo struct {
	Path string `json:"Path"`
	Size int64  `json:"Size"`
}

type Torrent struct {
	ID           string     `clover:"_id" json:"_id"`
	Name         string     `json:"Name"`
	InfoHash     string     `json:"InfoHash"`
	Files        []FileInfo `json:"Files"`
	Size         int64      `json:"TotalSize"`
	DiscoveredOn int64      `json:"DiscoveredOn"`
	Categories   []string   `json:"Categories"`
}

type SearchResponse struct {
	TimeMs int       `json:"TimeMs"`
	Count  int       `json:"Count"`
	Items  []Torrent `json:"Items"`
}

var dbDir = "../.db"

func main() {
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/api/search", searchHandler)
	http.HandleFunc("/api/stats", statsHandler)

	fmt.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getDBFiles() []string {
	var files []string
	entries, err := os.ReadDir(dbDir)
	if err != nil {
		log.Printf("Error reading db dir: %v", err)
		return files
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".db") {
			files = append(files, filepath.Join(dbDir, entry.Name()))
		}
	}
	return files
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sType := r.URL.Query().Get("type") // "name", "hash", "file"
	
	w.Header().Set("Content-Type", "application/json")
	if q == "" {
		json.NewEncoder(w).Encode(SearchResponse{})
		return
	}
	q = strings.ToLower(q)

	start := time.Now()
	dbFiles := getDBFiles()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []Torrent

	limit := 100

	for _, dbFile := range dbFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			db, err := bbolt.Open(file, 0600, &bbolt.Options{ReadOnly: true})
			if err != nil {
				log.Printf("Failed to open %s: %v", file, err)
				return
			}
			defer db.Close()

			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte("root"))
				if b == nil {
					return nil
				}
				c := b.Cursor()
				for k, v := c.First(); k != nil; k, v = c.Next() {
					// Check prefix to ensure it's a torrent document
					if !bytes.HasPrefix(k, []byte("c:torrents;d:")) {
						continue
					}

					doc, err := document.Decode(v)
					if err != nil {
						continue
					}

					var t Torrent
					if err := doc.Unmarshal(&t); err != nil {
						continue
					}

					match := false
					if sType == "hash" {
						if strings.Contains(strings.ToLower(t.InfoHash), q) {
							match = true
						}
					} else if sType == "file" {
						for _, f := range t.Files {
							if strings.Contains(strings.ToLower(f.Path), q) {
								match = true
								break
							}
						}
					} else { // default to name
						if strings.Contains(strings.ToLower(t.Name), q) {
							match = true
						}
					}

					if match {
						mu.Lock()
						if len(results) < limit {
							results = append(results, t)
						}
						mu.Unlock()
						// Early exit if limit reached (per file approximation)
						if len(results) >= limit {
							return nil
						}
					}
				}
				return nil
			})
		}(dbFile)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].DiscoveredOn > results[j].DiscoveredOn
	})

	if len(results) > limit {
		results = results[:limit]
	}

	resp := SearchResponse{
		TimeMs: int(time.Since(start).Milliseconds()),
		Count:  len(results),
		Items:  results,
	}

	json.NewEncoder(w).Encode(resp)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	files := getDBFiles()
	var totalTorrents int
	for _, f := range files {
		db, err := bbolt.Open(f, 0600, &bbolt.Options{ReadOnly: true})
		if err == nil {
			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte("root"))
				if b != nil {
					totalTorrents += b.Stats().KeyN
				}
				return nil
			})
			db.Close()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"files":    len(files),
		"torrents": totalTorrents,
	})
}
