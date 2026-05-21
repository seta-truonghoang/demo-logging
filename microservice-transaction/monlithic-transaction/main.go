package main

import (
	"log"
	"os"

	"monlithic-transaction/internal/db"
	"monlithic-transaction/internal/handler"
	"monlithic-transaction/internal/repository"
	"monlithic-transaction/internal/service"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=demo password=demo dbname=demo port=5432 sslmode=disable TimeZone=UTC"
	}

	database, err := db.NewPostgresDB(dsn)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewAccountRepository(database)
	svc := service.NewAccountService(database, repo)
	if err := svc.SeedAccounts(); err != nil {
		log.Fatal(err)
	}

	router := handler.NewAccountHandler(svc).Router()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("transaction demo server listening on :%s", port)
	log.Fatal(router.Run(":" + port))
}
