package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/models"
	"github.com/xiaowork-dev/xiaowork-nav/backend/internal/store"
)

// --- helpers ---

func solidPNG(t *testing.T, size int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{R: 30, G: 120, B: 220, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func transparentPNG(t *testing.T, size int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// icoWithSize builds a minimal ICO container declaring one entry of size x size.
func icoWithSize(size int) []byte {
	out := []byte{0, 0, 1, 0, 1, 0}
	out = append(out, byte(size), byte(size), 0, 0, 1, 0, 32, 0)
	out = append(out, 0, 0, 0, 0) // size in bytes
	out = append(out, 0, 0, 0, 0) // offset
	out = append(out, make([]byte, 64)...)
	return out
}

// webpVP8X builds a minimal VP8X header declaring width x height.
func webpVP8X(width, height int) []byte {
	out := make([]byte, 30)
	copy(out[0:4], "RIFF")
	copy(out[8:12], "WEBP")
	copy(out[12:16], "VP8X")
	w := width - 1
	h := height - 1
	out[24] = byte(w)
	out[25] = byte(w >> 8)
	out[26] = byte(w >> 16)
	out[27] = byte(h)
	out[28] = byte(h >> 8)
	out[29] = byte(h >> 16)
	return out
}

// newFetchTestService returns an IconService whose HTTP client is permissive so
// that httptest servers on loopback can be reached. SSRF rules themselves are
// covered by the security package tests and the URL validation cases below.
func newFetchTestService(t *testing.T) (*IconService, *store.Store, string) {
	t.Helper()
	st := newTestStore(t)
	dir := filepath.Join(t.TempDir(), "uploads")
	svc := NewIconService(st, dir)
	svc.client = &http.Client{}
	return svc, st, dir
}

// --- link parsing ---

func TestParseLinkTags(t *testing.T) {
	html := `
<html><head>
  <link rel="shortcut icon" href="/favicon.ico">
  <link rel="icon" type="image/png" sizes="32x32" href="/icon-32.png">
  <link rel="icon" type="image/png" sizes="192x192" href="/icon-192.png">
  <link rel="apple-touch-icon" sizes="180x180" href="/apple-touch.png">
  <link rel="manifest" href="/site.webmanifest">
  <link rel="stylesheet" href="/style.css">
</head></html>`

	links := parseLinkTags(html)

	icons := map[string]int{}
	var manifest string
	for _, l := range links {
		if l.manifest != "" {
			manifest = l.manifest
		}
		if l.source != "" {
			icons[l.source+" "+l.icon] = l.size
		}
	}
	if len(icons) != 4 {
		t.Fatalf("parsed %d icon links, want 4 (shortcut icon + 2 sizes + apple-touch): %v", len(icons), icons)
	}
	if size, ok := icons[models.IconSourceLinkIcon+" /icon-32.png"]; !ok || size != 32 {
		t.Errorf("missing link-icon /icon-32.png at 32: %v", icons)
	}
	if size, ok := icons[models.IconSourceLinkIcon+" /icon-192.png"]; !ok || size != 192 {
		t.Errorf("missing link-icon /icon-192.png at 192: %v", icons)
	}
	if size, ok := icons[models.IconSourceLinkIcon+" /favicon.ico"]; !ok || size != 0 {
		t.Errorf("missing shortcut-icon /favicon.ico: %v", icons)
	}
	if size, ok := icons[models.IconSourceAppleTouch+" /apple-touch.png"]; !ok || size != 180 {
		t.Errorf("missing apple-touch /apple-touch.png at 180: %v", icons)
	}
	if manifest != "/site.webmanifest" {
		t.Errorf("manifest = %q, want /site.webmanifest", manifest)
	}
}

// TestParseLinkTagsIgnoresDecorativeLinks covers rel values that contain the
// substring "icon" but are not usable tile icons.
func TestParseLinkTagsIgnoresDecorativeLinks(t *testing.T) {
	html := `
<html><head>
  <link rel="fluid-icon" href="https://example.com/fluidicon.png">
  <link rel="mask-icon" href="https://example.com/pinned.svg" color="#000">
  <link rel="apple-touch-icon-precomposed" sizes="180x180" href="/touch.png">
  <link rel="shortcut icon" href="/favicon.ico">
  <link rel="preload" as="image" href="/icons.svg">
</head></html>`

	links := parseLinkTags(html)
	if len(links) != 2 {
		t.Fatalf("parsed %d links, want 2 (apple-touch + shortcut icon): %+v", len(links), links)
	}
	if links[0].source != models.IconSourceAppleTouch || links[0].icon != "/touch.png" {
		t.Errorf("first = %+v, want the apple-touch icon", links[0])
	}
	if links[1].source != models.IconSourceLinkIcon || links[1].icon != "/favicon.ico" {
		t.Errorf("second = %+v, want the shortcut icon", links[1])
	}
}

// TestRankCandidatesOrdersSourcesByQuality guards the ordering rule: a brand
// guess is an SVG, so a naive format tiebreak would let it outrank the site's
// own favicon.ico. A larger declared icon must always come first.
func TestRankCandidatesOrdersSourcesByQuality(t *testing.T) {
	candidates := []iconCandidate{
		{URL: "https://cdn.simpleicons.org/github", Source: models.IconSourceBrandGuess, name: "github.svg", score: brandGuessScore},
		{URL: "https://example.com/favicon.ico", Source: models.IconSourceFaviconICO, name: "https://example.com/favicon.ico", score: faviconICOScore},
		{URL: "https://example.com/small.png", Source: models.IconSourceLinkIcon, name: "https://example.com/small.png", score: 32},
		{URL: "https://example.com/big.png", Source: models.IconSourceManifest, name: "https://example.com/big.png", score: 512},
		// A duplicate of the best candidate must be dropped.
		{URL: "https://example.com/big.png", Source: models.IconSourceManifest, name: "https://example.com/big.png", score: 512},
	}

	ranked := rankCandidates(candidates)
	if len(ranked) != 4 {
		t.Fatalf("ranked %d candidates, want 4 after de-duplication", len(ranked))
	}
	wantSources := []string{
		models.IconSourceManifest,
		models.IconSourceLinkIcon,
		models.IconSourceFaviconICO,
		models.IconSourceBrandGuess,
	}
	for i, want := range wantSources {
		if ranked[i].Source != want {
			t.Errorf("ranked[%d] = %q, want %q", i, ranked[i].Source, want)
		}
	}
}

// TestFetchIconReadsIconLinksFromOversizedPages covers real home pages that are
// far larger than the discovery budget (github.com is ~580 KB).
func TestFetchIconReadsIconLinksFromOversizedPages(t *testing.T) {
	icon := solidPNG(t, 128)
	filler := strings.Repeat("<!-- padding -->", 80_000) // ~1.2 MB of body

	site := newIconSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><head><link rel="icon" sizes="128x128" href="/icon.png"></head><body>%s</body></html>`, filler)
		case "/icon.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(icon)
		default:
			http.NotFound(w, r)
		}
	})

	svc, _, _ := newFetchTestService(t)
	result, err := svc.FetchIcon(context.Background(), site.server.URL, false)
	if err != nil {
		t.Fatalf("FetchIcon() error = %v", err)
	}
	if result.Source != models.IconSourceLinkIcon {
		t.Errorf("source = %q, want the declared link icon from a truncated page", result.Source)
	}
}

func TestParseSizes(t *testing.T) {
	cases := map[string]int{
		"":                     0,
		"any":                  0,
		"16x16":                16,
		"16x16 32x32":          32,
		"192x192 512x512":      512,
		"512x512 192x192":      512,
		"180x180":              180,
		"not-a-size":           0,
		"48x48 96x96 144x144":  144,
	}
	for raw, want := range cases {
		if got := parseSizes(raw); got != want {
			t.Errorf("parseSizes(%q) = %d, want %d", raw, got, want)
		}
	}
}

// --- quality gate ---

func TestImageQuality(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		mime string
		ok   bool
	}{
		{"32x32 png", solidPNG(t, 32), "image/png", true},
		{"8x8 png", solidPNG(t, 8), "image/png", false},
		{"blank png", transparentPNG(t, 32), "image/png", false},
		{"ico 32", icoWithSize(32), "image/x-icon", true},
		{"ico 8", icoWithSize(8), "image/x-icon", false},
		{"webp 128", webpVP8X(128, 128), "image/webp", true},
		{"webp 4", webpVP8X(4, 4), "image/webp", false},
		{"tiny svg", []byte(`<svg/>`), "image/svg+xml", false},
		{"real svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0h24v24H0z"/></svg>`), "image/svg+xml", true},
	}
	for _, tc := range cases {
		ok, reason := imageQuality(tc.data, tc.mime)
		if ok != tc.ok {
			t.Errorf("%s: imageQuality() = %v (%s), want %v", tc.name, ok, reason, tc.ok)
		}
	}
}

