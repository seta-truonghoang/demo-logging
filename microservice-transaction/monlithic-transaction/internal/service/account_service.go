package service

import (
	"context"
	"errors"
	"fmt"

	"monlithic-transaction/internal/model"
	"monlithic-transaction/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransferRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
}

type TransferResponse struct {
	Message string        `json:"message"`
	From    model.Account `json:"from"`
	To      model.Account `json:"to"`
}

type AccountService struct {
	db   *gorm.DB
	repo *repository.AccountRepository
}

func NewAccountService(db *gorm.DB, repo *repository.AccountRepository) *AccountService {
	return &AccountService{db: db, repo: repo}
}

func (s *AccountService) SeedAccounts() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		count, err := s.repo.Count(context.Background(), tx)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		return s.repo.CreateMany(context.Background(), tx, []model.Account{
			{Name: "alice", Balance: 1000},
			{Name: "bob", Balance: 500},
		})
	})
}

func (s *AccountService) ListAccounts(ctx context.Context) ([]model.Account, error) {
	return s.repo.List(ctx, s.db)
}

func (s *AccountService) Transfer(ctx context.Context, req TransferRequest) (TransferResponse, error) {
	if req.Amount <= 0 {
		return TransferResponse{}, errors.New("amount must be greater than zero")
	}
	if req.From == req.To {
		return TransferResponse{}, errors.New("from and to accounts must be different")
	}

	var response TransferResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fromAccount, err := s.repo.FindByName(ctx, tx.Clauses(clause.Locking{Strength: "UPDATE"}), req.From)
		if err != nil {
			return err
		}

		toAccount, err := s.repo.FindByName(ctx, tx.Clauses(clause.Locking{Strength: "UPDATE"}), req.To)
		if err != nil {
			return err
		}

		if fromAccount.Balance < req.Amount {
			return fmt.Errorf("insufficient balance in %q", req.From)
		}

		fromAccount.Balance -= req.Amount
		toAccount.Balance += req.Amount

		if err := s.repo.Save(ctx, tx, &fromAccount); err != nil {
			return err
		}
		if err := s.repo.Save(ctx, tx, &toAccount); err != nil {
			return err
		}

		response = TransferResponse{
			Message: "transfer completed",
			From:    fromAccount,
			To:      toAccount,
		}
		return nil
	})

	return response, err
}
