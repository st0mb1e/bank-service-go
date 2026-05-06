package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/st0mb1e/bank-service-go/dto"
	"github.com/st0mb1e/bank-service-go/httputil"
	"github.com/st0mb1e/bank-service-go/service"
)

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.ErrorJSON(w, http.StatusBadRequest, "invalid json")
		return
	}

	res, err := h.authSvc.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			httputil.ErrorJSON(w, http.StatusConflict, "email or username already taken")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			httputil.ErrorJSON(w, http.StatusBadRequest, "validation failed")
			return
		}
		httputil.ErrorJSON(w, http.StatusInternalServerError, "registration failed")
		return
	}

	httputil.JSON(w, http.StatusCreated, res)
}
