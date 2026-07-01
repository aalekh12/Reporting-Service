// Package response centralizes JSON response/error encoding so handlers
// stay thin.
package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"reporting-service/internal/domain"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes v as a JSON body with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error maps a domain error (or any error) to the right HTTP status and
// writes a consistent {"error": {...}} envelope.
func Error(w http.ResponseWriter, err error) {
	var domErr *domain.Error
	if errors.As(err, &domErr) {
		status, code := statusFor(domErr.Kind)
		JSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: domErr.Message}})
		return
	}
	JSON(w, http.StatusInternalServerError, errorEnvelope{Error: errorBody{Code: "internal", Message: "internal server error"}})
}

func statusFor(kind domain.ErrorKind) (int, string) {
	switch kind {
	case domain.KindValidation:
		return http.StatusBadRequest, "validation_error"
	case domain.KindNotFound:
		return http.StatusNotFound, "not_found"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
