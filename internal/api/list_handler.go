package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/repository"
)

// ListHandler serves the HTTP endpoints for the list resource.
type ListHandler struct {
	repo  repository.ListRepository
	users repository.UserRepository
}

// NewListHandler returns a ListHandler backed by the given repositories. users
// is used to resolve an email to a user id when adding a member and to resolve
// member ids back to emails when listing them.
func NewListHandler(repo repository.ListRepository, users repository.UserRepository) *ListHandler {
	return &ListHandler{repo: repo, users: users}
}

// Create handles POST /api/v1/lists.
func (h *ListHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	var req listRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	list, err := h.repo.CreateList(r.Context(), ownerID, strings.TrimSpace(req.Name))
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	w.Header().Set("Location", "/api/v1/lists/"+list.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, list)
}

// GetAllLists handles GET /api/v1/lists.
func (h *ListHandler) GetAllLists(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	lists, err := h.repo.GetAllLists(r.Context(), userID)
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, lists)
}

// GetList handles GET /api/v1/lists/{id}.
func (h *ListHandler) GetList(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	list, err := h.repo.GetListByID(r.Context(), userID, r.PathValue("id"))
	if errors.Is(err, repository.ErrListNotFound) {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, list)
}

// Update handles PATCH /api/v1/lists/{id}: renaming, owner only.
func (h *ListHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	var req listRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	list, err := h.repo.UpdateList(r.Context(), userID, r.PathValue("id"), strings.TrimSpace(req.Name))
	if !h.writeListError(w, err) {
		return
	}

	httpx.WriteJSON(w, http.StatusOK, list)
}

// Delete handles DELETE /api/v1/lists/{id}: owner only, and never the inbox.
func (h *ListHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	err := h.repo.DeleteList(r.Context(), userID, r.PathValue("id"))
	if !h.writeListError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddMember handles POST /api/v1/lists/{id}/members: owner only.
func (h *ListHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	var req addMemberRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	target, err := h.users.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
	if errors.Is(err, repository.ErrUserNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "no user with that email")
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	listID := r.PathValue("id")
	if err := h.repo.AddMember(r.Context(), actorID, listID, target.ID); !h.writeListError(w, err) {
		return
	}

	w.Header().Set("Location", "/api/v1/lists/"+listID+"/members/"+target.ID.String())
	httpx.WriteJSON(w, http.StatusCreated, userResponse{ID: target.ID, Email: target.Email})
}

// RemoveMember handles DELETE /api/v1/lists/{id}/members/{userId}: the owner
// may remove anyone but themselves, any other member may only remove
// themselves ("leave").
func (h *ListHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	memberID, err := uuid.Parse(r.PathValue("userId"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "userId must be a uuid")
		return
	}

	if err := h.repo.RemoveMember(r.Context(), actorID, r.PathValue("id"), memberID); !h.writeListError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Members handles GET /api/v1/lists/{id}/members: any member may list them.
func (h *ListHandler) Members(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireOwner(w, r)
	if !ok {
		return
	}

	memberIDs, err := h.repo.ListMembers(r.Context(), userID, r.PathValue("id"))
	if !h.writeListError(w, err) {
		return
	}

	members := make([]userResponse, 0, len(memberIDs))
	for _, id := range memberIDs {
		user, err := h.users.GetUserByID(r.Context(), id)
		if err != nil {
			httpx.WriteISE(w)
			return
		}
		members = append(members, userResponse{ID: user.ID, Email: user.Email})
	}

	httpx.WriteJSON(w, http.StatusOK, members)
}

// writeListError maps the ListRepository sentinel errors to a response and
// reports whether the caller should continue (true) or has already responded
// with an error (false).
func (h *ListHandler) writeListError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, repository.ErrListNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrListForbidden):
		httpx.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, repository.ErrCannotDeleteInbox):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrAlreadyMember):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrNotAMember):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, repository.ErrCannotRemoveOwner):
		httpx.WriteError(w, http.StatusForbidden, err.Error())
	default:
		httpx.WriteISE(w)
	}
	return false
}
