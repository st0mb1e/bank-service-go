package auth

import "net/http"

type LoginDto struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}
