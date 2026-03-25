package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComputeLabels(t *testing.T) {
	tests := []struct {
		name               string
		entityType         string
		tasksComplete      bool
		verificationPassed bool
		gatesPass          bool
		want               []string
	}{
		{
			name:               "Feature with nothing complete",
			entityType:         "feature",
			tasksComplete:      false,
			verificationPassed: false,
			gatesPass:          false,
			want:               []string{LabelFeature},
		},
		{
			name:               "Bug with nothing complete",
			entityType:         "bug",
			tasksComplete:      false,
			verificationPassed: false,
			gatesPass:          false,
			want:               []string{LabelBug},
		},
		{
			name:               "Feature with tasks complete",
			entityType:         "feature",
			tasksComplete:      true,
			verificationPassed: false,
			gatesPass:          false,
			want:               []string{LabelFeature, LabelTaskComplete},
		},
		{
			name:               "Feature with verification passed",
			entityType:         "feature",
			tasksComplete:      false,
			verificationPassed: true,
			gatesPass:          false,
			want:               []string{LabelFeature, LabelVerified},
		},
		{
			name:               "Feature ready to merge",
			entityType:         "feature",
			tasksComplete:      true,
			verificationPassed: true,
			gatesPass:          true,
			want:               []string{LabelFeature, LabelTaskComplete, LabelVerified, LabelReadyToMerge},
		},
		{
			name:               "Bug ready to merge",
			entityType:         "bug",
			tasksComplete:      true,
			verificationPassed: true,
			gatesPass:          true,
			want:               []string{LabelBug, LabelTaskComplete, LabelVerified, LabelReadyToMerge},
		},
		{
			name:               "Feature case insensitive",
			entityType:         "FEATURE",
			tasksComplete:      false,
			verificationPassed: false,
			gatesPass:          false,
			want:               []string{LabelFeature},
		},
		{
			name:               "Bug case insensitive",
			entityType:         "BUG",
			tasksComplete:      false,
			verificationPassed: false,
			gatesPass:          false,
			want:               []string{LabelBug},
		},
		{
			name:               "Unknown entity type",
			entityType:         "task",
			tasksComplete:      true,
			verificationPassed: true,
			gatesPass:          true,
			want:               []string{LabelTaskComplete, LabelVerified, LabelReadyToMerge},
		},
		{
			name:               "Empty entity type",
			entityType:         "",
			tasksComplete:      false,
			verificationPassed: false,
			gatesPass:          false,
			want:               nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeLabels(tt.entityType, tt.tasksComplete, tt.verificationPassed, tt.gatesPass)

			if len(got) != len(tt.want) {
				t.Errorf("ComputeLabels() = %v, want %v", got, tt.want)
				return
			}

			for i, label := range got {
				if label != tt.want[i] {
					t.Errorf("ComputeLabels()[%d] = %q, want %q", i, label, tt.want[i])
				}
			}
		})
	}
}

func TestClient_EnsureLabel(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		statusCode int
		wantErr    error
	}{
		{
			name:    "No token",
			token:   "",
			wantErr: ErrNoToken,
		},
		{
			name:       "Label created",
			token:      "valid-token",
			statusCode: http.StatusCreated,
			wantErr:    nil,
		},
		{
			name:       "Label already exists",
			token:      "valid-token",
			statusCode: 422, // Validation failed (label exists)
			wantErr:    nil,
		},
		{
			name:       "Unauthorized",
			token:      "bad-token",
			statusCode: http.StatusUnauthorized,
			wantErr:    ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("Method = %q, want POST", r.Method)
				}
				if r.URL.Path != "/repos/owner/repo/labels" {
					t.Errorf("Path = %q, want /repos/owner/repo/labels", r.URL.Path)
				}

				// Decode and verify request body
				var req createLabelRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					if req.Name != "test-label" {
						t.Errorf("Request name = %q, want %q", req.Name, "test-label")
					}
					if req.Color != "ff0000" {
						t.Errorf("Request color = %q, want %q", req.Color, "ff0000")
					}
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(tt.token)
			client.SetBaseURL(server.URL)

			repo := RepoInfo{Owner: "owner", Repo: "repo"}
			err := client.EnsureLabel(context.Background(), repo, "test-label", "#ff0000", "Test description")

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("EnsureLabel() error = %v, wantErr = %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("EnsureLabel() unexpected error = %v", err)
			}
		})
	}
}

