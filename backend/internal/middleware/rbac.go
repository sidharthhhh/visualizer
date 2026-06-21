package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/store"
)

type rbacContextKey string

const (
	OrgIDKey rbacContextKey = "org_id"
	RoleKey  rbacContextKey = "role"
)

var roleHierarchy = map[string]int{
	"viewer": 0,
	"member": 1,
	"admin":  2,
	"owner":  3,
}

func RequireRole(minRole string, store *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaims(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			orgIDStr := chi.URLParam(r, "orgID")
			if orgIDStr == "" {
				http.Error(w, `{"error":"missing org id"}`, http.StatusBadRequest)
				return
			}

			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				http.Error(w, `{"error":"invalid org id"}`, http.StatusBadRequest)
				return
			}

			membership, err := store.GetMembership(r.Context(), claims.UserID, orgID)
			if err != nil {
				http.Error(w, `{"error":"not a member of this org"}`, http.StatusForbidden)
				return
			}

			if roleHierarchy[membership.Role] < roleHierarchy[minRole] {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, OrgIDKey, orgID)
			ctx = context.WithValue(ctx, RoleKey, membership.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetOrgID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(OrgIDKey).(uuid.UUID)
	return id, ok
}

func GetRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(RoleKey).(string)
	return role, ok
}
