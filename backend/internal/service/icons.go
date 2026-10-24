package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/security"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// Icon upload limits and allowed types.
const (
	MaxIconBytes     = 1 << 20 // 1 MiB
	maxHTMLLength    = 512 << 10
	maxRedirectHops  = 3
	iconFetchTimeout = 15 * time.Second
	// UploadURLPrefix is the public path that serves uploaded icons.
	UploadURLPrefix = "/api/v1/public/uploads/"
)

var (
	safeFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	iconLinkPattern     = regexp.MustCompile(`(?is)<link[^>]+rel=["']?[^"'>]*icon[^"'>]*["']?[^>]*>`)
	hrefPattern         = regexp.MustCompile(`(?is)href=["']([^"']+)["']`)
)

var allowedImageTypes = map[string]string{
	"image/png":                ".png",
	"image/jpeg":               ".jpg",
	"image/webp":               ".webp",
	"image/gif":                ".gif",
	"image/x-icon":             ".ico",
	"image/vnd.microsoft.icon": ".ico",
	"image/svg+xml":            ".svg",
}

// IconService stores uploaded icons and safely fetches remote favicons.
type IconService struct {
	store      *store.Store
	uploadsDir string
	client     *http.Client
}

// NewIconService builds an IconService.
func NewIconService(st *store.Store, uploadsDir string) *IconService {
	return &IconService{
		store:      st,
		uploadsDir: uploadsDir,
		client:     security.SafeHTTPClient(iconFetchTimeout, maxRedirectHops),
	}
}

// SaveUpload validates and stores an uploaded icon file.
func (i *IconService) SaveUpload(ctx context.Context, data []byte, originalName string) (*models.Upload, error) {
	if len(data) == 0 {
		return nil, Validation("validation_failed", "上传文件为空", FieldError{Field: "file", Issue: "文件为空"})
	}
	if len(data) > MaxIconBytes {
		return nil, New(KindTooLarge, "payload_too_large", "图标文件过大（上限 1 MiB）")
	}

	mime, ext, ok := detectImageType(data, originalName)
	if !ok {
		return nil, New(KindUnsupported, "unsupported_media_type", "不支持的图标格式",
			FieldError{Field: "file", Issue: "仅支持 PNG/JPEG/WebP/GIF/ICO/SVG"})
	}

	name, err := randomFilename(ext)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(i.uploadsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create uploads dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(i.uploadsDir, name), data, 0o640); err != nil {
		return nil, fmt.Errorf("write icon: %w", err)
	}

	upload := &models.Upload{
		Filename:     name,
		OriginalName: truncate(filepath.Base(originalName), 255),
		Mime:         mime,
		Size:         int64(len(data)),
	}
	if err := i.store.CreateUpload(ctx, upload); err != nil {
		_ = os.Remove(filepath.Join(i.uploadsDir, name))
		return nil, err
	}
	return upload, nil
}

// UploadURL returns the public URL for a stored icon.
func UploadURL(filename string) string {
	return UploadURLPrefix + filename
}

// Fetch downloads a site's favicon and stores it locally, returning the icon
// reference to persist on a bookmark.
func (i *IconService) Fetch(ctx context.Context, rawURL string) (string, string, error) {
	if err := security.ValidateHTTPURL(rawURL); err != nil {
		return "", "", New(KindUnsupported, "invalid_url_scheme", "URL 非法：仅允许 http/https",
			FieldError{Field: "url", Issue: err.Error()})
	}
	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", New(KindUnsupported, "invalid_url_scheme", "URL 非法")
	}

	candidates := []string{}
	if iconURL := i.discoverIcon(ctx, pageURL); iconURL != "" {
		candidates = append(candidates, iconURL)
	}
	candidates = append(candidates, pageURL.Scheme+"://"+pageURL.Host+"/favicon.ico")

	var lastErr error
	for _, candidate := range candidates {
		data, _, err := i.download(ctx, candidate, MaxIconBytes)
		if err != nil {
			lastErr = err
			continue
		}
		if _, _, ok := detectImageType(data, candidate); !ok {
			lastErr = fmt.Errorf("not an image: %s", candidate)
			continue
		}
		upload, err := i.SaveUpload(ctx, data, path.Base(candidate))
		if err != nil {
			lastErr = err
			continue
		}
		return "upload", UploadURL(upload.Filename), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no favicon found")
	}
	return "", "", New(KindNotFound, "not_found", "未能获取站点图标："+lastErr.Error())
}

