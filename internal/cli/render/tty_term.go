package render

import "golang.org/x/term"

// isTerminal wraps golang.org/x/term.IsTerminal for testability.
// In platform-specific builds, this could use os.Stdin or other fd directly.
func isTerminal(fd int) bool {
	return term.IsTerminal(fd)
}
