package service

import (
	"context"
	"strings"

	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/dto"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req dto.LoginRequest, jwtSecret []byte) (*dto.LoginResponse, error)
}

type authService struct {
	userRepo repo.UserRepo
}

func NewAuthService(userRepo repo.UserRepo) AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)
	if err := dto.ValidateRegisterFields(req.Email, req.Username, req.Password); err != nil {
		return nil, ErrValidation
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.AddUser(ctx, req.Email, req.Username, string(hash))
	if err != nil {
		if repo.IsUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &dto.RegisterResponse{ID: user.ID, Email: user.Email, Username: user.Username}, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest, jwtSecret []byte) (*dto.LoginResponse, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if len(jwtSecret) == 0 {
		return nil, ErrValidation
	}
	if err := dto.ValidateLoginFields(req.Email, req.Password); err != nil {
		return nil, ErrValidation
	}
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrUnauthorized
	}
	token, err := SignJWT(jwtSecret, user.ID)
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{Token: token, ExpiresInHours: 24}, nil
}
