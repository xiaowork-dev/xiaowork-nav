package api

import (
	"net/http"
	"strings"
)

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := s.backup.List()
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"items": backups})
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	backup, err := s.backup.Create(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, backup)
}

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, r, ValidationError("备份 ID 不能为空", FieldDetail("id", "必填")))
		return
	}
	if err := s.backup.Restore(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}
