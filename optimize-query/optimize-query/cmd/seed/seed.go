package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const dbDSN = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

func main() {
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer db.Close()

	// 1. Run the schema.sql file
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("Error reading schema.sql: %v", err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		log.Fatalf("Error executing schema: %v", err)
	}
	fmt.Println("Schema initialized successfully (old data cleared).")

	// 2. Insert 10,000 Users
	fmt.Println("Inserting 10,000 users...")
	insertUsers(db, 10000)

	// 3. Insert 1,000,000 Posts using Worker Pool
	fmt.Println("Inserting 1,000,000 posts using Worker Pool (please wait a few seconds)...")
	start := time.Now()
	insertPostsWithWorkerPool(db, 1000000, 10) // 10 workers for efficient parallel execution
	fmt.Printf("[Seed Data] Completed in: %v\n", time.Since(start))
	fmt.Println("Ready to run the demo (go run main.go)!")
}

func insertUsers(db *sql.DB, count int) {
	tx, _ := db.Begin()
	// Optimize inserting 10,000 rows
	stmt, _ := tx.Prepare("INSERT INTO users (name, email) VALUES ($1, $2)")
	for i := 1; i <= count; i++ {
		stmt.Exec(fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@example.com", i))
	}
	stmt.Close()
	tx.Commit()
}

func insertPostsWithWorkerPool(db *sql.DB, totalPosts int, numWorkers int) {
	var wg sync.WaitGroup
	postsPerWorker := totalPosts / numWorkers
	batchSize := 1000 // Insert 1000 records at a time per worker

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < postsPerWorker; i += batchSize {
				var valueStrings []string
				var valueArgs []interface{}
				for j := 0; j < batchSize; j++ {
					postIndex := workerID*postsPerWorker + i + j + 1
					userID := (postIndex % 10000) + 1 // Link to a random user
					
					// Assign parameters $1, $2, $3...
					offset := j * 3
					valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", offset+1, offset+2, offset+3))
					valueArgs = append(valueArgs, userID, fmt.Sprintf("Title %d", postIndex), fmt.Sprintf("Content %d", postIndex))
				}
				query := fmt.Sprintf("INSERT INTO posts (user_id, title, content) VALUES %s", strings.Join(valueStrings, ","))
				_, err := db.Exec(query, valueArgs...)
				if err != nil {
					log.Printf("Worker %d failed to insert data: %v", workerID, err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
}
