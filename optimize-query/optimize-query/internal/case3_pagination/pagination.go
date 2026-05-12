package case3_pagination

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Demo executes the Offset vs Keyset Pagination test
func Demo(db *sql.DB) {
	fmt.Println("\n=======================================================")
	fmt.Println(" CASE 3: OFFSET VS KEYSET (CURSOR) PAGINATION")
	fmt.Println("=======================================================")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Offset Pagination
	start := time.Now()
	// Very slow because the DB has to scan and discard 900,000 records
	rows, err := db.QueryContext(ctx, "SELECT id, title FROM posts ORDER BY id ASC LIMIT 20 OFFSET 900000")
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for rows.Next() {
		count++
	}
	rows.Close()
	fmt.Printf("[Offset Pagination LIMIT 20 OFFSET 900000] Took: %v (Rows: %d)\n", time.Since(start), count)

	// 2. Keyset (Cursor) Pagination
	start = time.Now()
	// Extremely fast because it leverages the B-Tree Index of the Primary Key
	rows2, err := db.QueryContext(ctx, "SELECT id, title FROM posts WHERE id > 900000 ORDER BY id ASC LIMIT 20")
	if err != nil {
		log.Fatal(err)
	}
	count2 := 0
	for rows2.Next() {
		count2++
	}
	rows2.Close()
	fmt.Printf("[Cursor Pagination WHERE id > 900000 LIMIT 20] Took: %v (Rows: %d)\n", time.Since(start), count2)
}
