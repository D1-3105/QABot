package base_api

import (
	"fmt"
	"net/http"
)

type APIError struct {
	Error string `json:"error"`
}

func APIReturnError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "%s!"}`, err.Error())))
}
