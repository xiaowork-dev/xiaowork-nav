package api

import (
	"net/http"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/service"
)

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.catalog.ListCategories(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, newList(categories, len(categories), 1, len(categories)))
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var in service.CategoryInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, err)
		return
	}
	category, err := s.catalog.CreateCategory(r.Context(), in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, category)
}

func (s *Server) handleGetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	category, err := s.catalog.GetCategory(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, category)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	patch, err := decodeRawJSON(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	category, err := s.catalog.UpdateCategory(r.Context(), id, patch)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, category)
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.catalog.DeleteCategory(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type orderRequest struct {
	Items []models.OrderItem `json:"items"`
}

func (s *Server) handleReorderCategories(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	updated, err := s.catalog.ReorderCategories(r.Context(), req.Items)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"ok": true, "updated": updated})
}
