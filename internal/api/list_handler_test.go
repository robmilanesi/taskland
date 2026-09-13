package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

func registerTestUser(t *testing.T, store *repository.Store, email string) models.User {
	t.Helper()
	user, err := store.Users.CreateUser(context.Background(), models.User{Email: email, PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return user
}

func newListHandler(t *testing.T) (*ListHandler, *repository.Store) {
	t.Helper()
	store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return NewListHandler(store.Lists, store.Users), store
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

func TestListHandler_AddMember_Owner(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	owner := uuid.New()
	list, err := store.Lists.CreateList(ctx, owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	friend := registerTestUser(t, store, "friend@example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lists/"+list.ID.String()+"/members",
		strings.NewReader(`{"email":"friend@example.com"}`))
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["email"] != "friend@example.com" {
		t.Errorf("email = %v, want friend@example.com", body["email"])
	}
	if rec.Header().Get("Location") == "" {
		t.Error("expected a Location header")
	}
	if member, err := store.Lists.IsMember(ctx, friend.ID, list.ID); err != nil || !member {
		t.Errorf("friend should now be a member: %v, %v", member, err)
	}
}

func TestListHandler_AddMember_UnknownEmail(t *testing.T) {
	h, store := newListHandler(t)
	owner := uuid.New()
	list, err := store.Lists.CreateList(context.Background(), owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lists/"+list.ID.String()+"/members",
		strings.NewReader(`{"email":"ghost@example.com"}`))
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestListHandler_AddMember_AlreadyMember(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	owner := uuid.New()
	list, err := store.Lists.CreateList(ctx, owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	friend := registerTestUser(t, store, "friend@example.com")
	if err := store.Lists.AddMember(ctx, owner, list.ID.String(), friend.ID); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lists/"+list.ID.String()+"/members",
		strings.NewReader(`{"email":"friend@example.com"}`))
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestListHandler_AddMember_NonOwnerForbidden(t *testing.T) {
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
	registerTestUser(t, store, "friend@example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/lists/"+list.ID.String()+"/members",
		strings.NewReader(`{"email":"friend@example.com"}`))
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, member)
	rec := httptest.NewRecorder()

	h.AddMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestListHandler_RemoveMember_OwnerRemovesMember(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String()+"/members/"+member.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req.SetPathValue("userId", member.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body %s", rec.Code, rec.Body)
	}
	if isMember, err := store.Lists.IsMember(ctx, member, list.ID); err != nil || isMember {
		t.Errorf("member should have been removed: %v, %v", isMember, err)
	}
}

func TestListHandler_RemoveMember_SelfLeave(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String()+"/members/"+member.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req.SetPathValue("userId", member.String())
	req = withOwner(req, member)
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("self-leave should be allowed, status = %d; body %s", rec.Code, rec.Body)
	}
}

func TestListHandler_RemoveMember_CannotRemoveOwner(t *testing.T) {
	h, store := newListHandler(t)
	owner := uuid.New()
	list, err := store.Lists.CreateList(context.Background(), owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String()+"/members/"+owner.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req.SetPathValue("userId", owner.String())
	req = withOwner(req, owner)
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestListHandler_RemoveMember_NonOwnerCannotRemoveOthers(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	owner, a, b := uuid.New(), uuid.New(), uuid.New()
	list, err := store.Lists.CreateList(ctx, owner, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	for _, m := range []uuid.UUID{a, b} {
		if err := store.Lists.AddMember(ctx, owner, list.ID.String(), m); err != nil {
			t.Fatalf("AddMember: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String()+"/members/"+b.String(), nil)
	req.SetPathValue("id", list.ID.String())
	req.SetPathValue("userId", b.String())
	req = withOwner(req, a)
	rec := httptest.NewRecorder()

	h.RemoveMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestListHandler_Members_ListsEveryone(t *testing.T) {
	h, store := newListHandler(t)
	ctx := context.Background()
	owner := registerTestUser(t, store, "owner@example.com")
	list, err := store.Lists.CreateList(ctx, owner.ID, "shared")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	friend := registerTestUser(t, store, "friend@example.com")
	if err := store.Lists.AddMember(ctx, owner.ID, list.ID.String(), friend.ID); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lists/"+list.ID.String()+"/members", nil)
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, friend.ID)
	rec := httptest.NewRecorder()

	h.Members(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	var members []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&members); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("got %d members, want 2 (owner + friend)", len(members))
	}
}

func TestListHandler_Members_NotAMember(t *testing.T) {
	h, store := newListHandler(t)
	list, err := store.Lists.CreateList(context.Background(), uuid.New(), "private")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lists/"+list.ID.String()+"/members", nil)
	req.SetPathValue("id", list.ID.String())
	req = withOwner(req, uuid.New())
	rec := httptest.NewRecorder()

	h.Members(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
