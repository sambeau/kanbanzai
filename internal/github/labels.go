package github

import (
	"context"
	"fmt"
	"strings"
)

// Standard labels used by Kanbanzai.
const (
	LabelFeature      = "feature"
	LabelBug          = "bug"
	LabelTaskComplete = "tasks-complete"
	LabelVerified     = "verified"
	LabelReadyToMerge = "ready-to-merge"
)

// LabelColors defines the default colors for each label.
var LabelColors = map[string]string{
	LabelFeature:      "0366d6", // Blue
	LabelBug:          "d73a4a", // Red
	LabelTaskComplete: "28a745", // Green
	LabelVerified:     "6f42c1", // Purple
	LabelReadyToMerge: "2ea44f", // Bright green
}

// LabelDescriptions defines descriptions for each label.
var LabelDescriptions = map[string]string{
	LabelFeature:      "Feature development work",
	LabelBug:          "Bug fix work",
	LabelTaskComplete: "All tasks are complete",
	LabelVerified:     "Verification criteria passed",
	LabelReadyToMerge: "All merge gates pass - ready for merge",
}

// apiLabel is the GitHub API response structure for a label.
type apiLabel struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// createLabelRequest is the request body for creating a label.
type createLabelRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}

// setLabelsRequest is the request body for setting labels on an issue/PR.
type setLabelsRequest struct {
	Labels []string `json:"labels"`
}

// EnsureLabel creates a label if it doesn't exist.
// If the label already exists, this is a no-op.
func (c *Client) EnsureLabel(ctx context.Context, repo RepoInfo, name, color, description string) error {
	if c.token == "" {
		return ErrNoToken
	}

	// Strip # from color if present
	color = strings.TrimPrefix(color, "#")

	path := fmt.Sprintf("/repos/%s/%s/labels", repo.Owner, repo.Repo)
	reqBody := createLabelRequest{
		Name:        name,
		Color:       color,
		Description: description,
	}

	resp, err := c.doRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return fmt.Errorf("create label: %w", err)
	}
	defer resp.Body.Close()

	// 201 = created, 422 = already exists (validation failed)
	if resp.StatusCode == 201 || resp.StatusCode == 422 {
		return nil
	}

	return c.checkResponse(resp)
}

// EnsureStandardLabels ensures all standard Kanbanzai labels exist in the repository.
func (c *Client) EnsureStandardLabels(ctx context.Context, repo RepoInfo) error {
	labels := []string{
		LabelFeature,
		LabelBug,
		LabelTaskComplete,
		LabelVerified,
		LabelReadyToMerge,
	}

	for _, label := range labels {
		color := LabelColors[label]
		desc := LabelDescriptions[label]
		if err := c.EnsureLabel(ctx, repo, label, color, desc); err != nil {
			return fmt.Errorf("ensure label %s: %w", label, err)
		}
	}

	return nil
}

// SetPRLabels sets the labels on a PR.
// This replaces all existing labels with the provided set.
func (c *Client) SetPRLabels(ctx context.Context, repo RepoInfo, number int, labels []string) error {
	if c.token == "" {
		return ErrNoToken
	}

	// GitHub uses the issues endpoint for labels on PRs
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", repo.Owner, repo.Repo, number)
	reqBody := setLabelsRequest{
		Labels: labels,
	}

	resp, err := c.doRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return fmt.Errorf("set PR labels: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		if err == ErrRepoNotFound {
			return ErrPRNotFound
		}
		return fmt.Errorf("set PR labels: %w", err)
	}

	return nil
}

// AddPRLabels adds labels to a PR without removing existing ones.
func (c *Client) AddPRLabels(ctx context.Context, repo RepoInfo, number int, labels []string) error {
	if c.token == "" {
		return ErrNoToken
	}

	// GitHub uses the issues endpoint for labels on PRs
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", repo.Owner, repo.Repo, number)
	reqBody := setLabelsRequest{
		Labels: labels,
	}

	resp, err := c.doRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return fmt.Errorf("add PR labels: %w", err)
	}
	defer resp.Body.Close()

	if err := c.checkResponse(resp); err != nil {
		if err == ErrRepoNotFound {
			return ErrPRNotFound
		}
		return fmt.Errorf("add PR labels: %w", err)
	}

	return nil
}

// ComputeLabels determines which labels apply to an entity based on its state.
//
// Label conditions:
//   - Entity is Feature → "feature"
//   - Entity is Bug → "bug"
//   - All tasks complete → "tasks-complete"
//   - Verification passed → "verified"
//   - Merge gates pass → "ready-to-merge"
func ComputeLabels(entityType string, tasksComplete, verificationPassed, gatesPass bool) []string {
	var labels []string

	// Entity type label
	switch strings.ToLower(entityType) {
	case "feature":
		labels = append(labels, LabelFeature)
	case "bug":
		labels = append(labels, LabelBug)
	}

	// Status-based labels
	if tasksComplete {
		labels = append(labels, LabelTaskComplete)
	}

	if verificationPassed {
		labels = append(labels, LabelVerified)
	}

	if gatesPass {
		labels = append(labels, LabelReadyToMerge)
	}

	return labels
}