func (i *IconService) discoverIcon(ctx context.Context, pageURL *url.URL) string {
	body, _, err := i.download(ctx, pageURL.String(), maxHTMLLength)
	if err != nil {
		return ""
	}
	matches := iconLinkPattern.FindAllString(string(body), -1)
	for _, tag := range matches {
		href := hrefPattern.FindStringSubmatch(tag)
		if len(href) < 2 {
			continue
		}
		ref, err := url.Parse(strings.TrimSpace(href[1]))
		if err != nil {
			continue
		}
		resolved := pageURL.ResolveReference(ref)
		if err := security.ValidateHTTPURL(resolved.String()); err == nil {
			return resolved.String()
		}
	}
	return ""
}

func (i *IconService) download(ctx context.Context, rawURL string, maxBytes int64) ([]byte, string, error) {
	resp, err := security.Fetch(ctx, i.client, rawURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", fmt.Errorf("response exceeds %d bytes", maxBytes)
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// Open returns a reader for a stored icon together with its MIME type.
func (i *IconService) Open(ctx context.Context, filename string) (io.ReadSeekCloser, string, error) {
	if !safeFilenamePattern.MatchString(filename) || strings.Contains(filename, "..") {
		return nil, "", NotFound("文件不存在")
	}
	upload, err := i.store.GetUploadByFilename(ctx, filename)
	if err != nil {
		return nil, "", NotFound("文件不存在")
	}
	f, err := os.Open(filepath.Join(i.uploadsDir, filename))
	if err != nil {
		return nil, "", NotFound("文件不存在")
	}
	return f, upload.Mime, nil
}

// SaveDataURI decodes a `data:` URI image (percent-encoded or base64) and
// stores it as an uploaded icon. This preserves legacy icons that were embedded
// directly in the source data instead of being hosted at a URL.
func (i *IconService) SaveDataURI(ctx context.Context, dataURI string) (*models.Upload, error) {
	data, mediaType, err := decodeDataURI(dataURI)
	if err != nil {
		return nil, err
	}
	ext := extensionForMediaType(mediaType)
	if ext == "" {
		return nil, fmt.Errorf("unsupported data URI media type %q", mediaType)
	}
	return i.SaveUpload(ctx, data, "icon"+ext)
}

// IsDataURI reports whether a value is a data: URI.
func IsDataURI(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "data:")
}

func decodeDataURI(dataURI string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(dataURI)
	if !strings.HasPrefix(trimmed, "data:") {
		return nil, "", fmt.Errorf("not a data URI")
	}
	rest := trimmed[len("data:"):]
	comma := strings.Index(rest, ",")
	if comma < 0 {
		return nil, "", fmt.Errorf("malformed data URI")
	}
	meta := rest[:comma]
	payload := rest[comma+1:]

	mediaType := strings.TrimSpace(strings.Split(meta, ";")[0])
	if mediaType == "" {
		mediaType = "text/plain"
	}

	if strings.Contains(strings.ToLower(meta), ";base64") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(payload))
		}
		if err != nil {
			return nil, "", fmt.Errorf("decode base64 data URI: %w", err)
		}
		return decoded, mediaType, nil
	}

	decoded, err := url.PathUnescape(payload)
	if err != nil {
		return nil, "", fmt.Errorf("decode data URI: %w", err)
	}
	return []byte(decoded), mediaType, nil
}

func extensionForMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/svg+xml":
		return ".svg"
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ""
	}
}

func detectImageType(data []byte, name string) (mime, ext string, ok bool) {
	sniffed := http.DetectContentType(data)
	sniffed = strings.Split(sniffed, ";")[0]
	if e, found := allowedImageTypes[sniffed]; found {
		return sniffed, e, true
	}
	// SVG is not detected by content sniffing; accept it by extension plus a
	// leading '<' marker (an XML document).
	if strings.EqualFold(filepath.Ext(name), ".svg") {
		trimmed := strings.TrimSpace(string(data[:min(len(data), 512)]))
		if strings.HasPrefix(trimmed, "<") {
			return "image/svg+xml", ".svg", true
		}
	}
	return "", "", false
}

func randomFilename(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate filename: %w", err)
	}
	return hex.EncodeToString(b) + ext, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