func TestClient_EnsureLabel_StripsHashFromColor(t *testing.T) {
	var receivedColor string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req createLabelRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedColor = req.Color
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	client.EnsureLabel(context.Background(), repo, "label", "#ff0000", "")

	if receivedColor != "ff0000" {
		t.Errorf("Color = %q, want %q (should strip #)", receivedColor, "ff0000")
	}
}

func TestClient_SetPRLabels(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		labels     []string
		statusCode int
		wantErr    error
	}{
		{
			name:    "No token",
			token:   "",
			labels:  []string{"label1"},
			wantErr: ErrNoToken,
		},
		{
			name:       "Successfully set labels",
			token:      "valid-token",
			labels:     []string{"feature", "verified"},
			statusCode: http.StatusOK,
			wantErr:    nil,
		},
		{
			name:       "PR not found",
			token:      "valid-token",
			labels:     []string{"label"},
			statusCode: http.StatusNotFound,
			wantErr:    ErrPRNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPut {
					t.Errorf("Method = %q, want PUT", r.Method)
				}
				if r.URL.Path != "/repos/owner/repo/issues/42/labels" {
					t.Errorf("Path = %q, want /repos/owner/repo/issues/42/labels", r.URL.Path)
				}

				// Verify labels in request body
				var req setLabelsRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					if len(req.Labels) != len(tt.labels) {
						t.Errorf("Labels count = %d, want %d", len(req.Labels), len(tt.labels))
					}
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode([]apiLabel{})
				}
			}))
			defer server.Close()

			client := NewClient(tt.token)
			client.SetBaseURL(server.URL)

			repo := RepoInfo{Owner: "owner", Repo: "repo"}
			err := client.SetPRLabels(context.Background(), repo, 42, tt.labels)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("SetPRLabels() error = %v, wantErr = %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("SetPRLabels() unexpected error = %v", err)
			}
		})
	}
}

func TestClient_AddPRLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify POST method (add vs PUT for set)
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/issues/42/labels" {
			t.Errorf("Path = %q, want /repos/owner/repo/issues/42/labels", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]apiLabel{})
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	err := client.AddPRLabels(context.Background(), repo, 42, []string{"new-label"})

	if err != nil {
		t.Errorf("AddPRLabels() unexpected error = %v", err)
	}
}

func TestClient_EnsureStandardLabels(t *testing.T) {
	createdLabels := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req createLabelRequest
		json.NewDecoder(r.Body).Decode(&req)
		createdLabels[req.Name] = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient("token")
	client.SetBaseURL(server.URL)

	repo := RepoInfo{Owner: "owner", Repo: "repo"}
	err := client.EnsureStandardLabels(context.Background(), repo)

	if err != nil {
		t.Fatalf("EnsureStandardLabels() error = %v", err)
	}

	expectedLabels := []string{
		LabelFeature,
		LabelBug,
		LabelTaskComplete,
		LabelVerified,
		LabelReadyToMerge,
	}

	for _, label := range expectedLabels {
		if !createdLabels[label] {
			t.Errorf("Label %q was not created", label)
		}
	}
}

func TestLabelColors(t *testing.T) {
	// Verify all standard labels have colors defined
	standardLabels := []string{
		LabelFeature,
		LabelBug,
		LabelTaskComplete,
		LabelVerified,
		LabelReadyToMerge,
	}

	for _, label := range standardLabels {
		if _, ok := LabelColors[label]; !ok {
			t.Errorf("No color defined for label %q", label)
		}
	}
}

func TestLabelDescriptions(t *testing.T) {
	// Verify all standard labels have descriptions defined
	standardLabels := []string{
		LabelFeature,
		LabelBug,
		LabelTaskComplete,
		LabelVerified,
		LabelReadyToMerge,
	}

	for _, label := range standardLabels {
		if _, ok := LabelDescriptions[label]; !ok {
			t.Errorf("No description defined for label %q", label)
		}
	}
}
