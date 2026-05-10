package kbzinit

import "testing"

// TestCompareManaged is a table-driven test covering all six decision rules
// from REQ-004 for both IntCounter and Semver VersionKinds (AC-002).
func TestCompareManaged(t *testing.T) {
	// intSpec is a typical workflow-skill MarkerSpec (IntCounter).
	intSpec := MarkerSpec{
		Comment:      "# kanbanzai-version:",
		VersionKind:  IntCounter,
		CurrentValue: "5",
	}

	// semSpec is a typical stage-bindings MarkerSpec (Semver).
	semSpec := MarkerSpec{
		Comment:      "# kanbanzai-version:",
		VersionKind:  Semver,
		CurrentValue: "v2.0.0",
	}

	// htmlSpec mimics AGENTS.md where the version is embedded in an HTML
	// comment marker on line 1.
	htmlSpec := MarkerSpec{
		Comment:      "<!-- kanbanzai-managed: v",
		VersionKind:  IntCounter,
		CurrentValue: "5",
	}

	cases := []struct {
		name     string
		existing []byte
		spec     MarkerSpec
		want     Decision
	}{
		// ── Rule 1: absent file → Create ──────────────────────────────────────
		{
			name:     "nil bytes returns Create",
			existing: nil,
			spec:     intSpec,
			want:     Create,
		},
		{
			name:     "empty bytes returns Create",
			existing: []byte{},
			spec:     intSpec,
			want:     Create,
		},

		// ── Rule 2: present, no marker → WarnSkip ────────────────────────────
		{
			name:     "IntCounter: no marker line returns WarnSkip",
			existing: []byte("# some user content\nno managed marker here\n"),
			spec:     intSpec,
			want:     WarnSkip,
		},
		{
			name:     "Semver: no marker line returns WarnSkip",
			existing: []byte("# some user content\nno managed marker here\n"),
			spec:     semSpec,
			want:     WarnSkip,
		},

		// ── Rule 3: present, marker found, version unparseable → WarnSkip ────
		{
			name:     "IntCounter: garbage version returns WarnSkip",
			existing: []byte("# kanbanzai-version: not-a-number\n"),
			spec:     intSpec,
			want:     WarnSkip,
		},
		{
			name:     "Semver: garbage version returns WarnSkip",
			existing: []byte("# kanbanzai-version: not-a-semver\n"),
			spec:     semSpec,
			want:     WarnSkip,
		},
		{
			name:     "Semver: empty version string returns WarnSkip",
			existing: []byte("# kanbanzai-version: \n"),
			spec:     semSpec,
			want:     WarnSkip,
		},

		// ── Rule 4: present, marker found, version older → Overwrite ─────────
		{
			name:     "IntCounter: older version (3 < 5) returns Overwrite",
			existing: []byte("# kanbanzai-version: 3\n"),
			spec:     intSpec,
			want:     Overwrite,
		},
		{
			name:     "IntCounter: version 1 returns Overwrite",
			existing: []byte("preamble\n# kanbanzai-version: 1\ntrailing\n"),
			spec:     intSpec,
			want:     Overwrite,
		},
		{
			name:     "Semver: older minor version returns Overwrite",
			existing: []byte("# kanbanzai-version: v1.9.9\n"),
			spec:     semSpec,
			want:     Overwrite,
		},
		{
			name:     "Semver: older patch version returns Overwrite",
			existing: []byte("# kanbanzai-version: v1.99.99\n"),
			spec:     semSpec,
			want:     Overwrite,
		},
		{
			name:     "HTML IntCounter: older version returns Overwrite",
			existing: []byte("<!-- kanbanzai-managed: v3 -->\nsome content\n"),
			spec:     htmlSpec,
			want:     Overwrite,
		},

		// ── Rule 5: present, marker found, version equal → NoOp ──────────────
		{
			name:     "IntCounter: equal version (5 == 5) returns NoOp",
			existing: []byte("# kanbanzai-version: 5\n"),
			spec:     intSpec,
			want:     NoOp,
		},
		{
			name:     "Semver: equal version returns NoOp",
			existing: []byte("# kanbanzai-version: v2.0.0\n"),
			spec:     semSpec,
			want:     NoOp,
		},
		{
			name:     "HTML IntCounter: equal version returns NoOp",
			existing: []byte("<!-- kanbanzai-managed: v5 -->\nsome content\n"),
			spec:     htmlSpec,
			want:     NoOp,
		},

		// ── Rule 6: present, marker found, version newer → NoOp ──────────────
		{
			name:     "IntCounter: newer version (999 > 5) returns NoOp",
			existing: []byte("# kanbanzai-version: 999\n"),
			spec:     intSpec,
			want:     NoOp,
		},
		{
			name:     "IntCounter: version 6 returns NoOp",
			existing: []byte("# kanbanzai-version: 6\n"),
			spec:     intSpec,
			want:     NoOp,
		},
		{
			name:     "Semver: newer major version returns NoOp",
			existing: []byte("# kanbanzai-version: v9.9.9\n"),
			spec:     semSpec,
			want:     NoOp,
		},
		{
			name:     "Semver: newer minor version returns NoOp",
			existing: []byte("# kanbanzai-version: v2.1.0\n"),
			spec:     semSpec,
			want:     NoOp,
		},
		{
			name:     "HTML IntCounter: newer version returns NoOp",
			existing: []byte("<!-- kanbanzai-managed: v999 -->\nsome content\n"),
			spec:     htmlSpec,
			want:     NoOp,
		},

		// ── Semver without leading v ──────────────────────────────────────────
		{
			name:     "Semver: older version without v prefix returns Overwrite",
			existing: []byte("# kanbanzai-version: 1.0.0\n"),
			spec:     semSpec,
			want:     Overwrite,
		},
		{
			name:     "Semver: equal version without v prefix returns NoOp",
			existing: []byte("# kanbanzai-version: 2.0.0\n"),
			spec:     semSpec,
			want:     NoOp,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareManaged(tc.existing, tc.spec)
			if got != tc.want {
				t.Errorf("compareManaged(%q, spec{Comment:%q, Kind:%q, Current:%q}) = %v, want %v",
					tc.existing, tc.spec.Comment, tc.spec.VersionKind, tc.spec.CurrentValue,
					got, tc.want)
			}
		})
	}
}
