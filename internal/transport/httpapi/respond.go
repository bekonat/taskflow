package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bekonat/taskflow/internal/domain"
)

type errorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: ve.Message, Field: ve.Field})
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(w, http.StatusConflict, "resource already exists")
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	default:
		h.log.ErrorContext(r.Context(), "internal error", slog.String("error", err.Error()),
			slog.String("path", r.URL.Path))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
