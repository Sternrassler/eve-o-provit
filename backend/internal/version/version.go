// Package version exposes the application version.
package version

// Version is the API version reported by the /version endpoint. It is injected
// at build time via -ldflags from the top released CHANGELOG.md entry (the
// SemVer single source of truth); "dev" is the default for local builds.
var Version = "dev"
