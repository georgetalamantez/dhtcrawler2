package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Hit struct {
	Source struct {
		InfoHash string `json:"infohash"`
		Title    string `json:"title"`
		Size     int64  `json:"size"`
	} `json:"_source"`
}

type SearchResponse struct {
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
		Hits []Hit `json:"hits"`
	} `json:"hits"`
}

func main() {
	// Default output path assumes it is run from dhtcrawler2/es_export
	outPath := "../../torrent-paradise/index-generator/dump.csv"
	if len(os.Args) > 1 {
		outPath = os.Args[1]
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	cfg := elasticsearch.Config{
		Addresses: []string{"http://127.0.0.1:9200"},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	res, err := es.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()
	log.Println("Connected to Elasticsearch:", res.Status())

	// Start scroll
	scrollTime := time.Minute
	size := 1000
	req := esapi.SearchRequest{
		Index:  []string{"torrents"},
		Scroll: scrollTime,
		Size:   &size,
	}

	res, err = req.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("Search Error: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Fatalf("Search response error: %s", res.String())
	}

	var searchRes SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchRes); err != nil {
		log.Fatalf("JSON decode Error: %s", err)
	}

	total := 0
	scrollID := searchRes.ScrollID

	for len(searchRes.Hits.Hits) > 0 {
		for _, hit := range searchRes.Hits.Hits {
			src := hit.Source
			// Format: infohash,name,length(bytes),seeders,leechers,completed
			record := []string{
				src.InfoHash,
				src.Title,
				strconv.FormatInt(src.Size, 10),
				"1", // Seeders set to 1 so torrent-paradise frontend doesn't filter it out
				"0", // Leechers
				"0", // Completed
			}
			if err := writer.Write(record); err != nil {
				log.Printf("CSV Write Error: %v", err)
			}
			total++
		}

		scrollReq := esapi.ScrollRequest{
			ScrollID: scrollID,
			Scroll:   scrollTime,
		}
		res, err = scrollReq.Do(context.Background(), es)
		if err != nil {
			log.Fatalf("Scroll Error: %s", err)
		}
		
		if res.IsError() {
			log.Fatalf("Scroll response error: %s", res.String())
		}

		if err := json.NewDecoder(res.Body).Decode(&searchRes); err != nil {
			log.Fatalf("JSON decode Error: %s", err)
		}
		scrollID = searchRes.ScrollID
		res.Body.Close()
	}

	fmt.Printf("Successfully exported %d records to %s\n", total, outPath)
}
