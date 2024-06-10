package main

import (
	"encoding/json"
	"net/http"
)

func WriteString(w http.ResponseWriter, statusCode int, s string) error {
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(s))
	return err
}

func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(&data)
}
