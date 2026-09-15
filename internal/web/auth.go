package web

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/auth"
)

// sessionCookie is the name of the cookie carrying the session JWT. It is
// the same token format issued by /api/v1/auth/login, just delivered as a
// cookie instead of a JSON body, since a browser can't attach an
// Authorization header to a plain navigation.
const sessionCookie = "taskland_session"

type webCtxKey int

const userIDKey webCtxKey = iota

// setSession writes the session cookie for userID.
func setSession(w http.ResponseWriter, r *http.Request, issuer *auth.Issuer, userID uuid.UUID) error {
	token, err := issuer.Issue(userID)
	if err != nil {
		return err
	}
	//nolint:gosec // G124: Secure is derived from r.TLS, not omitted - false over
	// plain HTTP only so local dev (no TLS) keeps working; always true once the
	// server sits behind TLS (directly or via a proxy that terminates it).
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(issuer.TTL().Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// clearSession removes the session cookie, logging the user out.
func clearSession(w http.ResponseWriter, r *http.Request) {
	//nolint:gosec // G124: see setSession above.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireAuth redirects to /login when the request has no valid session
// cookie, and otherwise stores the authenticated user's id in the request
// context where UserFromContext can read it. A request made by htmx (marked
// with the HX-Request header) gets an HX-Redirect response instead of a
// plain 302, since htmx does not follow a normal redirect for its own
// fetches the way a full page navigation would.
func RequireAuth(issuer *auth.Issuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			userID, err := issuer.Verify(cookie.Value)
			if err != nil {
				redirectToLogin(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// UserFromContext returns the authenticated user id stored by RequireAuth.
// The bool is false when the request never passed through RequireAuth.
func UserFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}
