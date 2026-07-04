package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/darlingson/Oort-Object-Storage/internal/services"
)

type AuthHandler struct {
	userService *services.UserService
	jwtService  *services.JWTService
}

func NewAuthHandler(
	userService *services.UserService,
	jwtService *services.JWTService,
) *AuthHandler {

	return &AuthHandler{
		userService: userService,
		jwtService:  jwtService,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (h *AuthHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {

	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	user, err := h.userService.Login(
		r.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		http.Error(
			w,
			"invalid credentials",
			http.StatusUnauthorized,
		)
		return
	}

	permissions, err := h.userService.GetPermissions(
		r.Context(),
		user.ID,
	)
	if err != nil {
		http.Error(
			w,
			"internal error",
			http.StatusInternalServerError,
		)
		return
	}

	permNames := make([]string, len(permissions))
	for i, p := range permissions {
		permNames[i] = p.Name
	}

	token, err := h.jwtService.Create(user, permNames)
	if err != nil {
		http.Error(
			w,
			"internal error",
			http.StatusInternalServerError,
		)
		return
	}

	resp := LoginResponse{
		Token: token,
	}

	resp.User.ID = user.ID.String()
	resp.User.Email = user.Email

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
