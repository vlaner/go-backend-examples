package main

import (
	"errors"
	"net/http"
)

func Index(w http.ResponseWriter, r *http.Request) error {
	return WriteString(w, http.StatusOK, "hello from index")
}

func JSONResponse(w http.ResponseWriter, r *http.Request) error {
	type response struct {
		ID int `json:"id"`
	}
	return WriteJSON(w, http.StatusOK, response{ID: 123})
}

func JSONErrorResponse(w http.ResponseWriter, r *http.Request) error {
	return NewApiError(http.StatusNotFound, errors.New("user not found"))
}

func main() {
	http.Handle("GET /", MakeHttpHandler(Index))
	http.Handle("GET /json", MakeHttpHandler(JSONResponse))
	http.Handle("GET /json/error", MakeHttpHandler(JSONErrorResponse))

	http.ListenAndServe(":8080", nil)
}
