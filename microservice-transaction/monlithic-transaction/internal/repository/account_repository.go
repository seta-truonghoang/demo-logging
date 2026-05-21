package repository

import (
	"context"
	"errors"
	"fmt"

	"monlithic-transaction/internal/model"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Count(ctx context.Context, db *gorm.DB) (int64, error) {
	var count int64
	if err := db.WithContext(ctx).Model(&model.Account{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count accounts: %w", err)
	}
	return count, nil
}

func (r *AccountRepository) CreateMany(ctx context.Context, db *gorm.DB, accounts []model.Account) error {
	if len(accounts) == 0 {
		return nil
	}
	if err := db.WithContext(ctx).Create(&accounts).Error; err != nil {
		return fmt.Errorf("create accounts: %w", err)
	}
	return nil
}

func (r *AccountRepository) List(ctx context.Context, db *gorm.DB) ([]model.Account, error) {
	var accounts []model.Account
	if err := db.WithContext(ctx).Order("name asc").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

func (r *AccountRepository) FindByName(ctx context.Context, db *gorm.DB, name string) (model.Account, error) {
	var account model.Account
	if err := db.WithContext(ctx).Where("name = ?", name).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Account{}, fmt.Errorf("account %q not found", name)
		}
		return model.Account{}, fmt.Errorf("find account %q: %w", name, err)
	}
	return account, nil
}

func (r *AccountRepository) Save(ctx context.Context, db *gorm.DB, account *model.Account) error {
	if err := db.WithContext(ctx).Save(account).Error; err != nil {
		return fmt.Errorf("save account %q: %w", account.Name, err)
	}
	return nil
}
