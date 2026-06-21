package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
)

type OrgHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewOrgHandler(store *store.Store, logger *slog.Logger) *OrgHandler {
	return &OrgHandler{store: store, logger: logger}
}

type CreateOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type InviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

func (h *OrgHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Slug == "" {
		writeError(w, http.StatusBadRequest, "name and slug are required")
		return
	}

	org, err := h.store.CreateOrg(r.Context(), req.Name, req.Slug)
	if err != nil {
		h.logger.Error("creating org", "error", err)
		writeError(w, http.StatusConflict, "slug already exists")
		return
	}

	membership, err := h.store.CreateMembership(r.Context(), claims.UserID, org.ID, "owner")
	if err != nil {
		h.logger.Error("creating owner membership", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = h.store.CreateAuditLog(r.Context(), org.ID, claims.UserID, "org.create", org.ID.String(), nil)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"org":        org,
		"membership": membership,
	})
}

func (h *OrgHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orgs, err := h.store.ListOrgsByUser(r.Context(), claims.UserID)
	if err != nil {
		h.logger.Error("listing orgs", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, orgs)
}

func (h *OrgHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	org, err := h.store.GetOrgByID(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusNotFound, "org not found")
		return
	}

	writeJSON(w, http.StatusOK, org)
}

func (h *OrgHandler) Invite(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	role := req.Role
	if role == "" {
		role = "member"
	}

	validRoles := map[string]bool{"owner": true, "admin": true, "member": true, "viewer": true}
	if !validRoles[role] {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	membership, err := h.store.CreateMembership(r.Context(), user.ID, orgID, role)
	if err != nil {
		h.logger.Error("creating membership", "error", err)
		writeError(w, http.StatusConflict, "user is already a member")
		return
	}

	_ = h.store.CreateAuditLog(r.Context(), orgID, claims.UserID, "org.invite", user.ID.String(), map[string]interface{}{"role": role})

	writeJSON(w, http.StatusCreated, membership)
}

func (h *OrgHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	membership, err := h.store.GetMembership(r.Context(), claims.UserID, orgID)
	if err != nil {
		writeError(w, http.StatusNotFound, "no pending invite")
		return
	}

	writeJSON(w, http.StatusOK, membership)
}

func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	members, err := h.store.ListMembers(r.Context(), orgID)
	if err != nil {
		h.logger.Error("listing members", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, members)
}

func (h *OrgHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validRoles := map[string]bool{"admin": true, "member": true, "viewer": true}
	if !validRoles[req.Role] {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := h.store.UpdateMemberRole(r.Context(), userID, orgID, req.Role); err != nil {
		h.logger.Error("updating role", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = h.store.CreateAuditLog(r.Context(), orgID, claims.UserID, "org.update_role", userID.String(), map[string]interface{}{"role": req.Role})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if userID == claims.UserID {
		writeError(w, http.StatusBadRequest, "cannot remove yourself")
		return
	}

	if err := h.store.RemoveMember(r.Context(), userID, orgID); err != nil {
		h.logger.Error("removing member", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = h.store.CreateAuditLog(r.Context(), orgID, claims.UserID, "org.remove_member", userID.String(), nil)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *OrgHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	logs, err := h.store.ListAuditLogs(r.Context(), orgID)
	if err != nil {
		h.logger.Error("listing audit logs", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, logs)
}
