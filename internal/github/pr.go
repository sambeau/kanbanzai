package github

import (
	"fmt"
	"net/url"
)

// PR represents a pull request.
type PR struct {
	URL          string   // Full URL to the PR
	Number       int      // PR number
	Title        string   // PR title
	Body         string   // PR body/description
	State        string   // "open", "closed", "merged"
	Draft        bool     // Whether the PR is a draft
	CIStatus     string   // "passed", "failed", "pending", ""
	ReviewStatus string   // "approved", "changes_requested", "pending", ""
	Reviews      []Review // Individual reviews
	HasConflicts bool     // Whether the PR has merge conflicts
	Mergeable    bool     // Whether the PR can be merged
	HeadBranch   string   // The source branch
	BaseBranch   string   // The target branch
}

// Review represents a pull request review.
type Review struct {
	User  string // Username of the reviewer
	State string // "approved", "changes_requested", "commented", "pending"
}

// apiPR is the GitHub API response structure for a pull request.
type apiPR struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	Draft   bool   `json:"draft"`
	Head    struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Merged          bool   `json:"merged"`
	Mergeable       *bool  `json:"mergeable"`
	MergeableState  string `json:"mergeable_state"`
	MergeCommitSHA  string `json:"merge_commit_sha"`
	RebaseMergeable bool   `json:"rebaseable"`
}

// apiReview is the GitHub API response structure for a review.
type apiReview struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	State string `json:"state"`
}

// apiCombinedStatus is the GitHub API response for combined commit status.
type apiCombinedStatus struct {
	State    string `json:"state"`
	Statuses []struct {
		State       string `json:"state"`
		Context     string `json:"context"`
		Description string `json:"description"`
	} `json:"statuses"`
}

// CreatePRRequest is the request body for creating a PR.
type CreatePRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft,omitempty"`
}

// UpdatePRRequest is the request body for updating a PR.
type UpdatePRRequest struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// CreatePR creates a new pull request.
func (c *Client) CreatePR(repo RepoInfo, head, base, title, body string, draft bool) (*PR, error) {
	if c.token == "" {
		return nil, ErrNoToken
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls", repo.Owner, repo.Repo)
	reqBody := CreatePRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
		Draft: draft,
	}

	var apiResp apiPR
	if err := c.post(path, reqBody, &apiResp); err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	return convertAPIPR(&apiResp), nil
}

