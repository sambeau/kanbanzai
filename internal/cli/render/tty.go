// Package render provides TTY-aware rendering for CLI output.
// It handles Unicode/ASCII symbol mapping, ANSI colour codes, and TTY detection.
package render

// TTYDetector reports whether output is a terminal (TTY).
// The interface is injectable so tests can control TTY behaviour.
type TTYDetector interface {
	IsTTY() bool
}

// termTTY is the real implementation using golang.org/x/term.
type termTTY struct {
	// fd is the file descriptor to check. Defaults to stdin.
	fd int
}

// NewTermTTY creates a TTYDetector using the terminal's stdout file descriptor.
// Callers can pass any fd that supports IsTerminal checking.
func NewTermTTY(fd int) TTYDetector {
	return &termTTY{fd: fd}
}

// IsTTY returns true if the file descriptor is a terminal.
func (t *termTTY) IsTTY() bool {
	// This function is replaced at link time by the platform-specific implementation.
	// For this project, we use the golang.org/x/term package.
	return isTerminal(t.fd)
}

// StaticTTY is a TTYDetector that always returns a fixed value.
// Useful for testing.
type StaticTTY struct {
	Value bool
}

// IsTTY returns the static value.
func (s StaticTTY) IsTTY() bool {
	return s.Value
}
