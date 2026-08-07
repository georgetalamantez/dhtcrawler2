package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
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
	TimeMs      int       `json:"TimeMs"`
	Count       int       `json:"Count"`
	TotalPages  int       `json:"TotalPages"`
	CurrentPage int       `json:"CurrentPage"`
	Limit       int       `json:"Limit"`
	Items       []Torrent `json:"Items"`
}

var dbDir = "../.db"

func main() {
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/api/search", searchHandler)
	http.HandleFunc("/api/stats", statsHandler)
	http.HandleFunc("/search/", esProxyHandler)

	fmt.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func esProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Handle CORS for development if React is on a different port (e.g. 5173)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" && r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target, _ := url.Parse("http://127.0.0.1:9200")
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Strip the /search prefix from the path
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/search")
	
	// Strip Accept-Encoding to prevent gzip chunking issues with Vite proxy
	r.Header.Del("Accept-Encoding")
	
	proxy.ServeHTTP(w, r)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	sType := r.URL.Query().Get("type") // "name", "hash", "file"
	
	w.Header().Set("Content-Type", "application/json")
	if q == "" {
		json.NewEncoder(w).Encode(SearchResponse{})
		return
	}

	start := time.Now()

	limitStr := r.URL.Query().Get("limit")
	limit := 25
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	sortDir := r.URL.Query().Get("sort")
	if sortDir == "" {
		sortDir = "desc" // default to newest first
	}
	
	offset := (page - 1) * limit

	// Build Elasticsearch Query
	fields := []string{"title"}
	if sType == "hash" {
		fields = []string{"infohash"}
	} else if sType == "file" {
		fields = []string{"files.Path"}
	}
	
	esQuery := map[string]interface{}{
		"from": offset,
		"size": limit,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": q,
				"fields": fields,
			},
		},
		"sort": []map[string]interface{}{
			{
				"discovered_on": map[string]interface{}{
					"order": sortDir,
				},
			},
		},
	}
	
	queryBytes, _ := json.Marshal(esQuery)
	resp, err := http.Post("http://127.0.0.1:9200/torrents/_search", "application/json", bytes.NewBuffer(queryBytes))
	if err != nil {
		http.Error(w, "Elasticsearch error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	var esResponse struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&esResponse); err != nil {
		http.Error(w, "Error decoding ES response", http.StatusInternalServerError)
		return
	}

	totalCount := esResponse.Hits.Total.Value
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	var results []Torrent
	for _, hit := range esResponse.Hits.Hits {
		src := hit.Source
		
		t := Torrent{}
		if val, ok := src["title"].(string); ok { t.Name = val }
		if val, ok := src["infohash"].(string); ok { t.InfoHash = val }
		if val, ok := src["discovered_on"].(float64); ok { t.DiscoveredOn = int64(val) }
		if val, ok := src["size"].(float64); ok { t.Size = int64(val) }
		
		if cats, ok := src["category"].([]interface{}); ok {
			for _, c := range cats {
				if s, ok := c.(string); ok {
					t.Categories = append(t.Categories, s)
				}
			}
		}
		
		if files, ok := src["files"].([]interface{}); ok {
			for _, fInter := range files {
				if fMap, ok := fInter.(map[string]interface{}); ok {
					fInfo := FileInfo{}
					if p, ok := fMap["Path"].(string); ok { fInfo.Path = p }
					if s, ok := fMap["Size"].(float64); ok { fInfo.Size = int64(s) }
					t.Files = append(t.Files, fInfo)
				}
			}
		}
		results = append(results, t)
	}

	respPayload := SearchResponse{
		TimeMs:      int(time.Since(start).Milliseconds()),
		Count:       totalCount,
		TotalPages:  totalPages,
		CurrentPage: page,
		Limit:       limit,
		Items:       results,
	}

	json.NewEncoder(w).Encode(respPayload)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://127.0.0.1:9200/torrents/_count")
	if err != nil {
		http.Error(w, "Elasticsearch error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	var esResponse struct {
		Count int `json:"count"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&esResponse); err != nil {
		http.Error(w, "Error decoding ES response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"files":    1, // Hardcoded since we use 1 ES index
		"torrents": esResponse.Count,
	})
}

