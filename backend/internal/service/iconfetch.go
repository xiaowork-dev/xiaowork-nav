package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/security"
)

// Automatic icon fetching limits.
const (
	// autoIconMaxBytes caps a single auto-fetched icon (user uploads keep MaxIconBytes).
	autoIconMaxBytes = 256 << 10
	// manifestMaxBytes caps a Web App Manifest document.
	manifestMaxBytes = 256 << 10
	// minIconPixels rejects icons too small to look good in a tile.
	minIconPixels = 16
	// iconCandidateTimeout bounds one candidate download.
	iconCandidateTimeout = 10 * time.Second
	// iconTotalBudget bounds one whole fetch (page + every candidate).
	iconTotalBudget = 25 * time.Second
	// transparentCheckPixels caps the work spent scanning for a blank PNG.
	transparentCheckPixels = 512 * 512
)

// FetchResult describes a successful automatic icon fetch.
type FetchResult struct {
	IconType  string `json:"icon_type"`
	IconValue string `json:"icon_value"`
	Source    string `json:"icon_source"`
}

// iconCandidate is one place an icon might be downloaded from.
type iconCandidate struct {
	URL    string
	Source string
	// name is the virtual file name used for format sniffing.
	name  string
	score int
}

// formatRank prefers vector and modern raster formats over legacy ICO.
func formatRank(raw string) int {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, ".svg"):
		return 4
	case strings.Contains(lower, ".png"):
		return 3
	case strings.Contains(lower, ".webp"):
		return 2
	case strings.Contains(lower, ".ico"):
		return 1
	default:
		return 0
	}
}

// Candidate scoring. Real icon declarations always outrank the site's own
// favicon, which in turn outranks a brand guess, so a guess can never displace
// a better source even when its format would sort higher.
const (
	brandGuessScore  = -1
	faviconICOScore  = 0
	manifestScore    = 192
	appleTouchScore  = 180
	defaultIconScore = 48
)

// sourceDefaultSize is the size hint used when an icon declares none.
func sourceDefaultSize(source string) int {
	switch source {
	case models.IconSourceManifest:
		return manifestScore
	case models.IconSourceAppleTouch:
		return appleTouchScore
	default:
		return defaultIconScore
	}
}

// parseSizes returns the largest declared size ("192x192 512x512" -> 512).
func parseSizes(raw string) int {
	best := 0
	for _, part := range strings.Fields(strings.ToLower(raw)) {
		part = strings.TrimSpace(part)
		if part == "" || part == "any" {
			continue
		}
		dims := strings.SplitN(part, "x", 2)
		n, err := strconv.Atoi(strings.TrimSpace(dims[0]))
		if err != nil || n <= 0 {
			continue
		}
		if n > best {
			best = n
		}
	}
	return best
}

// FetchIcon fetches and caches an icon for a site, returning the icon reference
// to store on a bookmark. It never touches a bookmark itself.
//
// Candidate order is decided by quality (declared size, then format), which is
// what the requirement asks for: the clearest tile-friendly icon wins, with the
// site's own favicon and a conservative brand guess as last resorts.
func (i *IconService) FetchIcon(ctx context.Context, siteURL string, force bool) (*FetchResult, error) {
	if err := security.ValidateFetchURL(siteURL); err != nil {
		return nil, New(KindUnsupported, "invalid_url_scheme", "URL 非法：仅允许 http/https",
			FieldError{Field: "url", Issue: err.Error()})
	}
	pageURL, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil {
		return nil, New(KindUnsupported, "invalid_url_scheme", "URL 非法")
	}

	ctx, cancel := context.WithTimeout(ctx, iconTotalBudget)
	defer cancel()

	candidates := i.discoverCandidates(ctx, pageURL)

	var lastErr error
	for _, cand := range candidates {
		if ctx.Err() != nil {
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			break
		}
		result, err := i.storeCandidate(ctx, cand, force)
		if err != nil {
			lastErr = err
			continue
		}
		return result, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("未找到可用图标")
	}
	return nil, New(KindNotFound, "not_found", "未能获取站点图标："+lastErr.Error())
}

