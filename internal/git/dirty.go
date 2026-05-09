package git

import (
	"fmt"
	"strings"
)

// CheckKbzDirty returns the list of modified or untracked files under
// .kbz/state/, .kbz/index/, and .kbz/context/ in the given repo root.
// Returns (nil, nil) when the working tree is clean for those paths.
func CheckKbzDirty(repoRoot string) ([]string, error) {
	// --untracked-files=all forces individual file listing rather than
	// directory-level summary (e.g. ".kbz/state/tasks/TASK-001.yaml" instead
	// of ".kbz/state/").
	out, err := runGitCmd(repoRoot, "status", "--porcelain", "--untracked-files=all",
		"--", ".kbz/state/", ".kbz/index/", ".kbz/context/")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	output := strings.TrimSpace(out)
	if output == "" {
		return nil, nil
	}

	lines := strings.Split(output, "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		// git status --porcelain format: "XY PATH" (X=index, Y=worktree, then space, then path).
		// Path starts at column 3 (0-indexed).
		if len(line) >= 4 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}
