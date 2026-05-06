package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/st0mb1e/bank-service-go/dao/entity"
	"github.com/shopspring/decimal"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type AccountRepo interface {
	Create(ctx context.Context, q SQLQuerier, userID string) (*entity.Account, error)
	GetByID(ctx context.Context, q SQLQuerier, id string) (*entity.Account, error)
	GetByIDForUser(ctx context.Context, q SQLQuerier, accountID, userID string) (*entity.Account, error)
	ListByUser(ctx context.Context, userID string) ([]entity.Account, error)
	AddBalance(ctx context.Context, q SQLQuerier, accountID string, delta decimal.Decimal) (decimal.Decimal, error)
}

func (r *accountRepo) AddBalance(ctx context.Context, q SQLQuerier, accountID string, delta decimal.Decimal) (decimal.Decimal, error) {
	row := q.QueryRowContext(ctx, `
		UPDATE accounts SET balance = balance + $2::numeric
		WHERE id = $1 AND balance + $2::numeric >= 0
		RETURNING balance::text`,
		accountID, delta,
	)
	var bal string
	if err := row.Scan(&bal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return decimal.Zero, ErrInsufficientFunds
		}
		return decimal.Zero, err
	}
	return decimal.NewFromString(bal)
}

type accountRepo struct {
	db *sql.DB
}

func NewAccountRepo(db *sql.DB) AccountRepo {
	return &accountRepo{db: db}
}

func (r *accountRepo) Create(ctx context.Context, q SQLQuerier, userID string) (*entity.Account, error) {
	row := q.QueryRowContext(ctx,
		`INSERT INTO accounts (user_id) VALUES ($1)
		 RETURNING id, user_id, balance::text, created_at`,
		userID,
	)
	return scanAccount(row)
}

func (r *accountRepo) GetByID(ctx context.Context, q SQLQuerier, id string) (*entity.Account, error) {
	row := q.QueryRowContext(ctx,
		`SELECT id, user_id, balance::text, created_at FROM accounts WHERE id = $1`,
		id,
	)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *accountRepo) GetByIDForUser(ctx context.Context, q SQLQuerier, accountID, userID string) (*entity.Account, error) {
	row := q.QueryRowContext(ctx,
		`SELECT id, user_id, balance::text, created_at
		 FROM accounts WHERE id = $1 AND user_id = $2`,
		accountID, userID,
	)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (r *accountRepo) ListByUser(ctx context.Context, userID string) ([]entity.Account, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, balance::text, created_at FROM accounts WHERE user_id = $1 ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.Account
	for rows.Next() {
		var a entity.Account
		var bal string
		if err := rows.Scan(&a.ID, &a.UserID, &bal, &a.CreatedAt); err != nil {
			return nil, err
		}
		d, err := decimal.NewFromString(bal)
		if err != nil {
			return nil, err
		}
		a.Balance = d
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanAccount(row *sql.Row) (*entity.Account, error) {
	var a entity.Account
	var bal string
	if err := row.Scan(&a.ID, &a.UserID, &bal, &a.CreatedAt); err != nil {
		return nil, err
	}
	d, err := decimal.NewFromString(bal)
	if err != nil {
		return nil, err
	}
	a.Balance = d
	return &a, nil
}
