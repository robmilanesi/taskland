package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/repository"
)

func newAuthHandler(t *testing.T) (*AuthHandler, *auth.Issuer, *repository.Store) {
	t.Helper()
	store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	issuer := auth.NewIssuer("auth-handler-test-secret-of-enough-length", time.Hour)
	return NewAuthHandler(store, issuer), issuer, store
}

func doJSON(t *testing.T, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestAuthHandler_Register_Created(t *testing.T) {
	h, _, store := newAuthHandler(t)

	rec := doJSON(t, h.Register, `{"email":"a@example.com","password":"supersecret"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["email"] != "a@example.com" {
		t.Errorf("email = %v, want a@example.com", body["email"])
	}
	idStr, _ := body["id"].(string)
	if idStr == "" {
		t.Fatalf("expected an id, got %v", body["id"])
	}
	if _, leaked := body["password_hash"]; leaked {
		t.Error("response leaked password_hash")
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		t.Fatalf("response id %q is not a uuid: %v", idStr, err)
	}
	lists, err := store.Lists.GetAllLists(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetAllLists: %v", err)
	}
	if len(lists) != 1 || !lists[0].IsInbox {
		t.Fatalf("expected registration to create exactly one inbox list, got %+v", lists)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	h, _, _ := newAuthHandler(t)

	first := doJSON(t, h.Register, `{"email":"dup@example.com","password":"supersecret"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first register status = %d", first.Code)
	}

	second := doJSON(t, h.Register, `{"email":"dup@example.com","password":"anotherone"}`)
	if second.Code != http.StatusConflict {
		t.Errorf("second register status = %d, want 409", second.Code)
	}
}

func TestAuthHandler_Register_Validation(t *testing.T) {
	h, _, _ := newAuthHandler(t)

	tests := map[string]string{
		"empty email":    `{"email":"  ","password":"supersecret"}`,
		"short password": `{"email":"a@example.com","password":"short"}`,
		"malformed json": `{"email":`,
		"unknown field":  `{"email":"a@example.com","password":"supersecret","admin":true}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			rec := doJSON(t, h.Register, body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestAuthHandler_Login_OK(t *testing.T) {
	h, issuer, _ := newAuthHandler(t)

	reg := doJSON(t, h.Register, `{"email":"login@example.com","password":"supersecret"}`)
	if reg.Code != http.StatusCreated {
		t.Fatalf("register status = %d", reg.Code)
	}
	var regBody struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(reg.Body).Decode(&regBody)

	rec := doJSON(t, h.Login, `{"email":"login@example.com","password":"supersecret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Token == "" {
		t.Fatal("expected a token")
	}

	uid, err := issuer.Verify(body.Token)
	if err != nil {
		t.Fatalf("issued token does not verify: %v", err)
	}
	if uid.String() != regBody.ID {
		t.Errorf("token subject = %s, want %s", uid, regBody.ID)
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	h, _, _ := newAuthHandler(t)

	if reg := doJSON(t, h.Register, `{"email":"w@example.com","password":"supersecret"}`); reg.Code != http.StatusCreated {
		t.Fatalf("register status = %d", reg.Code)
	}

	rec := doJSON(t, h.Login, `{"email":"w@example.com","password":"wrongpassword"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthHandler_Login_UnknownEmail(t *testing.T) {
	h, _, _ := newAuthHandler(t)

	rec := doJSON(t, h.Login, `{"email":"ghost@example.com","password":"whatever!!"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
