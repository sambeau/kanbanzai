package knowledge

import (
	"testing"
)

func TestFindDuplicateCandidates(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		summary   string
		existing  []ExistingEntity
		threshold float64
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "empty existing list",
			title:     "Add user login",
			summary:   "Implement login flow",
			existing:  nil,
			threshold: 0.5,
			wantCount: 0,
		},
		{
			name:    "no matches above threshold",
			title:   "Add user login",
			summary: "",
			existing: []ExistingEntity{
				{ID: "FEAT-001", Type: "feature", Title: "Delete old records", Summary: "Remove stale data"},
			},
			threshold: 0.5,
			wantCount: 0,
		},
		{
			name:    "exact title match",
			title:   "Add user login",
			summary: "",
			existing: []ExistingEntity{
				{ID: "FEAT-001", Type: "feature", Title: "Add user login", Summary: ""},
			},
			threshold: 0.5,
			wantCount: 1,
			wantIDs:   []string{"FEAT-001"},
		},
		{
			name:    "similar title above threshold",
			title:   "Add user login page",
			summary: "",
			existing: []ExistingEntity{
				// input words: {add, user, login, page} = 4
				// existing words: {add, user, login, form} = 4
				// intersection: 3, union: 5, jaccard = 0.6
				{ID: "FEAT-001", Type: "feature", Title: "Add user login form", Summary: ""},
			},
			threshold: 0.5,
			wantCount: 1,
			wantIDs:   []string{"FEAT-001"},
		},
		{
			name:    "similar summary above threshold",
			title:   "payment",
			summary: "credit card processing",
			existing: []ExistingEntity{
				// input words: {payment, credit, card, processing} = 4
				// existing words: {billing, credit, card, payment} = 4
				// intersection: {payment, credit, card} = 3, union: 5, jaccard = 0.6
				{ID: "FEAT-002", Type: "feature", Title: "billing", Summary: "credit card payment"},
			},
			threshold: 0.5,
			wantCount: 1,
			wantIDs:   []string{"FEAT-002"},
		},
		{
			name:    "below threshold",
			title:   "Add user login",
			summary: "",
			existing: []ExistingEntity{
				// input words: {add, user, login} = 3
				// existing words: {delete, old, records} = 3
				// intersection: 0, union: 6, jaccard = 0.0
				{ID: "FEAT-003", Type: "feature", Title: "Delete old records", Summary: ""},
			},
			threshold: 0.5,
			wantCount: 0,
		},
		{
			name:    "multiple matches",
			title:   "user login",
			summary: "",
			existing: []ExistingEntity{
				// input words: {user, login} = 2
				// each existing has intersection 2, union 3, jaccard ≈ 0.667
				{ID: "FEAT-010", Type: "feature", Title: "user login page", Summary: ""},
				{ID: "FEAT-011", Type: "feature", Title: "user login api", Summary: ""},
			},
			threshold: 0.5,
			wantCount: 2,
			wantIDs:   []string{"FEAT-010", "FEAT-011"},
		},
		{
			name:    "threshold boundary exactly 0.5 included",
			title:   "add user login",
			summary: "",
			existing: []ExistingEntity{
				// input words: {add, user, login} = 3
				// existing words: {add, user, auth} = 3
				// intersection: 2, union: 4, jaccard = 0.5 (exactly at threshold)
				{ID: "FEAT-020", Type: "feature", Title: "add user auth", Summary: ""},
			},
			threshold: 0.5,
			wantCount: 1,
			wantIDs:   []string{"FEAT-020"},
		},
		{
			name:    "threshold boundary just below excluded",
			title:   "add user login deploy",
			summary: "",
			existing: []ExistingEntity{
				// input words: {add, user, login, deploy} = 4
				// existing words: {add, user, auth, config} = 4
				// intersection: 2, union: 6, jaccard ≈ 0.333
				{ID: "FEAT-021", Type: "feature", Title: "add user auth config", Summary: ""},
			},
			threshold: 0.5,
			wantCount: 0,
		},
		{
			name:    "empty title and summary input",
			title:   "",
			summary: "",
			existing: []ExistingEntity{
				{ID: "FEAT-030", Type: "feature", Title: "Some feature", Summary: "With details"},
			},
			threshold: 0.5,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindDuplicateCandidates(tt.title, tt.summary, tt.existing, tt.threshold)

			if len(got) != tt.wantCount {
				t.Fatalf("got %d candidates, want %d", len(got), tt.wantCount)
			}

			if tt.wantIDs != nil {
				for i, want := range tt.wantIDs {
					if got[i].EntityID != want {
						t.Errorf("candidate[%d].EntityID = %q, want %q", i, got[i].EntityID, want)
					}
				}
			}

			// Verify all returned candidates meet threshold.
			for i, c := range got {
				if c.Similarity < tt.threshold {
					t.Errorf("candidate[%d] similarity %f is below threshold %f", i, c.Similarity, tt.threshold)
				}
			}
		})
	}
}

func TestFindDuplicateCandidates_ExactMatchSimilarity(t *testing.T) {
	existing := []ExistingEntity{
		{ID: "FEAT-001", Type: "feature", Title: "Add user login", Summary: ""},
	}

	got := FindDuplicateCandidates("Add user login", "", existing, 0.5)
	if len(got) != 1 {
		t.Fatalf("got %d candidates, want 1", len(got))
	}
	if got[0].Similarity != 1.0 {
		t.Errorf("exact match similarity = %f, want 1.0", got[0].Similarity)
	}
	if got[0].EntityType != "feature" {
		t.Errorf("EntityType = %q, want %q", got[0].EntityType, "feature")
	}
	if got[0].Title != "Add user login" {
		t.Errorf("Title = %q, want %q", got[0].Title, "Add user login")
	}
}
