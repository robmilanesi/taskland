package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/repository"
)

func newListHandler(t *testing.T) (*ListHandler, *repository.Store) {
	t.Helper()
	store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return NewListHandler(store.Lists), store
}

func TestListHandler_Create(t *testing.T) {
	h, _ := newListHandler(t)
	owner := uuid.New()

	req := withOwner(httptest.NewRequest(http.MethodPost, "/api/v1/lists",
		strings.NewReader(`{"name":"  Groceries  "}`)), owner)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "Groceries" {
		t.Errorf("name = %v, want trimmed %q", body["name"], "Groceries")
	}
	if body["is_inbox"] != false {
		t.Errorf("is_inbox = %v, want false", body["is_inbox"])
	}
	if rec.Header().Get("Location") == "" {
		t.Error("expected a Location header")
	}
}

func TestListHandler_Create_RequiresAuth(t *testing.T) {
	h, _ := newListHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lists", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestListHandler_Create_Validation(t *testing.T) {
	h, _ := newListHandler(t)

	req := withOwner(httptest.NewRequest(http.MethodPost, "/api/v1/lists",
		strings.NewReader(`{"name":"   "}`)), uuid.New())
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestListHandler_GetAllLists_OnlyMine(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()

	if _, err := store.Lists.CreateList(ctx, me, "a"); err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	if _, err := store.Lists.CreateList(ctx, me, "b"); err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	if _, err := store.Lists.CreateList(ctx, other, "not mine"); err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := withOwner(httptest.NewRequest(http.MethodGet, "/api/v1/lists", nil), me)
	rec := httptest.NewRecorder()

	h.GetAllLists(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var lists []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&lists); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(lists) != 2 {
		t.Errorf("got %d lists, want 2", len(lists))
	}
}

func TestListHandler_GetList_NotAMember(t *testing.T) {
	h, store := newListHandler(t)
	owner := uuid.New()
	list, err := store.Lists.CreateList(context.Background(), owner, "private")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lists/"+list.ID.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, uuid.New())
	rec := httptest.NewRecorder()

	h.GetList(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestListHandler_Update_Owner(t *testing.T) {
	h, store := newListHandler(t)
	owner := uuid.New()
	list, err := store.Lists.CreateList(context.Background(), owner, "old")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lists/"+list.ID.String(),
		strings.NewReader(`{"name":"new"}`))
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "new" {
		t.Errorf("name = %v, want new", body["name"])
	}
}

func TestListHandler_Update_NonOwnerMember_Forbidden(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	owner, member := uuid.New(), uuid.New()
	list, err := store.Lists.CreateList(ctx, owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	if err := store.Lists.AddMember(ctx, owner, list.ID.String(), member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/lists/"+list.ID.String(),
		strings.NewReader(`{"name":"renamed"}`))
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, member)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestListHandler_Delete_Owner(t *testing.T) {
	h, store := newListHandler(t)
	owner := uuid.New()
	list, err := store.Lists.CreateList(context.Background(), owner, "goner")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestListHandler_Delete_Inbox_Conflict(t *testing.T) {
	h, store := newListHandler(t)
	owner := uuid.New()
	inbox, err := store.Lists.InboxFor(context.Background(), owner)
	if err != nil {
		t.Fatalf("InboxFor: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+inbox.ID.String(), nil)
	req.SetPathValue("id", inbox.ID.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestListHandler_Delete_NonOwnerMember_Forbidden(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	owner, member := uuid.New(), uuid.New()
	list, err := store.Lists.CreateList(ctx, owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	if err := store.Lists.AddMember(ctx, owner, list.ID.String(), member); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, member)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}
