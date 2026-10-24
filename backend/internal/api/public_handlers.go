package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handlePublicNavigation(w http.ResponseWriter, r *http.Request) {
	nav, err := s.catalog.Navigation(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, nav)
}

func (s *Server) handlePublicCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.catalog.PublicCategories(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, newList(categories, len(categories), 1, len(categories)))
}

func (s *Server) handlePublicCategoryBookmarks(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items, err := s.catalog.PublicBookmarksByCategory(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, newList(items, len(items), 1, len(items)))
}

func (s *Server) handlePublicSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page, pageSize := pagination(r)
	items, total, err := s.catalog.Search(r.Context(), query, page, pageSize)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, newList(items, total, page, pageSize))
}

func (s *Server) handlePublicSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settings.Public(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, settings)
}

func (s *Server) handlePublicUpload(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimSpace(r.PathValue("filename"))
	file, mime, err := s.icons.Open(r.Context(), filename)
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer file.Close()

	if mime == "" {
		mime = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// SVG can carry script; serve it as a download-safe type.
	if strings.HasPrefix(mime, "image/svg") {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}
	if seeker, ok := file.(io.Seeker); ok {
		if size, err := seeker.Seek(0, io.SeekEnd); err == nil {
			_, _ = seeker.Seek(0, io.SeekStart)
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}
