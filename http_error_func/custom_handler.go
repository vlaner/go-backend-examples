package main

import (
	"fmt"
	"log"
	"net/http"
)

type HttpHandlerFunc func(w http.ResponseWriter, r *http.Request) error

func MakeHttpHandler(fn HttpHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			if e, ok := err.(APIError); ok {
				WriteJSON(w, e.Code, e)
			} else {
				WriteJSON(w, http.StatusInternalServerError, "internal server error")
				log.Println("error handling request:", err)
			}
		}
	}
}

type APIError struct {
	Code    int `json:"-"`
	Message any `json:"error"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("api error")
}

func NewApiError(statusCode int, err error) APIError {
	return APIError{
		Code:    statusCode,
		Message: err.Error(),
	}
}
