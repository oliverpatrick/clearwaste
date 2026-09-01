package login

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxRequestBytes = 4096

type Handler struct {
	service *Service
}

type request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type response struct {
	Ticket      string        `json:"ticket"`
	AccountID   uint64        `json:"accountId"`
	CharacterID uint64        `json:"characterId"`
	World       worldResponse `json:"world"`
}

type worldResponse struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func NewHandler(service *Service) http.Handler {
	return &Handler{service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input request
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	result, err := h.service.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login failed"})
		return
	}
	writeJSON(w, http.StatusOK, response{
		Ticket:      result.World.Ticket,
		AccountID:   uint64(result.AccountID),
		CharacterID: uint64(result.CharacterID),
		World:       worldResponse{Host: result.World.Host, Port: result.World.Port},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
