package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/atlan/mibo-media-server/internal/auth"
	"github.com/atlan/mibo-media-server/internal/database"
)

func (r *Router) handleListAdminUsers(w http.ResponseWriter, req *http.Request) {
	if !r.requireAdmin(w, req) {
		return
	}

	users, err := r.auth.ListUsers(req.Context())
	if err != nil {
		writeError(req.Context(), w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusOK, adminUserResponses(users))
}

func (r *Router) handleCreateAdminUser(w http.ResponseWriter, req *http.Request) {
	if !r.requireAdmin(w, req) {
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decodeJSON(req, &input); err != nil {
		writeError(req.Context(), w, http.StatusBadRequest, err)
		return
	}

	user, err := r.auth.CreateUser(req.Context(), input.Username, input.Password, input.Role)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, auth.ErrDuplicateUsername) && !errors.Is(err, auth.ErrInvalidRole) {
			status = http.StatusInternalServerError
		}
		writeError(req.Context(), w, status, err)
		return
	}
	writeJSON(req.Context(), w, http.StatusCreated, adminUserResponseFromUser(user))
}

type adminUserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func adminUserResponses(users []database.User) []adminUserResponse {
	responses := make([]adminUserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, adminUserResponseFromUser(user))
	}
	return responses
}

func adminUserResponseFromUser(user database.User) adminUserResponse {
	return adminUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
