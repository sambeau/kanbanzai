// Package render provides TTY-aware CLI rendering for status views.
package render

// StatusHealthSummary mirrors the health summary from the MCP status tool.
type StatusHealthSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// AttentionItem is a structured attention entry in status response objects.
type AttentionItem struct {
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	EntityID  string `json:"entity_id,omitempty"`
	DisplayID string `json:"display_id,omitempty"`
	Message   string `json:"message"`
}

// FeatureInput is the input for RenderFeature.
// It mirrors the synthesis output from the MCP status tool.
type FeatureInput struct {
	DisplayID   string
	ID          string
	Slug        string
	Summary     string
	Status      string
	PlanID      string
	PlanName    string
	TasksActive int
	TasksReady  int
	TasksDone   int
	TasksTotal  int
	Documents   []DocInput
	Attention   []AttentionItem
}

// DocInput holds document info for rendering.
type DocInput struct {
	Type   string
	Path   string
	Status string
}

// PlanInput is the input for RenderPlan.
type PlanInput struct {
	DisplayID   string
	ID          string
	Slug        string
	Name        string
	Status      string
	Features    []PlanFeatureInput
	TasksActive int
	TasksReady  int
	TasksDone   int
	TasksTotal  int
	Attention   []AttentionItem
}

// PlanFeatureInput holds a feature summary within a plan dashboard.
type PlanFeatureInput struct {
	DisplayID   string
	Slug        string
	Status      string
	HasDevPlan  bool
}

// ProjectInput is the input for RenderProject.
type ProjectInput struct {
	Name       string
	Plans      []ProjectPlanInput
	Health     *StatusHealthSummary
	Attention  []AttentionItem
	WorkQueue  ProjectWorkQueue
}

// ProjectPlanInput holds a strategic plan summary for the project overview.
type ProjectPlanInput struct {
	DisplayID        string
	Status           string
	FeaturesActive   int
	FeaturesTotal    int
}

// ProjectWorkQueue holds the task queue counts for the project overview.
type ProjectWorkQueue struct {
	Ready  int
	Active int
}
