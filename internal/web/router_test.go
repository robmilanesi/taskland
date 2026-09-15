package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/repository"
)

const webTestSecret = "web-test-secret-with-enough-length"

func newRouterAndStore(t *testing.T) (http.Handler, *repository.Store) {
	t.Helper()
	store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	issuer := auth.NewIssuer(webTestSecret, time.Hour)
	return NewRouter(store, issuer), store
}

// registerAndLogin drives the real register form to get a valid session
// cookie, exercising the same path a browser would.
func registerAndLogin(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()
	form := url.Values{"email": {email}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("register status = %d, want 303; body %s", rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatal("register did not set a session cookie")
	return nil
}

func TestRouter_Home_RequiresAuth(t *testing.T) {
	router, _ := newRouterAndStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestRouter_Home_HXRequestRedirectsViaHeader(t *testing.T) {
	router, _ := newRouterAndStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (htmx swaps on HX-Redirect, not on a real redirect status)", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "/login" {
		t.Errorf("HX-Redirect = %q, want /login", got)
	}
}

func TestRouter_RegisterThenHome(t *testing.T) {
	router, store := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "a@example.com")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "a@example.com") {
		t.Errorf("expected the home page to show the logged-in email, got: %s", rec.Body.String())
	}

	users, err := store.Users.GetUserByEmail(req.Context(), "a@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if users.Email != "a@example.com" {
		t.Errorf("unexpected user: %+v", users)
	}
}

func TestRouter_Register_DuplicateEmail(t *testing.T) {
	router, _ := newRouterAndStore(t)
	registerAndLogin(t, router, "dup@example.com")

	form := url.Values{"email": {"dup@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (re-rendered form with an error)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "already registered") {
		t.Errorf("expected an already-registered error, got: %s", rec.Body)
	}
}

func TestRouter_Login_InvalidCredentials(t *testing.T) {
	router, _ := newRouterAndStore(t)
	registerAndLogin(t, router, "b@example.com")

	form := url.Values{"email": {"b@example.com"}, "password": {"wrong-password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (re-rendered form with an error)", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid credentials") {
		t.Errorf("expected an invalid-credentials error, got: %s", rec.Body)
	}
}

func TestRouter_Login_Success(t *testing.T) {
	router, _ := newRouterAndStore(t)
	registerAndLogin(t, router, "c@example.com")

	form := url.Values{"email": {"c@example.com"}, "password": {"password123"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if rec.Header().Get("Location") != "/" {
		t.Errorf("Location = %q, want /", rec.Header().Get("Location"))
	}
}

func TestRouter_Logout(t *testing.T) {
	router, _ := newRouterAndStore(t)
	cookie := registerAndLogin(t, router, "d@example.com")

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}

	var cleared *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			cleared = c
		}
	}
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Fatalf("expected logout to clear the session cookie, got %+v", cleared)
	}
}

func TestRouter_Static(t *testing.T) {
	router, _ := newRouterAndStore(t)

	req := httptest.NewRequest(http.MethodGet, "/static/htmx.min.js", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected a non-empty response body")
	}
}

func TestRouter_UnknownRoute(t *testing.T) {
	router, _ := newRouterAndStore(t)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
