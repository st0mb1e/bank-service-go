package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/st0mb1e/bank-service-go/dao/entity"
)

type CardRepo interface {
	Insert(ctx context.Context, q SQLQuerier, c *entity.Card) error
	GetByIDForUser(ctx context.Context, q SQLQuerier, cardID, userID string) (*entity.Card, error)
	ListByUser(ctx context.Context, userID string) ([]entity.Card, error)
}

type cardRepo struct {
	db *sql.DB
}

func NewCardRepo(db *sql.DB) CardRepo {
	return &cardRepo{db: db}
}

func (r *cardRepo) Insert(ctx context.Context, q SQLQuerier, c *entity.Card) error {
	return q.QueryRowContext(ctx, `
		INSERT INTO cards (user_id, account_id, pan_encrypted, expiry_encrypted, cvv_hash, integrity_hmac, last4)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		c.UserID, c.AccountID, c.PANEncrypted, c.ExpiryEncrypted, c.CVVHash, c.IntegrityHMAC, c.Last4,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *cardRepo) GetByIDForUser(ctx context.Context, q SQLQuerier, cardID, userID string) (*entity.Card, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, user_id, account_id, pan_encrypted, expiry_encrypted, cvv_hash, integrity_hmac, last4, created_at
		FROM cards WHERE id = $1 AND user_id = $2`,
		cardID, userID,
	)
	var c entity.Card
	if err := row.Scan(&c.ID, &c.UserID, &c.AccountID, &c.PANEncrypted, &c.ExpiryEncrypted, &c.CVVHash, &c.IntegrityHMAC, &c.Last4, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *cardRepo) ListByUser(ctx context.Context, userID string) ([]entity.Card, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, account_id, pan_encrypted, expiry_encrypted, cvv_hash, integrity_hmac, last4, created_at
		FROM cards WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []entity.Card
	for rows.Next() {
		var c entity.Card
		if err := rows.Scan(&c.ID, &c.UserID, &c.AccountID, &c.PANEncrypted, &c.ExpiryEncrypted, &c.CVVHash, &c.IntegrityHMAC, &c.Last4, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
