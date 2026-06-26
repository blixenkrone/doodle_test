package znhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func EncodeJSON[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("error encoding json: %w", err)
	}
	return nil
}

func DecodeJSON[T any](r io.Reader) (T, error) {
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return v, fmt.Errorf("error decoding json: %w", err)
	}
	return v, nil
}

// TODO: create some error type that prevents info leakage
func JSONError(w http.ResponseWriter, status int, errs ...error) error {
	resp := HTTPError{
		Code:    status,
		Message: http.StatusText(status),
	}
	if len(errs) > 0 {
		resp.Message = strings.ReplaceAll(errors.Join(errs...).Error(), "\n", ", ")
	}
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return nil
	}
	return json.NewEncoder(w).Encode(&resp)
}

// Same as JSONError but panics if there's an error
func MustJSONError(w http.ResponseWriter, status int, errs ...error) {
	if err := JSONError(w, status, errs...); err != nil {
		panic(err)
	}
}

// HTTPError example
// @Description Error response
type HTTPError struct {
	Code    int    `json:"code,omitzero" example:"400"`
	Message string `json:"message,omitzero" example:"bad input"`
}
