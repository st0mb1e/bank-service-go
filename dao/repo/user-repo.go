package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/st0mb1e/bank-service-go/dao/entity"
)

type UserRepo interface {
	AddUser(ctx context.Context, email, username, passwordHash string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id string) (*entity.User, error)
}

type userRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) UserRepo {
	return &userRepo{db: db}
}

func (repo *userRepo) AddUser(ctx context.Context, email, username, passwordHash string) (*entity.User, error) {
	row := repo.db.QueryRowContext(ctx,
		`INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, email, username, password_hash, created_at`,
		email,
		username,
		passwordHash,
	)

	user := &entity.User{}
	if err := row.Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		return nil, err
	}
	return user, nil
}

func (repo *userRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	row := repo.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, created_at FROM users WHERE email = $1`,
		email,
	)
	user := &entity.User{}
	if err := row.Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (repo *userRepo) GetByID(ctx context.Context, id string) (*entity.User, error) {
	row := repo.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, created_at FROM users WHERE id = $1`,
		id,
	)
	user := &entity.User{}
	if err := row.Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}
