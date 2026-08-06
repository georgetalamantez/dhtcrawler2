package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

var dbDir = "../.db"

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
	// Process oldest first based on filename alphabetical order (which matches YYYY-MM-DD pattern usually)
	sort.Strings(files)
	return files
}

func main() {
	start := time.Now()
	files := getDBFiles()

	if len(files) == 0 {
		fmt.Println("No .db files found.")
		return
	}

	fmt.Printf("Found %d DB files to process for deduplication.\n", len(files))

	seenHashes := make(map[string]struct{})
	totalScanned := 0
	totalDeleted := 0

	for _, file := range files {
		fmt.Printf("Processing %s...\n", filepath.Base(file))

		db, err := bbolt.Open(file, 0666, &bbolt.Options{ReadOnly: false})
		if err != nil {
			log.Printf("Could not open %s: %v", file, err)
			continue
		}

		scannedInFile := 0
		deletedInFile := 0

		err = db.Update(func(tx *bbolt.Tx) error {
			root := tx.Bucket([]byte("root"))
			if root == nil {
				return nil
			}

			cursor := root.Cursor()
			for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
				hashStr := string(k)
				scannedInFile++

				if _, exists := seenHashes[hashStr]; exists {
					// Duplicate found, delete it!
					if err := root.Delete(k); err != nil {
						return fmt.Errorf("failed to delete duplicate %s: %v", hashStr, err)
					}
					deletedInFile++
				} else {
					// First time seeing this hash
					seenHashes[hashStr] = struct{}{}
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("Error during deduplication in %s: %v", file, err)
		}
		db.Close()

		fmt.Printf("  -> Scanned: %d | Deleted (Duplicates): %d\n", scannedInFile, deletedInFile)
		totalScanned += scannedInFile
		totalDeleted += deletedInFile
	}

	fmt.Println("==================================================")
	fmt.Printf("Deduplication Complete in %v!\n", time.Since(start))
	fmt.Printf("Total Records Scanned: %d\n", totalScanned)
	fmt.Printf("Total Duplicates Removed: %d\n", totalDeleted)
	fmt.Printf("Unique Torrents Kept: %d\n", len(seenHashes))
}
