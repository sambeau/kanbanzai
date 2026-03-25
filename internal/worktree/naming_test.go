package worktree

import "testing"

func TestGenerateBranchName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entityID string
		slug     string
		want     string
	}{
		{
			name:     "feature with slug",
			entityID: "FEAT-01JX987654321",
			slug:     "user-profiles",
			want:     "feature/FEAT-01JX987654321-user-profiles",
		},
		{
			name:     "feature without slug",
			entityID: "FEAT-01JX987654321",
			slug:     "",
			want:     "feature/FEAT-01JX987654321",
		},
		{
			name:     "bug with slug",
			entityID: "BUG-01JX123456789",
			slug:     "fix-crash",
			want:     "bugfix/BUG-01JX123456789-fix-crash",
		},
		{
			name:     "bug without slug",
			entityID: "BUG-01JX123456789",
			slug:     "",
			want:     "bugfix/BUG-01JX123456789",
		},
		{
			name:     "lowercase bug prefix",
			entityID: "bug-01JX123456789",
			slug:     "minor-fix",
			want:     "bugfix/bug-01JX123456789-minor-fix",
		},
		{
			name:     "slug with spaces becomes hyphens",
			entityID: "FEAT-01JX987654321",
			slug:     "add user profiles",
			want:     "feature/FEAT-01JX987654321-add-user-profiles",
		},
		{
			name:     "slug with special characters",
			entityID: "FEAT-01JX987654321",
			slug:     "Add User's Profile!",
			want:     "feature/FEAT-01JX987654321-add-user-s-profile",
		},
		{
			name:     "slug with underscores",
			entityID: "FEAT-01JX987654321",
			slug:     "user_profile_api",
			want:     "feature/FEAT-01JX987654321-user-profile-api",
		},
		{
			name:     "unknown entity type defaults to feature",
			entityID: "TASK-01JX987654321",
			slug:     "implement",
			want:     "feature/TASK-01JX987654321-implement",
		},
		{
			name:     "slug with mixed case",
			entityID: "FEAT-01JX987654321",
			slug:     "UserProfileAPI",
			want:     "feature/FEAT-01JX987654321-userprofileapi",
		},
		{
			name:     "slug with leading/trailing spaces",
			entityID: "FEAT-01JX987654321",
			slug:     "  user profiles  ",
			want:     "feature/FEAT-01JX987654321-user-profiles",
		},
		{
			name:     "slug with consecutive special chars",
			entityID: "FEAT-01JX987654321",
			slug:     "user---profile",
			want:     "feature/FEAT-01JX987654321-user-profile",
		},
		{
			name:     "slug with numbers",
			entityID: "FEAT-01JX987654321",
			slug:     "oauth2-integration",
			want:     "feature/FEAT-01JX987654321-oauth2-integration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GenerateBranchName(tt.entityID, tt.slug)
			if got != tt.want {
				t.Errorf("GenerateBranchName(%q, %q) = %q, want %q",
					tt.entityID, tt.slug, got, tt.want)
			}
		})
	}
}

func TestGenerateWorktreePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entityID string
		slug     string
		want     string
	}{
		{
			name:     "feature with slug",
			entityID: "FEAT-01JX987654321",
			slug:     "user-profiles",
			want:     ".worktrees/FEAT-01JX987654321-user-profiles",
		},
		{
			name:     "feature without slug",
			entityID: "FEAT-01JX987654321",
			slug:     "",
			want:     ".worktrees/FEAT-01JX987654321",
		},
		{
			name:     "bug with slug",
			entityID: "BUG-01JX123456789",
			slug:     "fix-crash",
			want:     ".worktrees/BUG-01JX123456789-fix-crash",
		},
		{
			name:     "bug without slug",
			entityID: "BUG-01JX123456789",
			slug:     "",
			want:     ".worktrees/BUG-01JX123456789",
		},
		{
			name:     "slug normalization",
			entityID: "FEAT-01JX987654321",
			slug:     "Add User Profiles!",
			want:     ".worktrees/FEAT-01JX987654321-add-user-profiles",
		},
		{
			name:     "slug with underscores",
			entityID: "FEAT-01JX987654321",
			slug:     "user_profile_api",
			want:     ".worktrees/FEAT-01JX987654321-user-profile-api",
		},
		{
			name:     "slug with numbers",
			entityID: "FEAT-01JX987654321",
			slug:     "oauth2-integration",
			want:     ".worktrees/FEAT-01JX987654321-oauth2-integration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GenerateWorktreePath(tt.entityID, tt.slug)
			if got != tt.want {
				t.Errorf("GenerateWorktreePath(%q, %q) = %q, want %q",
					tt.entityID, tt.slug, got, tt.want)
			}
		})
	}
}

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "user-profiles",
			want:  "user-profiles",
		},
		{
			name:  "uppercase",
			input: "USER-PROFILES",
			want:  "user-profiles",
		},
		{
			name:  "mixed case",
			input: "UserProfiles",
			want:  "userprofiles",
		},
		{
			name:  "spaces",
			input: "user profiles",
			want:  "user-profiles",
		},
		{
			name:  "underscores",
			input: "user_profiles",
			want:  "user-profiles",
		},
		{
			name:  "special characters",
			input: "user's profile!",
			want:  "user-s-profile",
		},
		{
			name:  "leading hyphen from special char",
			input: "!important",
			want:  "important",
		},
		{
			name:  "trailing hyphen from special char",
			input: "important!",
			want:  "important",
		},
		{
			name:  "multiple consecutive hyphens",
			input: "user---profile",
			want:  "user-profile",
		},
		{
			name:  "leading and trailing spaces",
			input: "  user profiles  ",
			want:  "user-profiles",
		},
		{
			name:  "numbers preserved",
			input: "oauth2",
			want:  "oauth2",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "!@#$%",
			want:  "",
		},
		{
			name:  "complex mix",
			input: "  Add User's OAuth2 Profile! (v2)  ",
			want:  "add-user-s-oauth2-profile-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeSlug(tt.input)
			if got != tt.want {
				t.Errorf("normalizeSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBranchPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entityID string
		want     string
	}{
		{"feature entity", "FEAT-01JX987654321", "feature"},
		{"bug entity", "BUG-01JX123456789", "bugfix"},
		{"lowercase feature", "feat-01JX987654321", "feature"},
		{"lowercase bug", "bug-01JX123456789", "bugfix"},
		{"task entity", "TASK-01JX987654321", "feature"},
		{"unknown entity", "UNKNOWN-01JX987654321", "feature"},
		{"empty entity", "", "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := branchPrefix(tt.entityID)
			if got != tt.want {
				t.Errorf("branchPrefix(%q) = %q, want %q", tt.entityID, got, tt.want)
			}
		})
	}
}

func TestGenerateBranchName_Consistency(t *testing.T) {
	t.Parallel()

	// Calling with same inputs should always produce same output
	entityID := "FEAT-01JX987654321"
	slug := "user profiles"

	result1 := GenerateBranchName(entityID, slug)
	result2 := GenerateBranchName(entityID, slug)

	if result1 != result2 {
		t.Errorf("GenerateBranchName not deterministic: %q != %q", result1, result2)
	}
}

func TestGenerateWorktreePath_Consistency(t *testing.T) {
	t.Parallel()

	// Calling with same inputs should always produce same output
	entityID := "FEAT-01JX987654321"
	slug := "user profiles"

	result1 := GenerateWorktreePath(entityID, slug)
	result2 := GenerateWorktreePath(entityID, slug)

	if result1 != result2 {
		t.Errorf("GenerateWorktreePath not deterministic: %q != %q", result1, result2)
	}
}
