package auth

import (
	"encoding/json"
	"net/http"

	"github.com/st0mb1e/bank-service-go/dto"
)

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	res, err := h.authSvc.Register(req)
	if err != nil {
		http.Error(w, `{"error":"registration failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}
