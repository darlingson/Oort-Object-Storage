package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

type UserHandler struct {
	userService *services.UserService
	permRepo    repositories.PermissionRepository
}

func NewUserHandler(
	userService *services.UserService,
	permRepo repositories.PermissionRepository,
) *UserHandler {

	return &UserHandler{
		userService: userService,
		permRepo:    permRepo,
	}
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *UserHandler) CreateUser(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	user, err := h.userService.Create(
		r.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusConflict,
		)
		return
	}

	resp := UserResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

type GrantPermissionRequest struct {
	Permission string `json:"permission"`
}

func (h *UserHandler) GrantPermission(
	w http.ResponseWriter,
	r *http.Request,
) {

	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(
			w,
			"invalid user id",
			http.StatusBadRequest,
		)
		return
	}

	_, err = h.userService.FindByID(r.Context(), userID)
	if err != nil {
		http.Error(
			w,
			"user not found",
			http.StatusNotFound,
		)
		return
	}

	var req GrantPermissionRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	perm, err := h.permRepo.FindByName(r.Context(), req.Permission)
	if err != nil {
		http.Error(
			w,
			"permission not found",
			http.StatusNotFound,
		)
		return
	}

	err = h.userService.AssignPermission(
		r.Context(),
		userID,
		perm.ID,
	)
	if err != nil {
		http.Error(
			w,
			"internal error",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}
