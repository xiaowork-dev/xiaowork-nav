package api

import (
	"net/http"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/service"
)

func (s *Server) handleListBookmarks(w http.ResponseWriter, r *http.Request) {
	page, pageSize := pagination(r)
	categoryID := queryInt64(r, "category_id")
	query := r.URL.Query().Get("q")
	isVisible := queryBool(r, "is_visible")

	items, total, err := s.catalog.ListBookmarks(r.Context(), categoryID, query, isVisible, page, pageSize)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, newList(items, total, page, pageSize))
}

func (s *Server) handleCreateBookmark(w http.ResponseWriter, r *http.Request) {
	var in service.BookmarkInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, r, err)
		return
	}
	bookmark, err := s.catalog.CreateBookmark(r.Context(), in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, bookmark)
}

func (s *Server) handleGetBookmark(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	bookmark, err := s.catalog.GetBookmark(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, bookmark)
}

func (s *Server) handleUpdateBookmark(w http.ResponseWriter, r *http.Request) {
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
	bookmark, err := s.catalog.UpdateBookmark(r.Context(), id, patch)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, bookmark)
}

func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.catalog.DeleteBookmark(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleReorderBookmarks(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	updated, err := s.catalog.ReorderBookmarks(r.Context(), req.Items)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"ok": true, "updated": updated})
}

type moveRequest struct {
	CategoryID int64 `json:"category_id"`
	SortOrder  *int  `json:"sort_order"`
}

func (s *Server) handleMoveBookmark(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req moveRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if req.CategoryID < 1 {
		writeError(w, r, ValidationError("分类 ID 非法", FieldDetail("category_id", "必须为正整数")))
		return
	}
	bookmark, err := s.catalog.MoveBookmark(r.Context(), id, req.CategoryID, req.SortOrder)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, bookmark)
}
