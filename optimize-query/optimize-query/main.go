package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/truonghoang/optimize-query-demo/internal/case1_nplusone"
	"github.com/truonghoang/optimize-query-demo/internal/case2_indexing"
	"github.com/truonghoang/optimize-query-demo/internal/case3_pagination"
	"github.com/truonghoang/optimize-query-demo/internal/case4_connpool"
)

const dbDSN = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [case1|case2|case3|case4|all]")
		fmt.Println("  case1 : N+1 Query vs JOIN")
		fmt.Println("  case2 : Indexing")
		fmt.Println("  case3 : Offset vs Cursor Pagination")
		fmt.Println("  case4 : Connection Pool Tuning")
		fmt.Println("  all   : Run all cases")
		return
	}

	mode := strings.ToLower(os.Args[1])

	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	defer db.Close()
	
	// Clean up index if you want to run this file multiple times (idempotent)
	cleanupIndexes(db)

	if mode == "case1" || mode == "all" {
		case1_nplusone.Demo(db)
	}

	if mode == "case2" || mode == "all" {
		case2_indexing.Demo(db)
	}

	if mode == "case3" || mode == "all" {
		case3_pagination.Demo(db)
	}

	if mode == "case4" || mode == "all" {
		case4_connpool.Demo(db)
	}
}

func cleanupIndexes(db *sql.DB) {
	_, _ = db.Exec("DROP INDEX IF EXISTS idx_users_email")
}
