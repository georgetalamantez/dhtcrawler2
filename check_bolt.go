package main

import (
	"fmt"
	"log"
	"os"

	"go.etcd.io/bbolt"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: check_bolt <db-file>")
	}
	db, err := bbolt.Open(os.Args[1], 0600, &bbolt.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			stats := b.Stats()
			fmt.Printf("Bucket: %s, KeyN: %d\n", name, stats.KeyN)
			c := b.Cursor()
			i := 0
			for k, v := c.First(); k != nil && i < 2; k, v = c.Next() {
				fmt.Printf("  Key: %s\n  Val: %s\n", string(k), string(v))
				i++
			}
			return nil
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}
