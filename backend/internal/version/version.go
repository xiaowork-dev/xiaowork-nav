// Package version holds build-time version information.
//
// Values are overridden at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X .../internal/version.Version=2.0.0"
package version

import "runtime"

var (
	// Version is the semantic version of the build.
	Version = "0.1.0-dev"
	// Commit is the git commit the binary was built from.
	Commit = "unknown"
	// BuildTime is the UTC build timestamp.
	BuildTime = "unknown"
)

// GoVersion returns the Go runtime version used to build the binary.
func GoVersion() string {
	return runtime.Version()
}