func TestDetectImageTypeRejectsHTMLAsSVG(t *testing.T) {
	html := []byte(`<!doctype html><html><body><svg xmlns="http://www.w3.org/2000/svg"></svg></body></html>`)
	if _, _, ok := detectImageType(html, "icon.svg"); ok {
		t.Error("an HTML document must not be accepted as an SVG icon")
	}
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/></svg>`)
	if _, ext, ok := detectImageType(svg, "icon.svg"); !ok || ext != ".svg" {
		t.Errorf("a real SVG must be accepted, got ok=%v ext=%q", ok, ext)
	}
}

// --- brand guess ---

func TestBrandSlugForHost(t *testing.T) {
	cases := map[string]string{
		"github.com":       "github",
		"www.github.com":   "github",
		"GITHUB.COM":       "github",
		"github.com:443":   "github",
		"github.com.cn":    "github",
		"192.168.1.1":      "",
		"localhost":        "",
		"notaslug-zzz.com": "",
		"example.com":      "",
	}
	for host, want := range cases {
		if got := brandSlugForHost(host); got != want {
			t.Errorf("brandSlugForHost(%q) = %q, want %q", host, got, want)
		}
	}
}

func TestBrandGuessCandidateIsLabelled(t *testing.T) {
	cand := brandGuessCandidate("github.com")
	if cand.Source != models.IconSourceBrandGuess {
		t.Errorf("source = %q, want %q", cand.Source, models.IconSourceBrandGuess)
	}
	if cand.URL != "https://cdn.simpleicons.org/github" {
		t.Errorf("url = %q", cand.URL)
	}
	if cand.name != "github.svg" {
		t.Errorf("name = %q, want github.svg so the SVG sniffer applies", cand.name)
	}
	if cand := brandGuessCandidate("notaslug-zzz.com"); cand.URL != "" {
		t.Errorf("unknown brand must not produce a candidate, got %q", cand.URL)
	}
}

// --- end-to-end fetch against a local server ---

type iconSite struct {
	server   *httptest.Server
	requests map[string]int
}

func newIconSite(t *testing.T, handler http.HandlerFunc) *iconSite {
	t.Helper()
	site := &iconSite{requests: map[string]int{}}
	site.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site.requests[r.URL.Path]++
		handler(w, r)
	}))
	t.Cleanup(site.server.Close)
	return site
}

func (s *iconSite) hits(path string) int { return s.requests[path] }

func TestFetchIconPrefersLargerDeclaredIconAndCaches(t *testing.T) {
	big := solidPNG(t, 512)
	small := solidPNG(t, 32)
	manifest := solidPNG(t, 192)

	site := newIconSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<html><head>
              <link rel="manifest" href="/manifest.json">
              <link rel="icon" sizes="32x32" href="/small.png">
              <link rel="icon" sizes="512x512" href="/big.png">
            </head></html>`)
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/manifest+json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"icons": []map[string]string{{"src": "/manifest.png", "sizes": "192x192", "type": "image/png"}},
			})
		case "/big.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(big)
		case "/small.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(small)
		case "/manifest.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(manifest)
		default:
			http.NotFound(w, r)
		}
	})

	svc, st, _ := newFetchTestService(t)
	ctx := context.Background()

	result, err := svc.FetchIcon(ctx, site.server.URL, false)
	if err != nil {
		t.Fatalf("FetchIcon() error = %v", err)
	}
	if result.Source != models.IconSourceLinkIcon {
		t.Errorf("source = %q, want %q (the 512px link icon wins)", result.Source, models.IconSourceLinkIcon)
	}
	if result.IconType != "auto" {
		t.Errorf("icon_type = %q, want auto", result.IconType)
	}
	if !strings.HasPrefix(result.IconValue, UploadURLPrefix) {
		t.Errorf("icon_value = %q, want a local upload path", result.IconValue)
	}

	// The cached file is recorded with its source URL and auto origin.
	filename := strings.TrimPrefix(result.IconValue, UploadURLPrefix)
	upload, err := st.GetUploadByFilename(ctx, filename)
	if err != nil {
		t.Fatalf("GetUploadByFilename() error = %v", err)
	}
	if upload.Origin != models.UploadOriginAuto {
		t.Errorf("origin = %q, want auto", upload.Origin)
	}
	if upload.SourceURL != site.server.URL+"/big.png" {
		t.Errorf("source_url = %q, want %q", upload.SourceURL, site.server.URL+"/big.png")
	}

	// A second fetch reuses the cache: no new request for the icon.
	before := site.hits("/big.png")
	again, err := svc.FetchIcon(ctx, site.server.URL, false)
	if err != nil {
		t.Fatalf("FetchIcon() second call error = %v", err)
	}
	if again.IconValue != result.IconValue {
		t.Errorf("cached fetch returned %q, want %q", again.IconValue, result.IconValue)
	}
	if after := site.hits("/big.png"); after != before {
		t.Errorf("cached fetch downloaded the icon again (%d -> %d)", before, after)
	}

	// force=true refreshes it.
	forced, err := svc.FetchIcon(ctx, site.server.URL, true)
	if err != nil {
		t.Fatalf("FetchIcon(force) error = %v", err)
	}
	if site.hits("/big.png") <= before {
		t.Error("force=true must re-download the icon")
	}
	if forced.IconValue == result.IconValue {
		t.Error("force=true must store a fresh file")
	}
}

