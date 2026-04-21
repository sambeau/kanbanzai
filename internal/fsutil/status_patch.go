package fsutil

import (
	"os"
	"regexp"
	"strings"
)

var (
	rePipeTable  = regexp.MustCompile(`(?i)^\| *status *\|[^|]+\|`)
	reBulletList = regexp.MustCompile(`(?i)^- *status *:`)
	reBareYAML   = regexp.MustCompile(`(?i)^status *:`)
)

// PatchStatusField reads the file at path line by line, replaces the first
// matching Status field line with newStatus, and writes the result atomically.
// Returns (false, nil) if no Status field is found, (false, err) on I/O
// failure, and (true, nil) on a successful patch.
func PatchStatusField(path string, newStatus string) (patched bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")

	// Preserve trailing newline: if the file ended with \n, Split produces an
	// empty string as the last element. We track and restore this.
	trailingNewline := len(data) > 0 && data[len(data)-1] == '\n'
	if trailingNewline && len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	patchedIdx := -1
	for i, line := range lines {
		switch {
		case rePipeTable.MatchString(line):
			lines[i] = "| Status | " + newStatus + " |"
			patchedIdx = i
		case reBulletList.MatchString(line):
			lines[i] = "- Status: " + newStatus
			patchedIdx = i
		case reBareYAML.MatchString(line):
			lines[i] = "status: " + newStatus
			patchedIdx = i
		}
		if patchedIdx >= 0 {
			break
		}
	}

	if patchedIdx < 0 {
		return false, nil
	}

	out := strings.Join(lines, "\n")
	if trailingNewline {
		out += "\n"
	}

	if err := WriteFileAtomic(path, []byte(out), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
