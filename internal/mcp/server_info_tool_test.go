package mcp

import (
	"testing"
	"time"

	"kanbanzai/internal/install"
)

func TestDeriveGitSHAShort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sha  string
		want string
	}{
		{name: "full_sha", sha: "abc1234def5678", want: "abc1234"},
		{name: "exactly_7", sha: "abc1234", want: "abc1234"},
		{name: "shorter_than_7", sha: "abc12", want: "abc12"},
		{name: "unknown", sha: "unknown", want: "unknown"},
		{name: "empty", sha: "", want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deriveGitSHAShort(tt.sha)
			if got != tt.want {
				t.Errorf("deriveGitSHAShort(%q) = %q, want %q", tt.sha, got, tt.want)
			}
		})
	}
}

func TestDeriveInSync(t *testing.T) {
	t.Parallel()

	makeRecord := func(sha string) *install.InstallRecord {
		return &install.InstallRecord{
			GitSHA:      sha,
			InstalledAt: time.Now(),
			InstalledBy: "test",
			BinaryPath:  "/usr/local/bin/kanbanzai",
		}
	}

	tests := []struct {
		name   string
		gitSHA string
		rec    *install.InstallRecord
		want   any // true, false, or nil
	}{
		{
			name:   "match_returns_true",
			gitSHA: "abc1234def5678",
			rec:    makeRecord("abc1234def5678"),
			want:   true,
		},
		{
			name:   "mismatch_returns_false",
			gitSHA: "abc1234def5678",
			rec:    makeRecord("zzz9999fff0000"),
			want:   false,
		},
		{
			name:   "unknown_git_sha_returns_nil",
			gitSHA: "unknown",
			rec:    makeRecord("abc1234def5678"),
			want:   nil,
		},
		{
			name:   "empty_git_sha_returns_nil",
			gitSHA: "",
			rec:    makeRecord("abc1234def5678"),
			want:   nil,
		},
		{
			name:   "no_record_returns_nil",
			gitSHA: "abc1234def5678",
			rec:    nil,
			want:   nil,
		},
		{
			name:   "record_with_empty_sha_returns_nil",
			gitSHA: "abc1234def5678",
			rec:    makeRecord(""),
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deriveInSync(tt.gitSHA, tt.rec)
			if got != tt.want {
				t.Errorf("deriveInSync(%q, rec) = %v (%T), want %v (%T)",
					tt.gitSHA, got, got, tt.want, tt.want)
			}
		})
	}
}
