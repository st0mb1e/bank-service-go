package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/st0mb1e/bank-service-go/dao/entity"
	"github.com/st0mb1e/bank-service-go/dao/repo"
)

type AccountService interface {
	Create(ctx context.Context, userID string) (*entity.Account, error)
	List(ctx context.Context, userID string) ([]entity.Account, error)
	Deposit(ctx context.Context, userID, accountID string, amount decimal.Decimal) error
	Withdraw(ctx context.Context, userID, accountID string, amount decimal.Decimal) error
}

type accountService struct {
	db      *sql.DB
	accRepo repo.AccountRepo
	txRepo  repo.TransactionRepo
}

func NewAccountService(db *sql.DB, accRepo repo.AccountRepo, txRepo repo.TransactionRepo) AccountService {
	return &accountService{db: db, accRepo: accRepo, txRepo: txRepo}
}

func (s *accountService) Create(ctx context.Context, userID string) (*entity.Account, error) {
	return s.accRepo.Create(ctx, s.db, userID)
}

func (s *accountService) List(ctx context.Context, userID string) ([]entity.Account, error) {
	return s.accRepo.ListByUser(ctx, userID)
}

func (s *accountService) Deposit(ctx context.Context, userID, accountID string, amount decimal.Decimal) error {
	if !amount.GreaterThan(decimal.Zero) {
		return ErrValidation
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	a, err := s.accRepo.GetByIDForUser(ctx, tx, accountID, userID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrForbidden
	}
	if _, err := s.accRepo.AddBalance(ctx, tx, accountID, amount); err != nil {
		return err
	}
	if _, err := s.txRepo.Insert(ctx, tx, accountID, amount, "deposit", nil, nil, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *accountService) Withdraw(ctx context.Context, userID, accountID string, amount decimal.Decimal) error {
	if !amount.GreaterThan(decimal.Zero) {
		return ErrValidation
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	a, err := s.accRepo.GetByIDForUser(ctx, tx, accountID, userID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrForbidden
	}
	if _, err := s.accRepo.AddBalance(ctx, tx, accountID, amount.Neg()); err != nil {
		if errors.Is(err, repo.ErrInsufficientFunds) {
			return ErrInsufficient
		}
		return err
	}
	if _, err := s.txRepo.Insert(ctx, tx, accountID, amount, "withdraw", nil, nil, nil); err != nil {
		return err
	}
	return tx.Commit()
}
