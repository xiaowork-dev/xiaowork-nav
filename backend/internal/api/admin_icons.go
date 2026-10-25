package api

import (
	"net/http"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/service"
)

// handleAutoBookmarkIcon fetches an icon for one bookmark and stores it.
//
// A failed fetch still returns 200 with ok=false and a reason, because the
// bookmark itself is unchanged and the admin needs the explanation.
func (s *Server) handleAutoBookmarkIcon(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req struct {
		Force bool `json:"force"`
	}
	if err := decodeOptionalJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	// ?force=true is accepted too, so the action also works without a body.
	if !req.Force {
		if force := queryBool(r, "force"); force != nil {
			req.Force = *force
		}
	}

	outcome, err := s.iconFill.FillBookmark(r.Context(), id, req.Force)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, outcome)
}

// handleIconSummary reports icon coverage across all bookmarks.
func (s *Server) handleIconSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.iconFill.Summary(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, summary)
}

type bulkIconRequest struct {
	OnlyMissing bool `json:"only_missing"`
	Force       bool `json:"force"`
	Limit       int  `json:"limit"`
	RefreshDays int  `json:"refresh_days"`
}

// handleStartBulkIcons starts the bulk icon completion job.
func (s *Server) handleStartBulkIcons(w http.ResponseWriter, r *http.Request) {
	var req bulkIconRequest
	if err := decodeOptionalJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	// A bare request means "fill in what is missing"; an explicit body wins.
	if r.ContentLength == 0 {
		req.OnlyMissing = true
	}

	snapshot, err := s.iconFill.StartBulk(service.BulkOptions{
		OnlyMissing: req.OnlyMissing,
		Force:       req.Force,
		Limit:       req.Limit,
		RefreshDays: req.RefreshDays,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusAccepted, map[string]any{"job": snapshot})
}

// handleBulkIconStatus returns the current or most recent job snapshot.
//
// The envelope is always an object (`{"job": null}` before the first run) so
// clients can parse the response unconditionally.
func (s *Server) handleBulkIconStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]any{"job": s.iconFill.BulkStatus()})
}

// handleCancelBulkIcons cancels the running bulk job.
func (s *Server) handleCancelBulkIcons(w http.ResponseWriter, r *http.Request) {
	canceled := s.iconFill.CancelBulk()
	writeJSON(w, r, http.StatusOK, map[string]any{
		"canceled": canceled,
		"job":      s.iconFill.BulkStatus(),
	})
}
