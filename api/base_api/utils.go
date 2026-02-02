package base_api

import (
	"fmt"
	"net/http"
)

func APIReturnError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error": "%s!"}`, err.Error())))
}
