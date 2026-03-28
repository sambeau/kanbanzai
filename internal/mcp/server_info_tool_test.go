package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

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

func TestHandleServerInfo_NoRecord(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	result, err := handleServerInfo(root)
	if err != nil {
		t.Fatalf("handleServerInfo: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, tc.Text)
	}

	// All 9 top-level fields must be present.
	for _, field := range []string{
		"version", "git_sha", "git_sha_short", "build_time",
		"go_version", "binary_path", "dirty", "install_record", "in_sync",
	} {
		if _, present := resp[field]; !present {
			t.Errorf("missing field %q", field)
		}
	}

	// No install record → install_record and in_sync should both be null.
	if resp["install_record"] != nil {
		t.Errorf("install_record: got %v, want nil", resp["install_record"])
	}
	if resp["in_sync"] != nil {
		t.Errorf("in_sync: got %v, want nil", resp["in_sync"])
	}
}

func TestHandleServerInfo_WithRecord(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	const recSHA = "aabbccdd1234567890abcdef1234567890abcdef"
	if err := install.WriteRecord(root, recSHA, "/usr/local/bin/kanbanzai", "test"); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	result, err := handleServerInfo(root)
	if err != nil {
		t.Fatalf("handleServerInfo: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, tc.Text)
	}

	// install_record should be populated with all four sub-fields.
	rec, ok := resp["install_record"].(map[string]any)
	if !ok {
		t.Fatalf("install_record: got %T (%v), want map", resp["install_record"], resp["install_record"])
	}
	if rec["git_sha"] != recSHA {
		t.Errorf("install_record.git_sha = %q, want %q", rec["git_sha"], recSHA)
	}
	if rec["installed_by"] != "test" {
		t.Errorf("install_record.installed_by = %q, want %q", rec["installed_by"], "test")
	}
	for _, sub := range []string{"installed_at", "binary_path"} {
		if _, present := rec[sub]; !present {
			t.Errorf("install_record.%s missing", sub)
		}
	}

	// In test builds buildinfo.GitSHA is "unknown", so in_sync must be nil.
	if resp["in_sync"] != nil {
		// If the binary was built with real ldflags the SHA will be known;
		// in that case in_sync is bool (false, since recSHA won't match).
		if _, isBool := resp["in_sync"].(bool); !isBool {
			t.Errorf("in_sync: got %T, want bool or nil", resp["in_sync"])
		}
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
