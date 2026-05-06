package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/st0mb1e/bank-service-go/dto"
	"github.com/st0mb1e/bank-service-go/httputil"
	"github.com/st0mb1e/bank-service-go/service"
)

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}
	res, err := h.authSvc.Login(r.Context(), req, h.jwtSecret)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			httputil.ErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httputil.ErrorJSON(w, http.StatusBadRequest, "validation failed")
			return
		}
		httputil.ErrorJSON(w, http.StatusInternalServerError, "login failed")
		return
	}
	httputil.JSON(w, http.StatusOK, res)
}
