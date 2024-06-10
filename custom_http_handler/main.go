package main

import (
	"encoding/json"
	"net/http"
	"regexp"
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
	return NewApiError(http.StatusNotFound, "user not found")
}

type request struct {
	Email string `json:"email"`
}

func (r request) Validate() map[string]string {
	if r.Email == "" {
		return map[string]string{"email": "this field is required"}
	}

	if !isEmailValid(r.Email) {
		return map[string]string{"email": "invalid email"}
	}

	return nil
}

func isEmailValid(e string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return emailRegex.MatchString(e)
}

func JSONValidatePostRequest(w http.ResponseWriter, r *http.Request) error {
	var req request
	_ = json.NewDecoder(r.Body).Decode(&req)

	errMap := req.Validate()
	if errMap != nil {
		return BadRequestError(errMap)
	}

	WriteString(w, http.StatusOK, "validation successful")
	return nil
}

func main() {
	http.Handle("GET /", MakeHttpHandler(Index))
	http.Handle("GET /json", MakeHttpHandler(JSONResponse))
	http.Handle("GET /json/error", MakeHttpHandler(JSONErrorResponse))
	http.Handle("POST /validate", MakeHttpHandler(JSONValidatePostRequest))

	http.ListenAndServe(":8080", nil)
}
