package auth

import (
	"net/http"
)

type RegisterDto struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
