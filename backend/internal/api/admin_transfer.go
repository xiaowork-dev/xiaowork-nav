package api

import (
	"encoding/json"
	"io"
	"net/http"
)

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	bundle, err := s.transfer.Export(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, bundle)
}

func (s *Server) handleImportPreview(w http.ResponseWriter, r *http.Request) {
	raw, err := readBundleBody(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	preview, err := s.transfer.Preview(r.Context(), raw)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, preview)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	raw, err := readBundleBody(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	strategy := "skip"
	var probe struct {
		ConflictStrategy string `json:"conflict_strategy"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && probe.ConflictStrategy != "" {
		strategy = probe.ConflictStrategy
	}

	result, err := s.transfer.Import(r.Context(), raw, strategy)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, r, http.StatusOK, result)
}

func readBundleBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, ValidationError("读取请求体失败")
	}
	if len(raw) == 0 {
		return nil, ValidationError("请求体不能为空")
	}
	if len(raw) > MaxBodyBytes {
		return nil, NewAPIError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "导入数据过大")
	}
	return raw, nil
}