// storeCandidate downloads one candidate, validates it and caches it locally.
// An already-cached icon for the same source URL is reused unless force is set.
func (i *IconService) storeCandidate(ctx context.Context, cand iconCandidate, force bool) (*FetchResult, error) {
	if !force {
		if upload, err := i.store.FindUploadBySourceURL(ctx, cand.URL); err == nil {
			if _, statErr := os.Stat(filepath.Join(i.uploadsDir, upload.Filename)); statErr == nil {
				return &FetchResult{
					IconType:  "auto",
					IconValue: UploadURL(upload.Filename),
					Source:    cand.Source,
				}, nil
			}
		}
	}

	data, err := i.download(ctx, cand.URL, autoIconMaxBytes, false, false)
	if err != nil {
		return nil, err
	}
	mime, ext, ok := detectImageType(data, cand.name)
	if !ok {
		return nil, fmt.Errorf("不是可识别的图片（%s）", cand.URL)
	}
	if ok, reason := imageQuality(data, mime); !ok {
		return nil, fmt.Errorf("图标质量不合格：%s（%s）", reason, cand.URL)
	}

	upload, err := i.saveUpload(ctx, data, "icon"+ext, cand.URL, models.UploadOriginAuto)
	if err != nil {
		return nil, err
	}
	return &FetchResult{
		IconType:  "auto",
		IconValue: UploadURL(upload.Filename),
		Source:    cand.Source,
	}, nil
}

// discoverCandidates collects every plausible icon location for a site.
func (i *IconService) discoverCandidates(ctx context.Context, pageURL *url.URL) []iconCandidate {
	out := make([]iconCandidate, 0, 8)

	if body, err := i.download(ctx, pageURL.String(), maxHTMLLength, true, true); err == nil {
		links := parseLinkTags(string(body))
		for _, l := range links {
			if l.manifest == "" {
				continue
			}
			out = append(out, i.manifestCandidates(ctx, pageURL, l.manifest)...)
		}
		for _, l := range links {
			if l.icon == "" {
				continue
			}
			resolved := resolveRef(pageURL, l.icon)
			if resolved == "" {
				continue
			}
			size := l.size
			if size == 0 {
				size = sourceDefaultSize(l.source)
			}
			out = append(out, iconCandidate{URL: resolved, Source: l.source, name: resolved, score: size})
		}
	}

	// The site's own favicon is always worth trying, but only after any icon the
	// page actually declares.
	favicon := pageURL.Scheme + "://" + pageURL.Host + "/favicon.ico"
	out = append(out, iconCandidate{URL: favicon, Source: models.IconSourceFaviconICO, name: favicon, score: faviconICOScore})

	// Conservative brand guess: only when the site itself offers nothing.
	if brand := brandGuessCandidate(pageURL.Host); brand.URL != "" {
		out = append(out, brand)
	}

	return rankCandidates(out)
}

// rankCandidates orders candidates best-first and removes duplicates.
//
// Ordering is by declared/assumed size, then by format, then by URL, which
// makes the result deterministic and keeps the brand guess (score -1) last.
func rankCandidates(in []iconCandidate) []iconCandidate {
	sort.SliceStable(in, func(a, b int) bool {
		if in[a].score != in[b].score {
			return in[a].score > in[b].score
		}
		ra, rb := formatRank(in[a].name), formatRank(in[b].name)
		if ra != rb {
			return ra > rb
		}
		return in[a].URL < in[b].URL
	})
	return dedupeCandidates(in)
}

func dedupeCandidates(in []iconCandidate) []iconCandidate {
	seen := make(map[string]bool, len(in))
	out := make([]iconCandidate, 0, len(in))
	for _, c := range in {
		if seen[c.URL] {
			continue
		}
		seen[c.URL] = true
		out = append(out, c)
	}
	return out
}

