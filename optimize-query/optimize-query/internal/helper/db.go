package helper

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// ExplainQuery executes an EXPLAIN ANALYZE command and prints the result to the terminal
func ExplainQuery(ctx context.Context, db *sql.DB, query string) {
	rows, err := db.QueryContext(ctx, "EXPLAIN ANALYZE "+query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var plan string
		rows.Scan(&plan)
		fmt.Println("  " + plan)
	}
}
