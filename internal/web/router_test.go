package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouter_Home(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Taskland") {
		t.Errorf("expected the home page to mention Taskland, got: %s", rec.Body.String())
	}
}

func TestRouter_Static(t *testing.T) {
	router := NewRouter()

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
	router := NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
