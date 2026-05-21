package db

import (
	"fmt"

	"monlithic-transaction/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(dsn string) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := database.AutoMigrate(&model.Account{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return database, nil
}
