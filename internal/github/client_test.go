package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.token != "test-token" {
		t.Errorf("client.token = %q, want %q", client.token, "test-token")
	}
	if client.baseURL != DefaultBaseURL {
		t.Errorf("client.baseURL = %q, want %q", client.baseURL, DefaultBaseURL)
	}
	if client.httpClient == nil {
		t.Error("client.httpClient is nil")
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	httpClient := &http.Client{}
	client := NewClientWithHTTPClient("test-token", httpClient)

	if client == nil {
		t.Fatal("NewClientWithHTTPClient() returned nil")
	}
	if client.httpClient != httpClient {
		t.Error("client.httpClient was not set to provided client")
	}
}

func TestClient_SetBaseURL(t *testing.T) {
	client := NewClient("token")
	client.SetBaseURL("https://custom.api.com/")

	// Should strip trailing slash
	if client.baseURL != "https://custom.api.com" {
		t.Errorf("client.baseURL = %q, want %q", client.baseURL, "https://custom.api.com")
	}
}

func TestClient_CreatePR(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		statusCode int
		response   any
		wantErr    error
		wantPR     *PR
	}{
		{
			name:    "No token",
			token:   "",
			wantErr: ErrNoToken,
		},
		{
			name:       "Successful creation",
			token:      "valid-token",
			statusCode: http.StatusCreated,
			response: apiPR{
				Number:  42,
				HTMLURL: "https://github.com/owner/repo/pull/42",
				Title:   "Test PR",
				Body:    "Test body",
				State:   "open",
				Draft:   false,
				Head: struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				}{Ref: "feature-branch", SHA: "abc123"},
				Base: struct {
					Ref string `json:"ref"`
				}{Ref: "main"},
			},
			wantPR: &PR{
				Number:     42,
				URL:        "https://github.com/owner/repo/pull/42",
				Title:      "Test PR",
				Body:       "Test body",
				State:      "open",
				Draft:      false,
				HeadBranch: "feature-branch",
				BaseBranch: "main",
			},
		},
		{
			name:       "Unauthorized",
			token:      "bad-token",
			statusCode: http.StatusUnauthorized,
			response:   map[string]string{"message": "Bad credentials"},
			wantErr:    ErrUnauthorized,
		},
		{
			name:       "Rate limited",
			token:      "valid-token",
			statusCode: http.StatusTooManyRequests,
			response:   map[string]string{"message": "Rate limit exceeded"},
			wantErr:    ErrRateLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request
				if r.Method != http.MethodPost {
					t.Errorf("Method = %q, want POST", r.Method)
				}
				if r.URL.Path != "/repos/owner/repo/pulls" {
					t.Errorf("Path = %q, want /repos/owner/repo/pulls", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer "+tt.token {
					t.Errorf("Authorization = %q, want %q", got, "Bearer "+tt.token)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClient(tt.token)
			client.SetBaseURL(server.URL)

			repo := RepoInfo{Owner: "owner", Repo: "repo"}
			pr, err := client.CreatePR(repo, "feature-branch", "main", "Test PR", "Test body", false)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CreatePR() error = %v, wantErr = %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("CreatePR() unexpected error = %v", err)
				return
			}

			if pr.Number != tt.wantPR.Number {
				t.Errorf("PR.Number = %d, want %d", pr.Number, tt.wantPR.Number)
			}
			if pr.URL != tt.wantPR.URL {
				t.Errorf("PR.URL = %q, want %q", pr.URL, tt.wantPR.URL)
			}
			if pr.Title != tt.wantPR.Title {
				t.Errorf("PR.Title = %q, want %q", pr.Title, tt.wantPR.Title)
			}
		})
	}
}