func TestFetchIconSkipsLowQualityAndFallsBackToFavicon(t *testing.T) {
	site := newIconSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><link rel="icon" sizes="8x8" href="/tiny.png"></head></html>`)
		case "/tiny.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(solidPNG(t, 8))
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/x-icon")
			_, _ = w.Write(icoWithSize(32))
		default:
			http.NotFound(w, r)
		}
	})

	svc, _, _ := newFetchTestService(t)
	result, err := svc.FetchIcon(context.Background(), site.server.URL, false)
	if err != nil {
		t.Fatalf("FetchIcon() error = %v", err)
	}
	if result.Source != models.IconSourceFaviconICO {
		t.Errorf("source = %q, want %q after the tiny icon is rejected", result.Source, models.IconSourceFaviconICO)
	}
}

func TestFetchIconFailsWhenNothingUsableExists(t *testing.T) {
	site := newIconSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head><link rel="icon" href="/icon.png"></head></html>`)
		case "/icon.png":
			// HTML pretending to be an image.
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body>not an image</body></html>")
		default:
			http.NotFound(w, r)
		}
	})

	svc, st, _ := newFetchTestService(t)
	_, err := svc.FetchIcon(context.Background(), site.server.URL, false)
	if err == nil {
		t.Fatal("FetchIcon() must fail when no usable icon exists")
	}

	// Nothing may have been stored for a failed fetch.
	uploads, listErr := st.ListUploads(context.Background())
	if listErr != nil {
		t.Fatalf("ListUploads() error = %v", listErr)
	}
	if len(uploads) != 0 {
		t.Errorf("stored %d uploads for a failed fetch, want 0", len(uploads))
	}
}

