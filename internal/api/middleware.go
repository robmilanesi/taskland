package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/httpx"
)

type ctxKey int

const ownerIDKey ctxKey = iota

// Authenticate rejects requests that do not carry a valid Bearer token and, on
// success, stores the authenticated user's id in the request context where
// OwnerFromContext can read it.
func Authenticate(issuer *auth.Issuer) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			userID, err := issuer.Verify(token)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), ownerIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return header[len(prefix):], true
}

// OwnerFromContext returns the authenticated user id stored by Authenticate. The
// bool is false when the request did not pass through Authenticate.
func OwnerFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ownerIDKey).(uuid.UUID)
	return id, ok
}

// requireOwner returns the authenticated user id for any protected handler. When
// the request never passed through Authenticate it writes a 401 and returns
// false, so the handler can just return.
func requireOwner(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := OwnerFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication required")
	}
	return id, ok
}
