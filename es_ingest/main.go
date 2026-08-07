package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/ostafen/clover/v2/document"
	"go.etcd.io/bbolt"
)

type FileInfo struct {
	Path string `json:"Path"`
	Size int64  `json:"Size"`
}

type Torrent struct {
	ID           string     `json:"-"` // Don't index the internal bolt ID
	Name         string     `json:"title"` // Mapping Name to title for React Search UI
	InfoHash     string     `json:"infohash"` // Lowercase for Search UI
	Files        []FileInfo `json:"files"`
	Size         int64      `json:"size"`
	DiscoveredOn int64      `json:"discovered_on"`
	Categories   []string   `json:"category"` // Mapping Categories to category for Search UI
}

// TorrentDb represents the structure as it exists in BoltDB
type TorrentDb struct {
	ID           string     `clover:"_id" json:"_id"`
	Name         string     `json:"Name"`
	InfoHash     string     `json:"InfoHash"`
	Files        []FileInfo `json:"Files"`
	Size         int64      `json:"TotalSize"`
	DiscoveredOn int64      `json:"DiscoveredOn"`
	Categories   []string   `json:"Categories"`
}

var dbDir = "../.db"
var indexName = "torrents"

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

func main() {
	start := time.Now()
	
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://127.0.0.1:9200",
		},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	// Ping the Elasticsearch server
	res, err := es.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()
	log.Println("Connected to Elasticsearch:", res.Status())

	files := getDBFiles()
	fmt.Printf("Found %d DB files to process for ingestion.\n", len(files))

	totalIndexed := 0

	for _, file := range files {
		fmt.Printf("Processing %s...\n", filepath.Base(file))

		db, err := bbolt.Open(file, 0600, &bbolt.Options{ReadOnly: true})
		if err != nil {
			log.Printf("Could not open %s: %v", file, err)
			continue
		}

		err = db.View(func(tx *bbolt.Tx) error {
			root := tx.Bucket([]byte("root"))
			if root == nil {
				return nil
			}

			var buf bytes.Buffer
			batchSize := 0
			
			c := root.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if !bytes.HasPrefix(k, []byte("c:torrents;d:")) {
					continue
				}

				doc, err := document.Decode(v)
				if err != nil {
					continue
				}

				var tDb TorrentDb
				if err := doc.Unmarshal(&tDb); err != nil {
					continue
				}

				// Map to Elasticsearch struct
				t := Torrent{
					Name:         tDb.Name,
					InfoHash:     tDb.InfoHash,
					Files:        tDb.Files,
					Size:         tDb.Size,
					DiscoveredOn: tDb.DiscoveredOn,
					Categories:   tDb.Categories,
				}

				// Prepare Bulk API payload
				meta := []byte(fmt.Sprintf(`{ "index" : { "_index" : "%s", "_id" : "%s" } }%s`, indexName, t.InfoHash, "\n"))
				data, err := json.Marshal(t)
				if err != nil {
					continue
				}
				data = append(data, "\n"...)

				buf.Grow(len(meta) + len(data))
				buf.Write(meta)
				buf.Write(data)
				batchSize++

				if batchSize >= 1000 {
					if err := sendBulk(es, &buf); err != nil {
						log.Printf("Error sending bulk: %v", err)
					}
					totalIndexed += batchSize
					batchSize = 0
					buf.Reset()
				}
			}

			// Send remaining
			if batchSize > 0 {
				if err := sendBulk(es, &buf); err != nil {
					log.Printf("Error sending bulk: %v", err)
				}
				totalIndexed += batchSize
			}
			return nil
		})

		if err != nil {
			log.Printf("Error during ingestion in %s: %v", file, err)
		}
		db.Close()
	}

	fmt.Println("==================================================")
	fmt.Printf("Ingestion Complete in %v!\n", time.Since(start))
	fmt.Printf("Total Records Indexed: %d\n", totalIndexed)
}

func sendBulk(es *elasticsearch.Client, buf *bytes.Buffer) error {
	req := esapi.BulkRequest{
		Index: indexName,
		Body:  bytes.NewReader(buf.Bytes()),
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk request error: %s", res.String())
	}
	return nil
}
