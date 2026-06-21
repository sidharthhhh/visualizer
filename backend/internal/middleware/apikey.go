package middleware

import (
	"context"
	"net/http"
)

type APIKeyContextKey string

const APIKeyOrgID APIKeyContextKey = "api_key_org_id"

func APIKeyAuth(validKeys map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			orgID, ok := validKeys[apiKey]
			if !ok {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), APIKeyOrgID, orgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAPIKeyOrgID(ctx context.Context) (string, bool) {
	orgID, ok := ctx.Value(APIKeyOrgID).(string)
	return orgID, ok
}

func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
