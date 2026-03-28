// Package buildinfo exposes build-time metadata injected via -ldflags.
package buildinfo

var (
	// Version is the semantic version or "dev" when built without ldflags.
	Version = "dev"

	// GitSHA is the full git commit SHA or "unknown" when built without ldflags.
	GitSHA = "unknown"

	// BuildTime is the UTC build timestamp or "unknown" when built without ldflags.
	BuildTime = "unknown"

	// Dirty is "true" if the working tree had uncommitted changes, "false" otherwise.
	Dirty = "false"
)