// manifestCandidates fetches a Web App Manifest and turns its icons into candidates.
func (i *IconService) manifestCandidates(ctx context.Context, pageURL *url.URL, ref string) []iconCandidate {
	manifestURL := resolveRef(pageURL, ref)
	if manifestURL == "" {
		return nil
	}
	data, err := i.download(ctx, manifestURL, manifestMaxBytes, false, false)
	if err != nil {
		return nil
	}
	var manifest struct {
		Icons []struct {
			Src   string `json:"src"`
			Sizes string `json:"sizes"`
			Type  string `json:"type"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	out := make([]iconCandidate, 0, len(manifest.Icons))
	for _, icon := range manifest.Icons {
		resolved := resolveRef(pageURL, icon.Src)
		if resolved == "" {
			continue
		}
		size := parseSizes(icon.Sizes)
		if size == 0 {
			size = sourceDefaultSize(models.IconSourceManifest)
		}
		out = append(out, iconCandidate{
			URL:    resolved,
			Source: models.IconSourceManifest,
			name:   resolved,
			score:  size,
		})
	}
	return out
}

// parsedLink is one <link> tag reduced to the fields we care about.
type parsedLink struct {
	icon     string
	source   string
	size     int
	manifest string
}

var (
	linkTagPattern   = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	linkHrefPattern  = regexp.MustCompile(`(?is)href\s*=\s*["']([^"']*)["']`)
	linkRelPattern   = regexp.MustCompile(`(?is)rel\s*=\s*["']?([^"'>]+)["']?`)
	linkSizesPattern = regexp.MustCompile(`(?is)sizes\s*=\s*["']([^"']*)["']`)
)

// parseLinkTags extracts icon and manifest links from an HTML document.
//
// The `rel` attribute is matched token by token so that decorative links such
// as `mask-icon` (a monochrome mask) and `fluid-icon` are not mistaken for a
// usable tile icon.
func parseLinkTags(html string) []parsedLink {
	tags := linkTagPattern.FindAllString(html, -1)
	out := make([]parsedLink, 0, len(tags))
	for _, tag := range tags {
		rel := ""
		if m := linkRelPattern.FindStringSubmatch(tag); len(m) > 1 {
			rel = strings.ToLower(strings.TrimSpace(m[1]))
		}
		if rel == "" {
			continue
		}
		href := ""
		if m := linkHrefPattern.FindStringSubmatch(tag); len(m) > 1 {
			href = strings.TrimSpace(m[1])
		}
		if href == "" {
			continue
		}
		size := 0
		if m := linkSizesPattern.FindStringSubmatch(tag); len(m) > 1 {
			size = parseSizes(m[1])
		}

		tokens := strings.Fields(rel)
		switch {
		case containsToken(tokens, "manifest"):
			out = append(out, parsedLink{manifest: href})
		case containsPrefix(tokens, "apple-touch-icon"):
			out = append(out, parsedLink{icon: href, source: models.IconSourceAppleTouch, size: size})
		case containsToken(tokens, "icon"):
			out = append(out, parsedLink{icon: href, source: models.IconSourceLinkIcon, size: size})
		}
	}
	return out
}

func containsToken(tokens []string, want string) bool {
	for _, t := range tokens {
		if t == want {
			return true
		}
	}
	return false
}

func containsPrefix(tokens []string, prefix string) bool {
	for _, t := range tokens {
		if strings.HasPrefix(t, prefix) {
			return true
		}
	}
	return false
}

// resolveRef resolves a possibly relative reference and validates it as a
// fetchable URL. Unsafe targets (bad scheme, credentials, odd port) are dropped.
func resolveRef(base *url.URL, ref string) string {
	parsed, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ""
	}
	raw := base.ResolveReference(parsed).String()
	if err := security.ValidateFetchURL(raw); err != nil {
		return ""
	}
	return raw
}

// brandGuessCandidate maps a host to a verified Simple Icons slug.
//
// This is deliberately the last resort and is labelled as a guess in the UI:
// the requirement is explicit that an icon of unverified origin must not be
// presented as an official one.
func brandGuessCandidate(host string) iconCandidate {
	slug := brandSlugForHost(host)
	if slug == "" {
		return iconCandidate{}
	}
	cdn := SimpleIconURL(slug)
	if cdn == "" {
		return iconCandidate{}
	}
	return iconCandidate{
		URL:    cdn,
		Source: models.IconSourceBrandGuess,
		// The CDN serves an SVG without a file extension; give the sniffer a
		// virtual .svg name so the format check still applies.
		name:  slug + ".svg",
		score: brandGuessScore,
	}
}

// multiPartSuffixes are second-level labels that belong to the public suffix
// rather than the registrable name (e.g. "example.com.cn" -> "example").
var multiPartSuffixes = map[string]bool{
	"com.cn": true, "net.cn": true, "org.cn": true, "gov.cn": true, "edu.cn": true,
	"co.uk": true, "org.uk": true, "ac.uk": true, "gov.uk": true,
	"com.au": true, "net.au": true, "org.au": true,
	"com.br": true, "com.hk": true, "com.tw": true, "com.sg": true, "co.jp": true,
	"co.kr": true, "com.mx": true, "com.tr": true, "co.in": true, "com.ru": true,
}

// brandSlugForHost returns a verified Simple Icons slug for a host, or "".
func brandSlugForHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(host, ".")
	host = strings.TrimPrefix(host, "www.")
	if host == "" || net.ParseIP(host) != nil {
		return ""
	}

	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return ""
	}
	index := len(labels) - 2
	if len(labels) >= 3 && multiPartSuffixes[strings.Join(labels[len(labels)-2:], ".")] {
		index = len(labels) - 3
	}
	if index < 0 {
		return ""
	}
	slug := labels[index]
	if !IsSimpleIconSlug(slug) {
		return ""
	}
	return slug
}

// imageQuality rejects icons that would look bad in a navigation tile:
// unreadably small images, blank images, and empty SVG documents.
func imageQuality(data []byte, mime string) (bool, string) {
	switch mime {
	case "image/png":
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return false, "PNG 无法解析"
		}
		if cfg.Width < minIconPixels || cfg.Height < minIconPixels {
			return false, fmt.Sprintf("尺寸过小 %dx%d", cfg.Width, cfg.Height)
		}
		if isBlankPNG(data, cfg.Width, cfg.Height) {
			return false, "图像完全透明"
		}
		return true, ""
	case "image/jpeg":
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return false, "JPEG 无法解析"
		}
		if cfg.Width < minIconPixels || cfg.Height < minIconPixels {
			return false, fmt.Sprintf("尺寸过小 %dx%d", cfg.Width, cfg.Height)
		}
		return true, ""
	case "image/gif":
		cfg, err := gif.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return false, "GIF 无法解析"
		}
		if cfg.Width < minIconPixels || cfg.Height < minIconPixels {
			return false, fmt.Sprintf("尺寸过小 %dx%d", cfg.Width, cfg.Height)
		}
		return true, ""
	case "image/webp":
		w, h, ok := webpDimensions(data)
		if !ok {
			return false, "WebP 无法解析"
		}
		if w < minIconPixels || h < minIconPixels {
			return false, fmt.Sprintf("尺寸过小 %dx%d", w, h)
		}
		return true, ""
	case "image/x-icon", "image/vnd.microsoft.icon":
		w, h, ok := icoDimensions(data)
		if !ok {
			return false, "ICO 无法解析"
		}
		if w < minIconPixels || h < minIconPixels {
			return false, fmt.Sprintf("尺寸过小 %dx%d", w, h)
		}
		return true, ""
	case "image/svg+xml":
		if len(data) < 64 {
			return false, "SVG 内容过小"
		}
		return true, ""
	}
	return false, "未知图片格式"
}

// isBlankPNG reports whether every pixel is fully transparent. It is skipped
// for very large images to bound the CPU cost of a fetch.
func isBlankPNG(data []byte, w, h int) bool {
	if w*h > transparentCheckPixels {
		return false
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				return false
			}
		}
	}
	return true
}

// webpDimensions reads the canvas size from a WebP container without pulling in
// an extra image dependency.
func webpDimensions(data []byte) (int, int, bool) {
	if len(data) < 30 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, false
	}
	switch string(data[12:16]) {
	case "VP8X":
		// Canvas size is stored as 24-bit little endian minus one.
		w := int(data[24]) | int(data[25])<<8 | int(data[26])<<16
		h := int(data[27]) | int(data[28])<<8 | int(data[29])<<16
		return w + 1, h + 1, true
	case "VP8L":
		if len(data) < 25 || data[20] != 0x2f {
			return 0, 0, false
		}
		bits := uint32(data[21]) | uint32(data[22])<<8 | uint32(data[23])<<16 | uint32(data[24])<<24
		return int(bits&0x3fff) + 1, int((bits>>14)&0x3fff) + 1, true
	case "VP8 ":
		if len(data) < 30 || data[23] != 0x9d || data[24] != 0x01 || data[25] != 0x2a {
			return 0, 0, false
		}
		w := int(data[26]) | int(data[27])<<8
		h := int(data[28]) | int(data[29])<<8
		return w & 0x3fff, h & 0x3fff, true
	}
	return 0, 0, false
}

// icoDimensions returns the largest image size declared in an ICO directory.
func icoDimensions(data []byte) (int, int, bool) {
	if len(data) < 6 || data[0] != 0 || data[1] != 0 || data[2] != 1 || data[3] != 0 {
		return 0, 0, false
	}
	count := int(data[4]) | int(data[5])<<8
	if count == 0 || len(data) < 6+count*16 {
		return 0, 0, false
	}
	bestW, bestH := 0, 0
	for idx := 0; idx < count; idx++ {
		entry := data[6+idx*16 : 6+idx*16+16]
		w, h := int(entry[0]), int(entry[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		if w*h > bestW*bestH {
			bestW, bestH = w, h
		}
	}
	if bestW == 0 {
		return 0, 0, false
	}
	return bestW, bestH, true
}

// looksLikeSVG applies a stricter SVG check than a bare "<svg" search: HTML
// documents (which may contain an inline <svg>) are rejected outright.
func looksLikeSVG(data []byte) bool {
	head := strings.ToLower(string(data[:min(len(data), 1024)]))
	if strings.Contains(head, "<html") || strings.Contains(head, "<!doctype html") {
		return false
	}
	return strings.Contains(head, "<svg")
}
