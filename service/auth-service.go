package service

import (
	"github.com/st0mb1e/bank-service-go/dao/repo"
	"github.com/st0mb1e/bank-service-go/dto"
)

type AuthService interface {
	Register(req dto.RegisterRequest) (*dto.RegisterResponse, error)
}

type authService struct {
	userRepo repo.UserRepo
}

func NewAuthService(userRepo repo.UserRepo) AuthService {
	return &authService{userRepo}
}

func (authService *authService) Register(req dto.RegisterRequest) (res *dto.RegisterResponse, err error) {
	user, err := authService.userRepo.AddUser(req.Email, req.Username, req.Password)
	if err != nil {
		return nil, err
	}

	return &dto.RegisterResponse{ID: user.ID, Email: user.Email, Username: user.Username}, nil
}
