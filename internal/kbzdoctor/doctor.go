// Package kbzdoctor implements the `kbz doctor` subcommand which validates
// an in-place Kanbanzai install. It checks for required files, managed markers,
// and ghost files not tracked by the install system.
package kbzdoctor

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// managedMarkerPrefix is the prefix for managed markers in markdown files.
const managedMarkerPrefix = "<!-- kanbanzai-managed: v"

// skillManagedMarker is the YAML comment marker for managed skill files.
const skillManagedMarker = "# kanbanzai-managed:"

// Doctor validates a Kanbanzai install.
type Doctor struct {
	stdout io.Writer
	stderr io.Writer
}

// New creates a new Doctor.
func New(stdout, stderr io.Writer) *Doctor {
	return &Doctor{stdout: stdout, stderr: stderr}
}

// CheckResult holds the outcome of a single check.
type CheckResult struct {
	Path    string
	Ok      bool
	Missing bool
	Warning string
}

// Run validates the install rooted at repoRoot. It returns an error only on
// I/O failures. Validation failures are reported via the returned results
// and determine the exit code (0 = all pass, 1 = missing required).
func (d *Doctor) Run(repoRoot string) ([]CheckResult, error) {
	var results []CheckResult

	kbzDir := filepath.Join(repoRoot, ".kbz")

	// Check .kbz/ directory.
	if _, err := os.Stat(kbzDir); os.IsNotExist(err) {
		fmt.Fprintln(d.stdout, "No .kbz/ directory found — this does not appear to be a Kanbanzai project.")
		return []CheckResult{{Path: ".kbz/", Missing: true, Warning: ".kbz/ directory not found"}}, nil
	}

	// Check required files.
	requiredFiles := []struct {
		path    string
		marker  string // empty means just check existence
		managed bool
	}{
		{filepath.Join(repoRoot, "AGENTS.md"), managedMarkerPrefix, true},
		{filepath.Join(repoRoot, ".github", "copilot-instructions.md"), managedMarkerPrefix, true},
		{filepath.Join(repoRoot, ".mcp.json"), `"kanbanzai-managed"`, true},
		{filepath.Join(repoRoot, ".zed", "settings.json"), `"kanbanzai-managed"`, true},
		{filepath.Join(kbzDir, "config.yaml"), "", false},
		{filepath.Join(kbzDir, "stage-bindings.yaml"), skillManagedMarker, true},
		{filepath.Join(kbzDir, ".init-complete"), "", false},
	}

	for _, rf := range requiredFiles {
		r := CheckResult{Path: rf.path}
		data, err := os.ReadFile(rf.path)
		if os.IsNotExist(err) {
			r.Missing = true
			r.Warning = "missing"
			results = append(results, r)
			continue
		}
		if err != nil {
			r.Warning = fmt.Sprintf("cannot read: %v", err)
			results = append(results, r)
			continue
		}

		if rf.managed && rf.marker != "" {
			if !containsMarker(data, rf.marker) {
				r.Warning = "not managed by Kanbanzai (no marker found)"
				results = append(results, r)
				continue
			}
		}

		r.Ok = true
		results = append(results, r)
	}

	// Check for ghost files in managed directories.
	ghostDirs := []string{
		filepath.Join(kbzDir, "skills"),
		filepath.Join(repoRoot, ".agents", "skills"),
		filepath.Join(kbzDir, "roles"),
	}

	for _, dir := range ghostDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".md") && !strings.HasSuffix(d.Name(), ".yaml") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if !containsMarker(data, skillManagedMarker) {
				r := CheckResult{Path: path, Warning: "ghost file (not in managed install)"}
				results = append(results, r)
			}
			return nil
		})
	}

	return results, nil
}

// ExitCode returns 0 if no required files are missing, 1 otherwise.
func ExitCode(results []CheckResult) int {
	for _, r := range results {
		if r.Missing {
			return 1
		}
	}
	return 0
}

// PrintResults writes human-readable results to the doctor's stdout.
func (d *Doctor) PrintResults(results []CheckResult) {
	hasErrors := false
	for _, r := range results {
		if r.Missing {
			hasErrors = true
			fmt.Fprintf(d.stdout, "ERROR: %s — %s\n", r.Path, r.Warning)
		} else if r.Warning != "" {
			fmt.Fprintf(d.stdout, "WARN:  %s — %s\n", r.Path, r.Warning)
		}
	}

	okCount := 0
	for _, r := range results {
		if r.Ok {
			okCount++
		}
	}

	if hasErrors {
		fmt.Fprintf(d.stdout, "\n%d/%d checks passed. Fix errors above.\n", okCount, len(results))
	} else {
		fmt.Fprintf(d.stdout, "\nAll %d checks passed.\n", len(results))
	}
}

// containsMarker checks if data contains the given marker string.
func containsMarker(data []byte, marker string) bool {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), marker) {
			return true
		}
	}
	return false
}

// extractMarkerVersion extracts the version number from a markdown managed marker.
// Expected format: "<!-- kanbanzai-managed: vN -->"
func extractMarkerVersion(line string) (int, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, managedMarkerPrefix) {
		return 0, false
	}
	inner := strings.TrimPrefix(line, managedMarkerPrefix)
	inner = strings.TrimSuffix(inner, " -->")
	inner = strings.TrimSpace(inner)
	v, err := strconv.Atoi(inner)
	if err != nil {
		return 0, false
	}
	return v, true
}
