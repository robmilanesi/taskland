package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/auth"
)

func TestAuthenticate_ValidToken(t *testing.T) {
	issuer := auth.NewIssuer("auth-mw-test-secret-of-enough-length", time.Hour)
	want := uuid.New()
	token, err := issuer.Issue(want)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	var got uuid.UUID
	var ok bool
	h := Authenticate(issuer)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, ok = OwnerFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !ok || got != want {
		t.Errorf("OwnerFromContext = (%s, %v), want (%s, true)", got, ok, want)
	}
}

func TestAuthenticate_Rejects(t *testing.T) {
	issuer := auth.NewIssuer("auth-mw-test-secret-of-enough-length", time.Hour)
	expired := auth.NewIssuer("auth-mw-test-secret-of-enough-length", -time.Hour)
	expiredToken, _ := expired.Issue(uuid.New())
	otherToken, _ := auth.NewIssuer("a-completely-different-secret-value!!", time.Hour).Issue(uuid.New())

	tests := map[string]string{
		"no header":      "",
		"wrong scheme":   "Token abc",
		"bare Bearer":    "Bearer",
		"empty token":    "Bearer ",
		"garbage token":  "Bearer not.a.jwt",
		"expired token":  "Bearer " + expiredToken,
		"foreign secret": "Bearer " + otherToken,
	}
	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			called := false
			h := Authenticate(issuer)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			if called {
				t.Error("downstream handler should not run")
			}
		})
	}
}

func TestOwnerFromContext_Absent(t *testing.T) {
	if id, ok := OwnerFromContext(context.Background()); ok || id != uuid.Nil {
		t.Errorf("got (%s, %v), want (00000000-..., false)", id, ok)
	}
}
