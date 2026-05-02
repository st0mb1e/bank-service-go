package auth

import "github.com/st0mb1e/bank-service-go/service"

// AuthHandlers обрабатывает HTTP для /auth/* и получает зависимости через конструктор
type AuthHandlers struct {
	authSvc service.AuthService
}

func NewAuthHandlers(authSvc service.AuthService) *AuthHandlers {
	return &AuthHandlers{authSvc: authSvc}
}
