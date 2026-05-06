package auth

import "github.com/st0mb1e/bank-service-go/service"

type AuthHandlers struct {
	authSvc   service.AuthService
	jwtSecret []byte
}

func NewAuthHandlers(authSvc service.AuthService, jwtSecret []byte) *AuthHandlers {
	return &AuthHandlers{authSvc: authSvc, jwtSecret: jwtSecret}
}
