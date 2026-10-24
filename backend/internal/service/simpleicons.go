package service

import (
	_ "embed"
	"strings"
	"sync"
)

// simpleIconsData is the verified Simple Icons slug list.
//
// Source: https://raw.githubusercontent.com/simple-icons/simple-icons/develop/slugs.md
// Retrieved: 2026-10-24. Regenerate by extracting the backtick-quoted slugs from
// that file, lowercasing and de-duplicating them (see
// docs/legacy-data-migration-report.md).
//
//go:embed data/simpleicons.txt
var simpleIconsData string

var (
	simpleIconsOnce sync.Once
	simpleIconsSet  map[string]struct{}
)

func simpleIcons() map[string]struct{} {
	simpleIconsOnce.Do(func() {
		simpleIconsSet = make(map[string]struct{}, 4096)
		for _, line := range strings.Split(simpleIconsData, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			simpleIconsSet[line] = struct{}{}
		}
	})
	return simpleIconsSet
}

// IsSimpleIconSlug reports whether value is a verified Simple Icons slug.
//
// This deliberately rejects arbitrary strings: legacy `si` values such as
// "微软" or "yongge" are not slugs and must not be turned into CDN URLs.
func IsSimpleIconSlug(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return false
	}
	_, ok := simpleIcons()[v]
	return ok
}

// SimpleIconURL returns the CDN URL for a verified slug, or "" if unknown.
func SimpleIconURL(value string) string {
	if !IsSimpleIconSlug(value) {
		return ""
	}
	return "https://cdn.simpleicons.org/" + strings.ToLower(strings.TrimSpace(value))
}

// SimpleIconCount returns how many verified slugs are bundled.
func SimpleIconCount() int {
	return len(simpleIcons())
}
