package main

import (
	"encoding/json"
	"log"
	"net/http"
)

var sv *StructValidator

type PostRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"required"`
}

type OkResponse struct {
	Ok bool `json:"ok"`
}

type ErrorResponse struct {
	Errors map[string]string `json:"errors"`
}

func ValidateHandler(w http.ResponseWriter, r *http.Request) {
	var req PostRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("error decoding json:", err)
		return
	}

	acceptLang := r.Header.Get("Accept-Language")
	errMsgs, err := sv.ValidateStruct(req, acceptLang)
	if err != nil {
		log.Println("validation failed:", err)
		return
	}

	if errMsgs != nil {
		errorResponse := ErrorResponse{Errors: errMsgs}
		err := json.NewEncoder(w).Encode(&errorResponse)
		if err != nil {
			log.Println("error encoding json:", err)
		}
		return
	}
	okResponse := OkResponse{Ok: true}
	err = json.NewEncoder(w).Encode(&okResponse)
	if err != nil {
		log.Println("error encoding json:", err)
	}
}

func main() {
	sv = NewStructValidator()
	sv.UseJsonTags()

	http.HandleFunc("POST /", ValidateHandler)
	http.ListenAndServe(":8080", nil)
}
