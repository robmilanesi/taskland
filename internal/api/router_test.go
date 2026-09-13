package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

const routerTestSecret = "router-test-secret-with-enough-length"

const (
	authValid   = ""
	authNone    = "none"
	authForeign = "foreign"
)

// setupTaskIn creates a task for owner in their inbox, creating the inbox if
// this is the first thing owner has ever needed.
func setupTaskIn(t *testing.T, store *repository.Store, owner uuid.UUID, title string) uuid.UUID {
	t.Helper()
	inbox, err := store.Lists.InboxFor(context.Background(), owner)
	if err != nil {
		t.Fatalf("setup InboxFor: %v", err)
	}
	task, err := store.Tasks.Create(context.Background(), owner, models.Task{Title: title, ListID: inbox.ID})
	if err != nil {
		t.Fatalf("setup Create: %v", err)
	}
	return task.ID
}

func TestRouter_Routes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       func(id uuid.UUID) string
		body       string
		auth       string
		setup      func(t *testing.T, store *repository.Store, owner uuid.UUID) uuid.UUID
		wantStatus int
	}{
		{
			name:   "GET task by id - found",
			method: http.MethodGet,
			setup: func(t *testing.T, store *repository.Store, owner uuid.UUID) uuid.UUID {
				return setupTaskIn(t, store, owner, "test")
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET task by id - not found",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks/" + uuid.NewString() },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET all tasks",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST create task",
			method:     http.MethodPost,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			body:       `{"title":"new task"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:   "PATCH update task",
			method: http.MethodPatch,
			setup: func(t *testing.T, store *repository.Store, owner uuid.UUID) uuid.UUID {
				return setupTaskIn(t, store, owner, "to update")
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			body:       `{"title":"updated"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:   "DELETE task",
			method: http.MethodDelete,
			setup: func(t *testing.T, store *repository.Store, owner uuid.UUID) uuid.UUID {
				return setupTaskIn(t, store, owner, "to delete")
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "tasks without a token",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			auth:       authNone,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "tasks with a foreign token",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			auth:       authForeign,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "cannot GET another owner's task",
			method: http.MethodGet,
			setup: func(t *testing.T, store *repository.Store, _ uuid.UUID) uuid.UUID {
				return setupTaskIn(t, store, uuid.New(), "not yours")
			},
			path:       func(id uuid.UUID) string { return "/api/v1/tasks/" + id.String() },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "method not allowed on collection",
			method:     http.MethodDelete,
			path:       func(uuid.UUID) string { return "/api/v1/tasks" },
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       func(uuid.UUID) string { return "/api/v1/unknown" },
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
			if err != nil {
				t.Fatalf("NewStore: %v", err)
			}

			owner := uuid.New()
			issuer := auth.NewIssuer(routerTestSecret, time.Hour)
			token, err := issuer.Issue(owner)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			router := NewRouter(store, issuer)

			var id uuid.UUID
			if tt.setup != nil {
				id = tt.setup(t, store, owner)
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path(id), strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path(id), nil)
			}

			switch tt.auth {
			case authNone:
				// send no Authorization header
			case authForeign:
				foreign, _ := auth.NewIssuer("a-totally-different-secret-value!!", time.Hour).Issue(uuid.New())
				req.Header.Set("Authorization", "Bearer "+foreign)
			default:
				req.Header.Set("Authorization", "Bearer "+token)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRouter_ListRoutes(t *testing.T) {
	newRouterAndStore := func(t *testing.T) (http.Handler, *repository.Store, *auth.Issuer) {
		t.Helper()
		store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
		if err != nil {
			t.Fatalf("NewStore: %v", err)
		}
		issuer := auth.NewIssuer(routerTestSecret, time.Hour)
		return NewRouter(store, issuer), store, issuer
	}

	tokenFor := func(t *testing.T, issuer *auth.Issuer, id uuid.UUID) string {
		t.Helper()
		token, err := issuer.Issue(id)
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		return token
	}

	t.Run("create a list", func(t *testing.T) {
		router, _, issuer := newRouterAndStore(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/lists", strings.NewReader(`{"name":"Groceries"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, uuid.New()))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want 201; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("lists without a token", func(t *testing.T) {
		router, _, _ := newRouterAndStore(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/lists", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("cannot GET a list you are not a member of", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		list, err := store.Lists.CreateList(context.Background(), uuid.New(), "private")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/lists/"+list.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, uuid.New()))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("a non-owner member cannot rename the list", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		owner, member := uuid.New(), uuid.New()
		list, err := store.Lists.CreateList(context.Background(), owner, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := store.Lists.AddMember(context.Background(), owner, list.ID.String(), member); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/lists/"+list.ID.String(), strings.NewReader(`{"name":"renamed"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, member))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("the inbox cannot be deleted", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		owner := uuid.New()
		inbox, err := store.Lists.InboxFor(context.Background(), owner)
		if err != nil {
			t.Fatalf("InboxFor: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+inbox.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, owner))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})

	t.Run("owner deletes a regular list", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		owner := uuid.New()
		list, err := store.Lists.CreateList(context.Background(), owner, "goner")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, owner))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("owner adds a member by email", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		owner := registerTestUser(t, store, "owner@example.com")
		friend := registerTestUser(t, store, "friend@example.com")
		list, err := store.Lists.CreateList(context.Background(), owner.ID, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/lists/"+list.ID.String()+"/members",
			strings.NewReader(`{"email":"friend@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, owner.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
		}
		if isMember, err := store.Lists.IsMember(context.Background(), friend.ID, list.ID); err != nil || !isMember {
			t.Errorf("friend should be a member: %v, %v", isMember, err)
		}
	})

	t.Run("a non-owner member cannot add members", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		owner := registerTestUser(t, store, "owner@example.com")
		member := registerTestUser(t, store, "member@example.com")
		registerTestUser(t, store, "outsider@example.com")
		list, err := store.Lists.CreateList(context.Background(), owner.ID, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := store.Lists.AddMember(context.Background(), owner.ID, list.ID.String(), member.ID); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/v1/lists/"+list.ID.String()+"/members",
			strings.NewReader(`{"email":"outsider@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, member.ID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("a member can leave, owner cannot be removed", func(t *testing.T) {
		router, store, issuer := newRouterAndStore(t)
		owner := registerTestUser(t, store, "owner@example.com")
		member := registerTestUser(t, store, "member@example.com")
		list, err := store.Lists.CreateList(context.Background(), owner.ID, "shared")
		if err != nil {
			t.Fatalf("CreateList: %v", err)
		}
		if err := store.Lists.AddMember(context.Background(), owner.ID, list.ID.String(), member.ID); err != nil {
			t.Fatalf("AddMember: %v", err)
		}

		leaveReq := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String()+"/members/"+member.ID.String(), nil)
		leaveReq.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, member.ID))
		leaveRec := httptest.NewRecorder()
		router.ServeHTTP(leaveRec, leaveReq)
		if leaveRec.Code != http.StatusNoContent {
			t.Fatalf("self-leave status = %d, want 204; body %s", leaveRec.Code, leaveRec.Body)
		}

		removeOwnerReq := httptest.NewRequest(http.MethodDelete, "/api/v1/lists/"+list.ID.String()+"/members/"+owner.ID.String(), nil)
		removeOwnerReq.Header.Set("Authorization", "Bearer "+tokenFor(t, issuer, owner.ID))
		removeOwnerRec := httptest.NewRecorder()
		router.ServeHTTP(removeOwnerRec, removeOwnerReq)
		if removeOwnerRec.Code != http.StatusForbidden {
			t.Errorf("removing the owner status = %d, want 403", removeOwnerRec.Code)
		}
	})
}

// TestRouter_ListSharingE2E walks through the full sharing scenario from
// next.md: A creates a list and a task, shares it with B, B sees and adds to
// it, then A revokes B's access and B loses visibility again.
func TestRouter_ListSharingE2E(t *testing.T) {
	store, err := repository.NewStore(repository.Config{Type: repository.TaskRepoInMemory})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	issuer := auth.NewIssuer(routerTestSecret, time.Hour)
	router := NewRouter(store, issuer)
	ctx := context.Background()

	a := registerTestUser(t, store, "a@example.com")
	b := registerTestUser(t, store, "b@example.com")
	tokenA, err := issuer.Issue(a.ID)
	if err != nil {
		t.Fatalf("Issue A: %v", err)
	}
	tokenB, err := issuer.Issue(b.ID)
	if err != nil {
		t.Fatalf("Issue B: %v", err)
	}

	do := func(token, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		var req *http.Request
		if body != "" {
			req = httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(method, path, nil)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// A creates the "Spesa" list.
	createList := do(tokenA, http.MethodPost, "/api/v1/lists", `{"name":"Spesa"}`)
	if createList.Code != http.StatusCreated {
		t.Fatalf("create list: status = %d; body %s", createList.Code, createList.Body)
	}
	list, err := store.Lists.GetAllLists(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetAllLists: %v", err)
	}
	var spesaID uuid.UUID
	for _, l := range list {
		if l.Name == "Spesa" {
			spesaID = l.ID
		}
	}
	if spesaID == uuid.Nil {
		t.Fatalf("Spesa list not found among A's lists: %+v", list)
	}

	// A creates a task in that list.
	createTask := do(tokenA, http.MethodPost, "/api/v1/tasks", `{"title":"milk","list_id":"`+spesaID.String()+`"}`)
	if createTask.Code != http.StatusCreated {
		t.Fatalf("create task: status = %d; body %s", createTask.Code, createTask.Body)
	}

	// A adds B as a member.
	addMember := do(tokenA, http.MethodPost, "/api/v1/lists/"+spesaID.String()+"/members", `{"email":"b@example.com"}`)
	if addMember.Code != http.StatusCreated {
		t.Fatalf("add member: status = %d; body %s", addMember.Code, addMember.Body)
	}

	// B now sees the list in GET /lists and the task in GET /tasks.
	bLists := do(tokenB, http.MethodGet, "/api/v1/lists", "")
	if bLists.Code != http.StatusOK || !strings.Contains(bLists.Body.String(), "Spesa") {
		t.Fatalf("B's lists should include Spesa: status %d, body %s", bLists.Code, bLists.Body)
	}
	bTasks := do(tokenB, http.MethodGet, "/api/v1/tasks", "")
	if bTasks.Code != http.StatusOK || !strings.Contains(bTasks.Body.String(), "milk") {
		t.Fatalf("B's tasks should include milk: status %d, body %s", bTasks.Code, bTasks.Body)
	}

	// B adds a task to the shared list; A sees it.
	bAddsTask := do(tokenB, http.MethodPost, "/api/v1/tasks", `{"title":"eggs","list_id":"`+spesaID.String()+`"}`)
	if bAddsTask.Code != http.StatusCreated {
		t.Fatalf("B create task: status = %d; body %s", bAddsTask.Code, bAddsTask.Body)
	}
	aTasks := do(tokenA, http.MethodGet, "/api/v1/tasks", "")
	if aTasks.Code != http.StatusOK || !strings.Contains(aTasks.Body.String(), "eggs") {
		t.Fatalf("A's tasks should include eggs: status %d, body %s", aTasks.Code, aTasks.Body)
	}

	// A removes B from the list.
	removeMember := do(tokenA, http.MethodDelete, "/api/v1/lists/"+spesaID.String()+"/members/"+b.ID.String(), "")
	if removeMember.Code != http.StatusNoContent {
		t.Fatalf("remove member: status = %d; body %s", removeMember.Code, removeMember.Body)
	}

	// B no longer sees the list or its tasks.
	bListsAfter := do(tokenB, http.MethodGet, "/api/v1/lists", "")
	if bListsAfter.Code != http.StatusOK || strings.Contains(bListsAfter.Body.String(), "Spesa") {
		t.Fatalf("B should no longer see Spesa: status %d, body %s", bListsAfter.Code, bListsAfter.Body)
	}
	bGetList := do(tokenB, http.MethodGet, "/api/v1/lists/"+spesaID.String(), "")
	if bGetList.Code != http.StatusNotFound {
		t.Errorf("B GET the list directly: status = %d, want 404", bGetList.Code)
	}
	bTasksAfter := do(tokenB, http.MethodGet, "/api/v1/tasks", "")
	if bTasksAfter.Code != http.StatusOK || strings.Contains(bTasksAfter.Body.String(), "milk") || strings.Contains(bTasksAfter.Body.String(), "eggs") {
		t.Fatalf("B should no longer see any Spesa task: status %d, body %s", bTasksAfter.Code, bTasksAfter.Body)
	}
}
