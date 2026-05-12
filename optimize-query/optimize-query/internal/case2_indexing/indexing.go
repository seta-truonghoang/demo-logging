package case2_indexing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/truonghoang/optimize-query-demo/internal/helper"
)

// Demo executes the test related to Indexing
func Demo(db *sql.DB) {
	fmt.Println("\n=======================================================")
	fmt.Println(" CASE 2: INDEXING (Email Search)")
	fmt.Println("=======================================================")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := "SELECT id, name, email FROM users WHERE email = 'user9999@example.com'"

	// BEFORE INDEX
	fmt.Println("-> EXPLAIN ANALYZE WITHOUT Index:")
	helper.ExplainQuery(ctx, db, query)
	
	start := time.Now()
	_, _ = db.ExecContext(ctx, query)
	fmt.Printf("[Search Without Index] Took: %v\n", time.Since(start))

	// CREATE INDEX
	fmt.Println("\n-> Running CREATE INDEX idx_users_email ON users(email) ...")
	_, err := db.ExecContext(ctx, "CREATE INDEX idx_users_email ON users(email)")
	if err != nil {
		log.Fatal(err)
	}

	// AFTER INDEX
	fmt.Println("\n-> EXPLAIN ANALYZE WITH Index:")
	helper.ExplainQuery(ctx, db, query)

	start = time.Now()
	_, _ = db.ExecContext(ctx, query)
	fmt.Printf("[Search With Index] Took: %v\n", time.Since(start))
}
