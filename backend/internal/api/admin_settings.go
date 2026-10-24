package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/service"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/version"
)

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settings.Get(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	patch, err := decodeRawJSON(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	settings, err := s.settings.Update(r.Context(), patch)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, settings)
}

func (s *Server) handleUploadIcon(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(int64(service.MaxIconBytes) + (64 << 10)); err != nil {
		writeError(w, r, NewAPIError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "上传内容过大或格式错误"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, ValidationError("缺少上传文件", FieldDetail("file", "必填")))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, int64(service.MaxIconBytes)+1))
	if err != nil {
		writeError(w, r, err)
		return
	}

	upload, err := s.icons.SaveUpload(r.Context(), data, header.Filename)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusCreated, map[string]any{
		"filename": upload.Filename,
		"url":      service.UploadURL(upload.Filename),
		"size":     upload.Size,
		"mime":     upload.Mime,
	})
}

type fetchIconRequest struct {
	URL string `json:"url"`
}

func (s *Server) handleFetchIcon(w http.ResponseWriter, r *http.Request) {
	var req fetchIconRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	iconType, iconValue, err := s.icons.Fetch(r.Context(), req.URL)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"icon_type":  iconType,
		"icon_value": iconValue,
	})
}

type updateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
	CheckedAt       string `json:"checked_at"`
}

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	info := updateInfo{
		CurrentVersion: version.Version,
		LatestVersion:  version.Version,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	if repo := strings.TrimSpace(s.cfg.UpdateRepo); repo != "" {
		if tag, url, err := s.fetchLatestRelease(r, repo); err == nil && tag != "" {
			info.LatestVersion = tag
			info.ReleaseURL = url
			info.UpdateAvailable = normalizeVersion(tag) != normalizeVersion(version.Version)
		} else if err != nil {
			s.log.Warn("update check failed", "error", err, "repo", repo)
		}
	}

	writeJSON(w, r, http.StatusOK, info)
}

func (s *Server) fetchLatestRelease(r *http.Request, repo string) (string, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "XiaoWork-Nav/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", "", err
	}
	return payload.TagName, payload.HTMLURL, nil
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}
