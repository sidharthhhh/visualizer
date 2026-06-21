package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/auth"
	"github.com/containerscope/backend/internal/config"
	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
	"github.com/containerscope/backend/internal/validation"
)

type AuthHandler struct {
	store  *store.Store
	config *config.AuthConfig
	logger *slog.Logger
}

func NewAuthHandler(store *store.Store, config *config.AuthConfig, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{store: store, config: config, logger: logger}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = validation.SanitizeString(req.Email)
	req.Name = validation.SanitizeString(req.Name)

	if err := validation.ValidateEmail(req.Email); err != nil {
		writeValidationError(w, err.Field, err.Message)
		return
	}

	if err := validation.ValidateName(req.Name); err != nil {
		writeValidationError(w, err.Field, err.Message)
		return
	}

	if err := validation.ValidatePassword(req.Password); err != nil {
		writeValidationError(w, err.Field, err.Message)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("hashing password", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user, err := h.store.CreateUser(r.Context(), req.Email, req.Name, hash)
	if err != nil {
		h.logger.Error("creating user", "error", err)
		writeError(w, http.StatusConflict, "email already exists")
		return
	}

	// Auto-create org for the user
	orgName := fmt.Sprintf("%s's Workspace", req.Name)
	slug := generateSlug(req.Email)
	org, err := h.store.CreateOrg(r.Context(), orgName, slug)
	if err != nil {
		h.logger.Error("creating default org", "error", err)
		// Don't fail registration if org creation fails
	} else {
		// Create membership with owner role
		_, err = h.store.CreateMembership(r.Context(), user.ID, org.ID, "owner")
		if err != nil {
			h.logger.Error("creating owner membership", "error", err)
		}
	}

	writeJSON(w, http.StatusCreated, user)
}

func generateSlug(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		slug := strings.ToLower(parts[0])
		slug = strings.ReplaceAll(slug, ".", "-")
		slug = strings.ReplaceAll(slug, "+", "-")
		slug = strings.ReplaceAll(slug, "_", "-")
		return slug
	}
	return "workspace"
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = validation.SanitizeString(req.Email)

	if err := validation.ValidateEmail(req.Email); err != nil {
		writeValidationError(w, err.Field, err.Message)
		return
	}

	if req.Password == "" {
		writeValidationError(w, "password", "password is required")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	valid, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !valid {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := auth.GenerateToken(user.ID, uuid.Nil, "", auth.TokenAccess, h.config.JWTSecret, h.config.AccessTTL)
	if err != nil {
		h.logger.Error("generating access token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	refreshToken, err := auth.GenerateToken(user.ID, uuid.Nil, "", auth.TokenRefresh, h.config.JWTSecret, h.config.RefreshTTL)
	if err != nil {
		h.logger.Error("generating refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		writeValidationError(w, "refresh_token", "refresh token is required")
		return
	}

	claims, err := auth.ValidateToken(req.RefreshToken, h.config.JWTSecret, auth.TokenRefresh)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	accessToken, err := auth.GenerateToken(claims.UserID, uuid.Nil, "", auth.TokenAccess, h.config.JWTSecret, h.config.AccessTTL)
	if err != nil {
		h.logger.Error("generating access token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	refreshToken, err := auth.GenerateToken(claims.UserID, uuid.Nil, "", auth.TokenRefresh, h.config.JWTSecret, h.config.RefreshTTL)
	if err != nil {
		h.logger.Error("generating refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.store.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
		"code":  status,
	})
}

func writeValidationError(w http.ResponseWriter, field, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "validation error",
		"code":    400,
		"field":   field,
		"message": message,
	})
}