func TestFetchIconRejectsUnsafeURLs(t *testing.T) {
	svc, _, _ := newFetchTestService(t)
	ctx := context.Background()

	for _, raw := range []string{
		"http://user:pass@example.com/icon.png",
		"http://example.com:8080/icon.png",
		"file:///etc/passwd",
		"",
	} {
		if _, err := svc.FetchIcon(ctx, raw, false); err == nil {
			t.Errorf("FetchIcon(%q) = nil error, want rejection", raw)
		}
	}
}

func TestFetchIconIgnoresUnsafeIconReferences(t *testing.T) {
	// The page advertises icons on a non-standard port and with credentials;
	// both must be dropped and the plain favicon used instead.
	site := newIconSite(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><head>
              <link rel="icon" href="http://evil.internal:8080/icon.png">
              <link rel="icon" href="http://user:pass@example.com/icon.png">
            </head></html>`)
		case "/favicon.ico":
			w.Header().Set("Content-Type", "image/x-icon")
			_, _ = w.Write(icoWithSize(32))
		default:
			http.NotFound(w, r)
		}
	})

	svc, _, _ := newFetchTestService(t)
	result, err := svc.FetchIcon(context.Background(), site.server.URL, false)
	if err != nil {
		t.Fatalf("FetchIcon() error = %v", err)
	}
	if result.Source != models.IconSourceFaviconICO {
		t.Errorf("source = %q, want the favicon fallback", result.Source)
	}
}