// UpdatePR updates a pull request's title and/or body.
func (c *Client) UpdatePR(repo RepoInfo, number int, title, body string) (*PR, error) {
	if c.token == "" {
		return nil, ErrNoToken
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", repo.Owner, repo.Repo, number)
	reqBody := UpdatePRRequest{
		Title: title,
		Body:  body,
	}

	var apiResp apiPR
	if err := c.patch(path, reqBody, &apiResp); err != nil {
		if err == ErrRepoNotFound {
			return nil, ErrPRNotFound
		}
		return nil, fmt.Errorf("update PR: %w", err)
	}

	return convertAPIPR(&apiResp), nil
}

// GetPR retrieves a pull request by number, including CI and review status.
func (c *Client) GetPR(repo RepoInfo, number int) (*PR, error) {
	// Get the PR details
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", repo.Owner, repo.Repo, number)
	var apiResp apiPR
	if err := c.get(path, &apiResp); err != nil {
		if err == ErrRepoNotFound {
			return nil, ErrPRNotFound
		}
		return nil, fmt.Errorf("get PR: %w", err)
	}

	pr := convertAPIPR(&apiResp)

	// Get CI status from combined status endpoint
	ciStatus, err := c.getCIStatus(repo, apiResp.Head.SHA)
	if err == nil {
		pr.CIStatus = ciStatus
	}

	// Get reviews
	reviews, reviewStatus, err := c.getReviews(repo, number)
	if err == nil {
		pr.Reviews = reviews
		pr.ReviewStatus = reviewStatus
	}

	return pr, nil
}

// GetPRByBranch finds a pull request by its head branch.
func (c *Client) GetPRByBranch(repo RepoInfo, branch string) (*PR, error) {
	// Use the pulls endpoint with head filter
	// The head parameter format is "owner:branch"
	head := url.QueryEscape(repo.Owner + ":" + branch)
	path := fmt.Sprintf("/repos/%s/%s/pulls?head=%s&state=all", repo.Owner, repo.Repo, head)

	var prs []apiPR
	if err := c.get(path, &prs); err != nil {
		return nil, fmt.Errorf("list PRs by branch: %w", err)
	}

	if len(prs) == 0 {
		return nil, ErrPRNotFound
	}

	// Return the first (most recent) PR for this branch
	pr := convertAPIPR(&prs[0])

	// Get additional status info
	ciStatus, err := c.getCIStatus(repo, prs[0].Head.SHA)
	if err == nil {
		pr.CIStatus = ciStatus
	}

	reviews, reviewStatus, err := c.getReviews(repo, prs[0].Number)
	if err == nil {
		pr.Reviews = reviews
		pr.ReviewStatus = reviewStatus
	}

	return pr, nil
}

// getCIStatus retrieves the combined CI status for a commit.
func (c *Client) getCIStatus(repo RepoInfo, sha string) (string, error) {
	path := fmt.Sprintf("/repos/%s/%s/commits/%s/status", repo.Owner, repo.Repo, sha)
	var status apiCombinedStatus
	if err := c.get(path, &status); err != nil {
		return "", err
	}

	// GitHub returns "success", "failure", "pending", or "error"
	switch status.State {
	case "success":
		return "passed", nil
	case "failure", "error":
		return "failed", nil
	case "pending":
		return "pending", nil
	default:
		return "", nil
	}
}

// getReviews retrieves all reviews for a PR and determines the overall status.
func (c *Client) getReviews(repo RepoInfo, number int) ([]Review, string, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", repo.Owner, repo.Repo, number)
	var apiReviews []apiReview
	if err := c.get(path, &apiReviews); err != nil {
		return nil, "", err
	}

	// Track the latest review state per user
	latestReviews := make(map[string]string)
	for _, r := range apiReviews {
		// Only consider reviews with meaningful states
		state := normalizeReviewState(r.State)
		if state != "" {
			latestReviews[r.User.Login] = state
		}
	}

	// Convert to Review slice
	reviews := make([]Review, 0, len(latestReviews))
	for user, state := range latestReviews {
		reviews = append(reviews, Review{User: user, State: state})
	}

	// Determine overall review status
	// Priority: changes_requested > approved > pending
	overallStatus := ""
	hasApproval := false
	hasChangesRequested := false

	for _, state := range latestReviews {
		switch state {
		case "changes_requested":
			hasChangesRequested = true
		case "approved":
			hasApproval = true
		}
	}

	if hasChangesRequested {
		overallStatus = "changes_requested"
	} else if hasApproval {
		overallStatus = "approved"
	} else if len(reviews) > 0 {
		overallStatus = "pending"
	}

	return reviews, overallStatus, nil
}

// normalizeReviewState converts GitHub API review states to our internal states.
func normalizeReviewState(state string) string {
	switch state {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes_requested"
	case "COMMENTED":
		return "commented"
	case "PENDING":
		return "pending"
	case "DISMISSED":
		return "" // Dismissed reviews don't count
	default:
		return ""
	}
}

// convertAPIPR converts a GitHub API PR response to our PR type.
func convertAPIPR(api *apiPR) *PR {
	state := api.State
	if api.Merged {
		state = "merged"
	}

	// Determine merge conflict status
	hasConflicts := false
	mergeable := false
	if api.Mergeable != nil {
		mergeable = *api.Mergeable
		// If not mergeable and state indicates conflicts
		if !*api.Mergeable && api.MergeableState == "dirty" {
			hasConflicts = true
		}
	}

	return &PR{
		URL:          api.HTMLURL,
		Number:       api.Number,
		Title:        api.Title,
		Body:         api.Body,
		State:        state,
		Draft:        api.Draft,
		HasConflicts: hasConflicts,
		Mergeable:    mergeable,
		HeadBranch:   api.Head.Ref,
		BaseBranch:   api.Base.Ref,
	}
}
