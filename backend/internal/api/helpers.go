package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// decodeJSON reads a JSON request body into dst, rejecting unknown behaviour
// with a typed 400 error.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return ValidationError("请求体不能为空")
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return NewAPIError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "请求体过大")
		}
		if errors.Is(err, io.EOF) {
			return ValidationError("请求体不能为空")
		}
		return ValidationError("请求体不是合法 JSON")
	}
	return nil
}

// decodeRawJSON reads a JSON object into a field -> raw value map, used for
// partial (PATCH) updates.
func decodeRawJSON(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, error) {
	var patch map[string]json.RawMessage
	if err := decodeJSON(w, r, &patch); err != nil {
		return nil, err
	}
	if patch == nil {
		patch = map[string]json.RawMessage{}
	}
	return patch, nil
}

// pathID reads the {id} path parameter as an int64.
func pathID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id < 1 {
		return 0, ValidationError("资源 ID 非法", FieldDetail("id", "必须为正整数"))
	}
	return id, nil
}

// pagination reads page/page_size query parameters with defaults and limits.
func pagination(r *http.Request) (page, pageSize int) {
	page = queryInt(r, "page", 1)
	pageSize = queryInt(r, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func queryBool(r *http.Request, key string) *bool {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &v
}

func queryInt64(r *http.Request, key string) *int64 {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

// listResponse is the collection envelope used by list endpoints.
type listResponse[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func newList[T any](items []T, total, page, pageSize int) listResponse[T] {
	if items == nil {
		items = []T{}
	}
	return listResponse[T]{Items: items, Total: total, Page: page, PageSize: pageSize}
}
