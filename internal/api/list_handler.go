package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/repository"
)

// ListHandler serves the HTTP endpoints for the list resource.
type ListHandler struct {
	repo repository.ListRepository
}

// NewListHandler returns a ListHandler backed by the given repository.
func NewListHandler(repo repository.ListRepository) *ListHandler {
	return &ListHandler{repo: repo}
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
	default:
		httpx.WriteISE(w)
	}
	return false
}
