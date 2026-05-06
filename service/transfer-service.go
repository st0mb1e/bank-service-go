package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/st0mb1e/bank-service-go/dao/repo"
)

type TransferService interface {
	Transfer(ctx context.Context, userID, fromAccountID, toAccountID string, amount decimal.Decimal) error
}

type transferService struct {
	db      *sql.DB
	accRepo repo.AccountRepo
	txRepo  repo.TransactionRepo
}

func NewTransferService(db *sql.DB, accRepo repo.AccountRepo, txRepo repo.TransactionRepo) TransferService {
	return &transferService{db: db, accRepo: accRepo, txRepo: txRepo}
}

func (s *transferService) Transfer(ctx context.Context, userID, fromAccountID, toAccountID string, amount decimal.Decimal) error {
	if !amount.GreaterThan(decimal.Zero) {
		return ErrValidation
	}
	if fromAccountID == toAccountID {
		return ErrValidation
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	fromAcc, err := s.accRepo.GetByIDForUser(ctx, tx, fromAccountID, userID)
	if err != nil {
		return err
	}
	if fromAcc == nil {
		return ErrForbidden
	}
	toAcc, err := s.accRepo.GetByID(ctx, tx, toAccountID)
	if err != nil {
		return err
	}
	if toAcc == nil {
		return ErrNotFound
	}

	if _, err := s.accRepo.AddBalance(ctx, tx, fromAccountID, amount.Neg()); err != nil {
		if errors.Is(err, repo.ErrInsufficientFunds) {
			return ErrInsufficient
		}
		return err
	}
	if _, err := s.accRepo.AddBalance(ctx, tx, toAccountID, amount); err != nil {
		return err
	}
	cp := toAccountID
	if _, err := s.txRepo.Insert(ctx, tx, fromAccountID, amount, "transfer_out", &cp, nil, nil); err != nil {
		return err
	}
	cp2 := fromAccountID
	if _, err := s.txRepo.Insert(ctx, tx, toAccountID, amount, "transfer_in", &cp2, nil, nil); err != nil {
		return err
	}
	return tx.Commit()
}
