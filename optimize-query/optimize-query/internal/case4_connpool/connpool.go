package case4_connpool

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	DirectDSN    = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	PgBouncerDSN = "postgres://postgres:postgres@localhost:6432/postgres?sslmode=disable"
)

// Demo executes the PgBouncer and Connection Pool Tuning test
func Demo(db *sql.DB) {
	fmt.Println("\n=======================================================")
	fmt.Println(" CASE 4: PGBOUNCER & CONNECTION POOLING")
	fmt.Println("=======================================================")

	fmt.Println("\n--- Test 1: Connect DIRECTLY to PostgreSQL (Port 5432) ---")
	fmt.Println("Spawning 500 concurrent connections (Postgres max_connections is 100 by default).")
	testConcurrentConnections(DirectDSN, 500)

	fmt.Println("\n--- Test 2: Connect via PgBouncer (Port 6432) ---")
	fmt.Println("Spawning 500 concurrent connections. PgBouncer handles it easily!")
	testConcurrentConnections(PgBouncerDSN, 1000)

	fmt.Println("\n=> EXPLANATION: ")
	fmt.Println("   - Direct Postgres: Fails when exceeding `max_connections` (we tried 150, standard is 100).")
	fmt.Println("   - PgBouncer: Queues and multiplexes 500 client connections over a small pool of 20 real DB connections smoothly.")
}

func testConcurrentConnections(dsn string, numQueries int) {
	testDB, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		return
	}
	defer testDB.Close()

	// Force the Go driver to actually open 'numQueries' connections simultaneously
	testDB.SetMaxOpenConns(numQueries)
	testDB.SetMaxIdleConns(numQueries)

	var wg sync.WaitGroup
	var successCount int
	var errorCount int
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < numQueries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 5-second timeout for quick failure detection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Simulate a query taking 10ms of I/O
			_, err := testDB.ExecContext(ctx, "SELECT pg_sleep(0.01)")

			mu.Lock()
			if err != nil {
				if errorCount == 0 {
					// Print the first error as an example
					fmt.Printf("   [Error example]: %v\n", err)
				}
				errorCount++
			} else {
				successCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	fmt.Printf("   Result: %d succeeded, %d failed. Took: %v\n", successCount, errorCount, time.Since(start))
}
