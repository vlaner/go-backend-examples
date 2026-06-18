package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /users/{id}", h.getUser)
	mux.HandleFunc("PATCH /users/{id}", h.patchUser)
}

func (h *UserHandler) getUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}

	user, err := h.service.Find(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("find user: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, mapUserResponse(user))
}

func (h *UserHandler) patchUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUserID(w, r)
	if !ok {
		return
	}

	var req PatchUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}

	user, err := h.service.UpdateBio(r.Context(), id, req.ToCommand())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("patch user: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, mapUserResponse(user))
}

func parseUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return 0, false
	}

	return id, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
