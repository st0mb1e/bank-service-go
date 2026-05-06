package repo

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"
	"github.com/st0mb1e/bank-service-go/dao/entity"
)

type TransactionRepo interface {
	Insert(ctx context.Context, q SQLQuerier, accountID string, amount decimal.Decimal, txType string, counterparty *string, creditID *string, cardID *string) (*entity.Transaction, error)
	SumByUserMonth(ctx context.Context, userID, yearMonth string) (income, expense decimal.Decimal, err error)
	ListByUserSince(ctx context.Context, userID string, since string) ([]entity.Transaction, error)
}

type transactionRepo struct {
	db *sql.DB
}

func NewTransactionRepo(db *sql.DB) TransactionRepo {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) Insert(ctx context.Context, q SQLQuerier, accountID string, amount decimal.Decimal, txType string, counterparty *string, creditID *string, cardID *string) (*entity.Transaction, error) {
	row := q.QueryRowContext(ctx, `
		INSERT INTO transactions (account_id, amount, tx_type, counterparty_account_id, credit_id, card_id)
		VALUES ($1, $2::numeric, $3, $4, $5, $6)
		RETURNING id, account_id, amount::text, tx_type, counterparty_account_id, credit_id, card_id, created_at`,
		accountID, amount, txType, counterparty, creditID, cardID,
	)
	return scanTransaction(row)
}

func scanTransaction(row *sql.Row) (*entity.Transaction, error) {
	var t entity.Transaction
	var amt string
	var cp, cr, ca sql.NullString
	if err := row.Scan(&t.ID, &t.AccountID, &amt, &t.TxType, &cp, &cr, &ca, &t.CreatedAt); err != nil {
		return nil, err
	}
	d, err := decimal.NewFromString(amt)
	if err != nil {
		return nil, err
	}
	t.Amount = d
	if cp.Valid {
		s := cp.String
		t.CounterpartyAccountID = &s
	}
	if cr.Valid {
		s := cr.String
		t.CreditID = &s
	}
	if ca.Valid {
		s := ca.String
		t.CardID = &s
	}
	return &t, nil
}

func (r *transactionRepo) SumByUserMonth(ctx context.Context, userID, yearMonth string) (income, expense decimal.Decimal, err error) {
	row := r.db.QueryRowContext(ctx, `
		WITH m AS (
			SELECT t.id, t.account_id, t.amount, t.tx_type
			FROM transactions t
			JOIN accounts a ON a.id = t.account_id
			WHERE a.user_id = $1::uuid
			  AND to_char(t.created_at AT TIME ZONE 'UTC', 'YYYY-MM') = $2
		)
		SELECT
			COALESCE(SUM(CASE WHEN tx_type IN ('deposit','transfer_in','credit_disbursement') THEN amount ELSE 0 END), 0)::text,
			COALESCE(SUM(CASE WHEN tx_type IN ('withdraw','transfer_out','card_payment','credit_payment','penalty') THEN amount ELSE 0 END), 0)::text
		FROM m`,
		userID, yearMonth,
	)
	var incStr, expStr string
	if err = row.Scan(&incStr, &expStr); err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	income, err = decimal.NewFromString(incStr)
	if err != nil {
		return
	}
	expense, err = decimal.NewFromString(expStr)
	return
}

func (r *transactionRepo) ListByUserSince(ctx context.Context, userID string, since string) ([]entity.Transaction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.account_id, t.amount::text, t.tx_type, t.counterparty_account_id, t.credit_id, t.card_id, t.created_at
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		WHERE a.user_id = $1::uuid AND t.created_at >= $2::timestamptz
		ORDER BY t.created_at`,
		userID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []entity.Transaction
	for rows.Next() {
		var t entity.Transaction
		var amt string
		var cp, cr, ca sql.NullString
		if err := rows.Scan(&t.ID, &t.AccountID, &amt, &t.TxType, &cp, &cr, &ca, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Amount, err = decimal.NewFromString(amt)
		if err != nil {
			return nil, err
		}
		if cp.Valid {
			s := cp.String
			t.CounterpartyAccountID = &s
		}
		if cr.Valid {
			s := cr.String
			t.CreditID = &s
		}
		if ca.Valid {
			s := ca.String
			t.CardID = &s
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
