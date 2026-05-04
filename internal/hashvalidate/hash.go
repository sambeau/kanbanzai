package hashvalidate

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// HashLength is the number of hex characters in the hash tag.
const HashLength = 2

// HashLine computes a 2-character uppercase hex hash tag for the given line content.
// Trailing newline characters are stripped before hashing.
// The hash is derived from SHA-256 truncated to HashLength characters.
func HashLine(content string) string {
	// Strip a single trailing newline if present.
	content = strings.TrimSuffix(content, "\n")

	h := sha256.Sum256([]byte(content))
	hex := fmt.Sprintf("%x", h)
	return strings.ToUpper(hex[:HashLength])
}
