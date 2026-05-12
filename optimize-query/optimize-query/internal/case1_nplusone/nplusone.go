package case1_nplusone

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Demo executes tests related to N+1 Query vs JOIN
func Demo(db *sql.DB) {
	fmt.Println("\n=======================================================")
	fmt.Println(" CASE 1: N+1 QUERY VS JOIN")
	fmt.Println("=======================================================")
	demoNPlusOne(db)
	demoJoin(db)
}

func demoNPlusOne(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	start := time.Now()
	
	// Query 1: Fetch 100 users
	rows, err := db.QueryContext(ctx, "SELECT id, name FROM users LIMIT 100")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	postCount := 0
	for rows.Next() {
		var userID int
		var name string
		rows.Scan(&userID, &name)

		// N Queries: Fetch posts for each user (extremely slow because user_id has no index, causing a Seq Scan of the posts table 100 times)
		postRows, err := db.QueryContext(ctx, "SELECT id FROM posts WHERE user_id = $1", userID)
		if err != nil {
			log.Fatal(err)
		}
		for postRows.Next() {
			postCount++
		}
		postRows.Close()
	}
	
	fmt.Printf("[N+1 Query] Took: %v (Found %d posts)\n", time.Since(start), postCount)
}

func demoJoin(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	
	// 1 single Query
	query := `
		SELECT u.id, u.name, p.id 
		FROM users u
		LEFT JOIN posts p ON u.id = p.user_id
		WHERE u.id <= 100
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	postCount := 0
	for rows.Next() {
		postCount++
	}
	
	fmt.Printf("[JOIN Query] Took: %v (Found %d posts)\n", time.Since(start), postCount)
}
