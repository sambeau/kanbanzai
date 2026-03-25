package github

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   error
	}{
		// HTTPS format
		{
			name:      "HTTPS with .git suffix",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS without .git suffix",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "HTTPS with org name containing dash",
			url:       "https://github.com/my-org/my-repo.git",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
		},
		{
			name:      "HTTPS with underscores",
			url:       "https://github.com/my_org/my_repo.git",
			wantOwner: "my_org",
			wantRepo:  "my_repo",
		},

		// SSH format (git@github.com:owner/repo)
		{
			name:      "SSH with .git suffix",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH without .git suffix",
			url:       "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH with org name containing dash",
			url:       "git@github.com:my-org/my-repo.git",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
		},

		// SSH with ssh:// prefix
		{
			name:      "SSH prefix format with .git",
			url:       "ssh://git@github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "SSH prefix format without .git",
			url:       "ssh://git@github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},

		// Real-world examples
		{
			name:      "Real HTTPS example",
			url:       "https://github.com/samphillips/kanbanzai.git",
			wantOwner: "samphillips",
			wantRepo:  "kanbanzai",
		},
		{
			name:      "Real SSH example",
			url:       "git@github.com:samphillips/kanbanzai.git",
			wantOwner: "samphillips",
			wantRepo:  "kanbanzai",
		},

		// Error cases
		{
			name:    "Empty URL",
			url:     "",
			wantErr: ErrRemoteNotFound,
		},
		{
			name:    "Whitespace only",
			url:     "   ",
			wantErr: ErrRemoteNotFound,
		},
		{
			name:    "GitLab URL",
			url:     "https://gitlab.com/owner/repo.git",
			wantErr: ErrNotGitHubRemote,
		},
		{
			name:    "Bitbucket URL",
			url:     "git@bitbucket.org:owner/repo.git",
			wantErr: ErrNotGitHubRemote,
		},
		{
			name:    "Local file path",
			url:     "/path/to/repo",
			wantErr: ErrNotGitHubRemote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRemoteURL(tt.url)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ParseRemoteURL() error = nil, wantErr = %v", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("ParseRemoteURL() error = %v, wantErr = %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRemoteURL() unexpected error = %v", err)
				return
			}

			if got.Owner != tt.wantOwner {
				t.Errorf("ParseRemoteURL() Owner = %q, want %q", got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("ParseRemoteURL() Repo = %q, want %q", got.Repo, tt.wantRepo)
			}
		})
	}
}

func TestRepoInfo_String(t *testing.T) {
	tests := []struct {
		name string
		info RepoInfo
		want string
	}{
		{
			name: "Normal repo",
			info: RepoInfo{Owner: "owner", Repo: "repo"},
			want: "owner/repo",
		},
		{
			name: "Org with dash",
			info: RepoInfo{Owner: "my-org", Repo: "my-repo"},
			want: "my-org/my-repo",
		},
		{
			name: "Empty",
			info: RepoInfo{},
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.String(); got != tt.want {
				t.Errorf("RepoInfo.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepoInfo_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		info RepoInfo
		want bool
	}{
		{
			name: "Both empty",
			info: RepoInfo{},
			want: true,
		},
		{
			name: "Only owner set",
			info: RepoInfo{Owner: "owner"},
			want: false,
		},
		{
			name: "Only repo set",
			info: RepoInfo{Repo: "repo"},
			want: false,
		},
		{
			name: "Both set",
			info: RepoInfo{Owner: "owner", Repo: "repo"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.IsEmpty(); got != tt.want {
				t.Errorf("RepoInfo.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
