package kbzinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestManifestIsCanonical asserts two invariants (AC-001 / REQ-002):
//
//  1. No artifact Name appears more than once inside the Manifest slice itself.
//  2. No artifact Name appears as a quoted string literal in any non-test
//     source file outside of manifest.go.
//
// Invariant 2 will fail until T4 removes the legacy skillNames /
// taskSkillNames slices from skills.go and task_skills.go. That is
// intentional — the test encodes the desired end state.
func TestManifestIsCanonical(t *testing.T) {
	t.Run("no_manifest_duplicates", testManifestNoDuplicates)
	t.Run("no_external_duplicates", testManifestNoExternalDuplicates)
}

// testManifestNoDuplicates verifies every Name in the Manifest slice is unique.
func testManifestNoDuplicates(t *testing.T) {
	t.Helper()
	seen := make(map[string]int)
	for _, a := range Manifest {
		if a.Name == "" {
			t.Errorf("Manifest entry has empty Name (Kind=%s, EmbedPath=%s)", a.Kind, a.EmbedPath)
			continue
		}
		seen[a.Name]++
	}
	for name, count := range seen {
		if count > 1 {
			t.Errorf("artifact %q appears %d times in Manifest (want 1)", name, count)
		}
	}
}

// testManifestNoExternalDuplicates scans all non-test .go files in the
// package directory (excluding manifest.go itself) and asserts that no
// artifact Name appears as a quoted string literal.
//
// After T4 removes skillNames / taskSkillNames, this test will pass.
func testManifestNoExternalDuplicates(t *testing.T) {
	t.Helper()

	// Collect the source of every non-test, non-manifest .go file in the
	// package directory. Tests run with CWD set to the package directory.
	sources, err := loadNonTestSources(t, ".")
	if err != nil {
		t.Fatalf("loading package sources: %v", err)
	}

	for _, a := range Manifest {
		if a.Name == "" {
			continue
		}
		quoted := `"` + a.Name + `"`
		for file, content := range sources {
			if strings.Contains(content, quoted) {
				t.Errorf("artifact name %s found as string literal in %s (should only appear in manifest.go)",
					quoted, file)
			}
		}
	}
}

// loadNonTestSources reads all .go files in dir that are not _test.go files
// and not manifest.go, returning a map[filename]content.
func loadNonTestSources(t *testing.T, dir string) (map[string]string, error) {
	t.Helper()
	result := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if name == "manifest.go" {
			continue // the Manifest's own declaration doesn't count
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		result[name] = string(data)
	}
	return result, nil
}
