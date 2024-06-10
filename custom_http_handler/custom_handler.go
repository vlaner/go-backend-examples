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

func NewApiError(statusCode int, data any) APIError {
	return APIError{
		Code:    statusCode,
		Message: data,
	}
}

func (e APIError) Error() string {
	return fmt.Sprintf("api error: %d", e.Code)
}

func BadRequestError(errors map[string]string) APIError {
	return APIError{
		Code:    http.StatusBadRequest,
		Message: errors,
	}
}
