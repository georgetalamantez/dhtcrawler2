package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/query"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: clover-extractor <path-to-db-dir>")
	}
	dbPath := os.Args[1]

	db, err := clover.Open(dbPath)
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}
	defer db.Close()

	collections, err := db.ListCollections()
	if err != nil {
		log.Fatal("Failed to list collections:", err)
	}

	for _, coll := range collections {
		docs, err := db.FindAll(query.NewQuery(coll))
		if err != nil {
			log.Printf("Error reading collection %s: %v", coll, err)
			continue
		}
		for _, doc := range docs {
			b, err := json.Marshal(doc.AsMap())
			if err != nil {
				log.Printf("Error marshaling document in %s: %v", coll, err)
				continue
			}
			fmt.Printf("%s\t%s\n", coll, string(b))
		}
	}
}
