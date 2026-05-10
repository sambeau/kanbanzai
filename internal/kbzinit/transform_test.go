package kbzinit

import (
	"strings"
	"testing"
)

// TestTransformSkillContent_InjectsMarkerWhenAbsent verifies AC-002: given an
// embedded skill source with no managed-marker or version line, the output
// contains both required frontmatter lines.
func TestTransformSkillContent_InjectsMarkerWhenAbsent(t *testing.T) {
	src := []byte("# My Skill\n\nSome content here.\n")

	got, err := transformSkillContent(src, "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := string(got)

	if !strings.Contains(out, "# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills") {
		t.Errorf("output missing managed-marker line:\n%s", out)
	}
	if !strings.Contains(out, "# kanbanzai-version: v1.2.3") {
		t.Errorf("output missing version line:\n%s", out)
	}
	// Body content must be preserved.
	if !strings.Contains(out, "# My Skill") {
		t.Errorf("output missing original content:\n%s", out)
	}
}

// TestTransformSkillContent_InjectedMarkersAppearsAtTop verifies that when
// markers are injected they appear before any other content.
func TestTransformSkillContent_InjectedMarkersAppearsAtTop(t *testing.T) {
	src := []byte("# My Skill\n\nSome content.\n")

	got, err := transformSkillContent(src, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(string(got), "\n")
	if len(lines) < 2 {
		t.Fatalf("output too short: %q", string(got))
	}
	if !strings.HasPrefix(lines[0], "# kanbanzai-managed:") {
		t.Errorf("first line should be managed-marker, got: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "# kanbanzai-version:") {
		t.Errorf("second line should be version-marker, got: %q", lines[1])
	}
}

// TestTransformSkillContent_ReplacesExistingMarkers verifies that when the
// managed-marker and version lines are already present they are replaced with
// the canonical text and current version.
func TestTransformSkillContent_ReplacesExistingMarkers(t *testing.T) {
	src := []byte("# kanbanzai-managed: true\n# kanbanzai-version: dev\n\n# Skill\n\nContent.\n")

	got, err := transformSkillContent(src, "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := string(got)

	if !strings.Contains(out, "# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills") {
		t.Errorf("canonical managed-marker text not found:\n%s", out)
	}
	if !strings.Contains(out, "# kanbanzai-version: v2.0.0") {
		t.Errorf("updated version not found:\n%s", out)
	}
	// Old short form must not appear.
	if strings.Contains(out, "# kanbanzai-managed: true") {
		t.Errorf("old short marker still present:\n%s", out)
	}
}

// TestTransformSkillContent_IdempotentWhenUpToDate verifies the invariant from
// the interface contract: when both markers are already present with the
// canonical text and the current version, the output equals the input.
func TestTransformSkillContent_IdempotentWhenUpToDate(t *testing.T) {
	canonical := "# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills\n" +
		"# kanbanzai-version: v1.0.0\n\n# Skill\n\nContent.\n"
	src := []byte(canonical)

	got, err := transformSkillContent(src, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != canonical {
		t.Errorf("output differs from input for up-to-date content.\ngot:\n%s\nwant:\n%s", got, canonical)
	}
}

// TestTransformSkillContent_NeverProducesMarkerlessOutput is a property check:
// for any input, the output must always contain the managed-marker line.
func TestTransformSkillContent_NeverProducesMarkerlessOutput(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"empty", ""},
		{"whitespace only", "\n\n\n"},
		{"no markers", "# Title\n\nBody.\n"},
		{"managed only", "# kanbanzai-managed: true\n# Title\n"},
		{"version only", "# kanbanzai-version: dev\n# Title\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := transformSkillContent([]byte(tc.src), "v9.9.9")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			out := string(got)
			if !strings.Contains(out, "# kanbanzai-managed:") {
				t.Errorf("output is marker-less for input %q:\n%s", tc.src, out)
			}
			if !strings.Contains(out, "# kanbanzai-version:") {
				t.Errorf("output has no version line for input %q:\n%s", tc.src, out)
			}
		})
	}
}
