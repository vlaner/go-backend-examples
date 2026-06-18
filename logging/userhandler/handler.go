package userhandler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/vlaner/go-backend-examples/logging/userservice"
)

type Handler struct {
	service userservice.CreateUser
	logger  *slog.Logger
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(service userservice.CreateUser, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger.With(slog.String("component", "http.user")),
	}
}

func (r createUserRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("username", r.Username),
		slog.String("password", "[REDACTED]"),
	)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /users", h.create)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	err := h.createUser(w, r)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create user failed", slog.Any("err", err))
		h.writeError(w, r, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) error {
	var request createUserRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return fmt.Errorf("decode create user request: %w", err)
	}

	h.logger.InfoContext(r.Context(), "decoded create user request", slog.Any("request", request))

	if request.Username == "" || request.Password == "" {
		h.writeError(w, r, http.StatusBadRequest, "username and password are required")
		return nil
	}

	user, err := h.service.CreateUser(r.Context(), userservice.CreateUserCommand{
		Username: request.Username,
		Password: request.Password,
	})
	if err != nil {
		return fmt.Errorf("call user service: %w", err)
	}

	err = writeJSON(w, http.StatusCreated, userResponse{ID: user.ID, Username: user.Username})
	if err != nil {
		return fmt.Errorf("write create user response: %w", err)
	}

	return nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	err := writeJSON(w, statusCode, errorResponse{Error: message})
	if err != nil {
		h.logger.ErrorContext(r.Context(), "write error response failed", slog.Any("err", err))
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return fmt.Errorf("encode json response: %w", err)
	}

	return nil
}
