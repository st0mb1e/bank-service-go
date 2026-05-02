package repo

import (
	"database/sql"

	"github.com/st0mb1e/bank-service-go/dao/entity"
)

type UserRepo interface {
	AddUser(email string, username string, password string) (*entity.User, error)
}

type userRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) UserRepo {
	return &userRepo{db: db}
}

func (repo *userRepo) AddUser(email string, username string, password string) (*entity.User, error) {
	row := repo.db.QueryRow(
		`INSERT INTO users (email, username, password) VALUES ($1, $2, $3)
		 RETURNING id, email, username, password, created_at, updated_at`,
		email,
		username,
		password,
	)

	user := &entity.User{}
	if err := row.Scan(&user.ID, &user.Email, &user.Username, &user.Password, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}

	return user, nil
}