func TestClient_GetPR(t *testing.T) {
	// Set up a server that returns PR details, status, and reviews
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/pulls/42":
			mergeable := true
			json.NewEncoder(w).Encode(apiPR{
				Number:  42,
				HTMLURL: "https://github.com/owner/repo/pull/42",
				Title:   "Test PR",
				State:   "open",
				Head: struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				}{Ref: "feature", SHA: "abc123"},
				Base: struct {
					Ref string `json:"ref"`
				}{Ref: "main"},
				Mergeable: &mergeable,
			})
		case "/repos/owner/repo/commits/abc123/status":
			json.NewEncoder(w).Encode(apiCombinedStatus{
				State: "success",
			})
		case "/repos/owner/repo/pulls/42/reviews":
			json.NewEncoder(w).Encode([]apiReview{
				{User: struct {
					Login string `json:"login"`
				}{Login: "reviewer1"}, State: "APPROVED"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	pr, err := client.GetPR(repo, 42)

	if err != nil {
		t.Fatalf("GetPR() error = %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("PR.Number = %d, want 42", pr.Number)
	}
	if pr.CIStatus != "passed" {
		t.Errorf("PR.CIStatus = %q, want %q", pr.CIStatus, "passed")
	}
	if pr.ReviewStatus != "approved" {
		t.Errorf("PR.ReviewStatus = %q, want %q", pr.ReviewStatus, "approved")
	}
	if !pr.Mergeable {
		t.Error("PR.Mergeable = false, want true")
	}
}

func TestClient_GetPR_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	_, err := client.GetPR(repo, 9999)

	if err != ErrPRNotFound {
		t.Errorf("GetPR() error = %v, want %v", err, ErrPRNotFound)
	}
}

func TestClient_GetPRByBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/owner/repo/pulls":
			// Return a list with one PR
			json.NewEncoder(w).Encode([]apiPR{
				{
					Number:  42,
					HTMLURL: "https://github.com/owner/repo/pull/42",
					Title:   "Feature PR",
					State:   "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{Ref: "feature-branch", SHA: "def456"},
					Base: struct {
						Ref string `json:"ref"`
					}{Ref: "main"},
				},
			})
		case r.URL.Path == "/repos/owner/repo/commits/def456/status":
			json.NewEncoder(w).Encode(apiCombinedStatus{State: "pending"})
		case r.URL.Path == "/repos/owner/repo/pulls/42/reviews":
			json.NewEncoder(w).Encode([]apiReview{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	pr, err := client.GetPRByBranch(repo, "feature-branch")

	if err != nil {
		t.Fatalf("GetPRByBranch() error = %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("PR.Number = %d, want 42", pr.Number)
	}
	if pr.HeadBranch != "feature-branch" {
		t.Errorf("PR.HeadBranch = %q, want %q", pr.HeadBranch, "feature-branch")
	}
	if pr.CIStatus != "pending" {
		t.Errorf("PR.CIStatus = %q, want %q", pr.CIStatus, "pending")
	}
}

func TestClient_GetPRByBranch_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty list
		json.NewEncoder(w).Encode([]apiPR{})
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	_, err := client.GetPRByBranch(repo, "nonexistent-branch")

	if err != ErrPRNotFound {
		t.Errorf("GetPRByBranch() error = %v, want %v", err, ErrPRNotFound)
	}
}

func TestClient_UpdatePR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/pulls/42" {
			t.Errorf("Path = %q, want /repos/owner/repo/pulls/42", r.URL.Path)
		}

		json.NewEncoder(w).Encode(apiPR{
			Number:  42,
			HTMLURL: "https://github.com/owner/repo/pull/42",
			Title:   "Updated Title",
			Body:    "Updated Body",
			State:   "open",
			Head: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{Ref: "feature", SHA: "abc123"},
			Base: struct {
				Ref string `json:"ref"`
			}{Ref: "main"},
		})
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	pr, err := client.UpdatePR(repo, 42, "Updated Title", "Updated Body")

	if err != nil {
		t.Fatalf("UpdatePR() error = %v", err)
	}

	if pr.Title != "Updated Title" {
		t.Errorf("PR.Title = %q, want %q", pr.Title, "Updated Title")
	}
	if pr.Body != "Updated Body" {
		t.Errorf("PR.Body = %q, want %q", pr.Body, "Updated Body")
	}
}

func TestClient_RateLimitViaForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	_, err := client.GetPR(repo, 1)

	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("GetPR() error = %v, want %v", err, ErrRateLimited)
	}
}

func TestConvertAPIPR_MergedState(t *testing.T) {
	api := &apiPR{
		Number:  1,
		HTMLURL: "https://github.com/o/r/pull/1",
		State:   "closed",
		Merged:  true,
	}

	pr := convertAPIPR(api)

	if pr.State != "merged" {
		t.Errorf("PR.State = %q, want %q", pr.State, "merged")
	}
}

func TestConvertAPIPR_HasConflicts(t *testing.T) {
	mergeable := false
	api := &apiPR{
		Number:         1,
		HTMLURL:        "https://github.com/o/r/pull/1",
		State:          "open",
		Mergeable:      &mergeable,
		MergeableState: "dirty",
	}

	pr := convertAPIPR(api)

	if !pr.HasConflicts {
		t.Error("PR.HasConflicts = false, want true")
	}
	if pr.Mergeable {
		t.Error("PR.Mergeable = true, want false")
	}
}

func TestNormalizeReviewState(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"APPROVED", "approved"},
		{"CHANGES_REQUESTED", "changes_requested"},
		{"COMMENTED", "commented"},
		{"PENDING", "pending"},
		{"DISMISSED", ""},
		{"UNKNOWN", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeReviewState(tt.input); got != tt.want {
				t.Errorf("normalizeReviewState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCIStatusMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/pulls/1":
			json.NewEncoder(w).Encode(apiPR{
				Number: 1,
				Head: struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				}{SHA: "abc"},
			})
		case "/repos/owner/repo/commits/abc/status":
			json.NewEncoder(w).Encode(apiCombinedStatus{State: "failure"})
		case "/repos/owner/repo/pulls/1/reviews":
			json.NewEncoder(w).Encode([]apiReview{})
		}
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	pr, _ := client.GetPR(repo, 1)

	if pr.CIStatus != "failed" {
		t.Errorf("PR.CIStatus = %q, want %q", pr.CIStatus, "failed")
	}
}

func TestReviewStatusPriority(t *testing.T) {
	// changes_requested should override approved
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/pulls/1":
			json.NewEncoder(w).Encode(apiPR{
				Number: 1,
				Head: struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				}{SHA: "abc"},
			})
		case "/repos/owner/repo/commits/abc/status":
			json.NewEncoder(w).Encode(apiCombinedStatus{State: "success"})
		case "/repos/owner/repo/pulls/1/reviews":
			json.NewEncoder(w).Encode([]apiReview{
				{User: struct {
					Login string `json:"login"`
				}{Login: "user1"}, State: "APPROVED"},
				{User: struct {
					Login string `json:"login"`
				}{Login: "user2"}, State: "CHANGES_REQUESTED"},
			})
		}
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	pr, _ := client.GetPR(repo, 1)

	if pr.ReviewStatus != "changes_requested" {
		t.Errorf("PR.ReviewStatus = %q, want %q", pr.ReviewStatus, "changes_requested")
	}
}
