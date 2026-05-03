package status

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/cli/render"
	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── FR-9.1 Feature JSON ──────────────────────────────────────────────────────

func TestJSONRenderer_RenderFeature_Full(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.FeatureInput{
		ID:          "FEAT-042",
		DisplayID:   "F-042",
		Slug:        "my-feature",
		Status:      "developing",
		PlanID:      "P1-my-plan",
		TasksActive: 1,
		TasksReady:  3,
		TasksDone:   7,
		TasksTotal:  11,
		Documents: []render.DocInput{
			{ID: "DOC-0019", Type: "design", Path: "work/design/my-feature.md", Status: "approved"},
			{ID: "DOC-0023", Type: "specification", Path: "work/spec/my-feature-spec.md", Status: "approved"},
		},
		Attention: []render.AttentionItem{
			{Severity: "warning", Message: "No dev-plan document registered — agents cannot begin planning"},
		},
	}

	out, err := r.RenderFeature(in)
	if err != nil {
		t.Fatalf("RenderFeature error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(out, &wrapper); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(wrapper.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(wrapper.Results))
	}
	res := wrapper.Results[0]

	// scope
	if res["scope"] != "feature" {
		t.Errorf("scope = %v, want feature", res["scope"])
	}

	// feature sub-object
	feat, ok := res["feature"].(map[string]any)
	if !ok {
		t.Fatal("feature is not an object")
	}
	if feat["id"] != "FEAT-042" {
		t.Errorf("feature.id = %v, want FEAT-042", feat["id"])
	}
	if feat["display_id"] != "F-042" {
		t.Errorf("feature.display_id = %v, want F-042", feat["display_id"])
	}
	if feat["slug"] != "my-feature" {
		t.Errorf("feature.slug = %v, want my-feature", feat["slug"])
	}
	if feat["status"] != "developing" {
		t.Errorf("feature.status = %v, want developing", feat["status"])
	}
	if feat["plan_id"] != "P1-my-plan" {
		t.Errorf("feature.plan_id = %v, want P1-my-plan", feat["plan_id"])
	}

	// documents
	docs, ok := res["documents"].(map[string]any)
	if !ok {
		t.Fatal("documents is not an object")
	}
	if docs["design"] == nil {
		t.Error("documents.design is nil, want object")
	}
	if docs["spec"] == nil {
		t.Error("documents.spec is nil, want object")
	}
	if docs["dev-plan"] != nil {
		t.Error("documents.dev-plan is not nil, want nil")
	}

	// tasks
	tasks, ok := res["tasks"].(map[string]any)
	if !ok {
		t.Fatal("tasks is not an object")
	}
	if tasks["active"] != float64(1) {
		t.Errorf("tasks.active = %v, want 1", tasks["active"])
	}
	if tasks["ready"] != float64(3) {
		t.Errorf("tasks.ready = %v, want 3", tasks["ready"])
	}
	if tasks["done"] != float64(7) {
		t.Errorf("tasks.done = %v, want 7", tasks["done"])
	}
	if tasks["total"] != float64(11) {
		t.Errorf("tasks.total = %v, want 11", tasks["total"])
	}

	// attention
	attn, ok := res["attention"].([]any)
	if !ok {
		t.Fatal("attention is not an array")
	}
	if len(attn) != 1 {
		t.Fatalf("attention len = %d, want 1", len(attn))
	}
}

// ─── FR-9.1 null plan_id ──────────────────────────────────────────────────────

func TestJSONRenderer_RenderFeature_NullPlanID(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.FeatureInput{
		ID:        "FEAT-099",
		DisplayID: "F-099",
		Slug:      "no-plan",
		Status:    "designing",
		PlanID:    "",
	}

	out, err := r.RenderFeature(in)
	if err != nil {
		t.Fatalf("RenderFeature error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)

	feat := wrapper.Results[0]["feature"].(map[string]any)
	if feat["plan_id"] != nil {
		t.Errorf("feature.plan_id = %v, want nil", feat["plan_id"])
	}
}

// ─── FR-9.2 Plan JSON ─────────────────────────────────────────────────────────

func TestJSONRenderer_RenderPlan(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.PlanInput{
		ID:     "P1-my-plan",
		Slug:   "my-plan",
		Status: "active",
		Features: []render.PlanFeatureInput{
			{Status: "developing"},
			{Status: "reviewing"},
			{Status: "done"},
			{Status: "done"},
			{Status: "done"},
		},
	}

	out, err := r.RenderPlan(in)
	if err != nil {
		t.Fatalf("RenderPlan error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)
	res := wrapper.Results[0]

	if res["scope"] != "plan" {
		t.Errorf("scope = %v, want plan", res["scope"])
	}

	plan := res["plan"].(map[string]any)
	if plan["id"] != "P1-my-plan" {
		t.Errorf("plan.id = %v", plan["id"])
	}

	feats := res["features"].(map[string]any)
	if feats["active"] != float64(2) {
		t.Errorf("features.active = %v, want 2", feats["active"])
	}
	if feats["done"] != float64(3) {
		t.Errorf("features.done = %v, want 3", feats["done"])
	}
	if feats["total"] != float64(5) {
		t.Errorf("features.total = %v, want 5", feats["total"])
	}
}

// ─── FR-9.3 Task JSON ─────────────────────────────────────────────────────────

func TestJSONRenderer_RenderTask(t *testing.T) {
	r := &JSONRenderer{}
	out, err := r.RenderTask("TASK-0099", "implement-output-flag", "active", "FEAT-042", nil)
	if err != nil {
		t.Fatalf("RenderTask error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)
	res := wrapper.Results[0]

	if res["scope"] != "task" {
		t.Errorf("scope = %v, want task", res["scope"])
	}

	task := res["task"].(map[string]any)
	if task["id"] != "TASK-0099" {
		t.Errorf("task.id = %v", task["id"])
	}
	if task["parent_feature_id"] != "FEAT-042" {
		t.Errorf("task.parent_feature_id = %v", task["parent_feature_id"])
	}

	attn := res["attention"].([]any)
	if len(attn) != 0 {
		t.Errorf("attention len = %d, want 0", len(attn))
	}
}

// ─── FR-9.4 Bug JSON ──────────────────────────────────────────────────────────

func TestJSONRenderer_RenderBug(t *testing.T) {
	r := &JSONRenderer{}
	out, err := r.RenderBug("BUG-0017", "crash-on-empty-project", "active", "high", "", nil)
	if err != nil {
		t.Fatalf("RenderBug error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)
	res := wrapper.Results[0]

	bug := res["bug"].(map[string]any)
	if bug["id"] != "BUG-0017" {
		t.Errorf("bug.id = %v", bug["id"])
	}
	if bug["parent_feature_id"] != nil {
		t.Errorf("bug.parent_feature_id = %v, want nil", bug["parent_feature_id"])
	}
}

func TestJSONRenderer_RenderBug_WithParentFeature(t *testing.T) {
	r := &JSONRenderer{}
	out, err := r.RenderBug("BUG-0017", "crash", "active", "high", "FEAT-042", nil)
	if err != nil {
		t.Fatalf("RenderBug error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)
	bug := wrapper.Results[0]["bug"].(map[string]any)
	if bug["parent_feature_id"] != "FEAT-042" {
		t.Errorf("bug.parent_feature_id = %v, want FEAT-042", bug["parent_feature_id"])
	}
}

// ─── FR-9.5 Document JSON ─────────────────────────────────────────────────────

func TestJSONRenderer_RenderDocument_Registered(t *testing.T) {
	r := &JSONRenderer{}
	d := &service.DocumentResult{
		ID:     "DOC-0023",
		Path:   "work/spec/my-feature-spec.md",
		Type:   "specification",
		Status: "approved",
		Owner:  "FEAT-042",
	}

	out, err := r.RenderDocument(d)
	if err != nil {
		t.Fatalf("RenderDocument error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)
	res := wrapper.Results[0]

	if res["scope"] != "document" {
		t.Errorf("scope = %v, want document", res["scope"])
	}

	doc := res["document"].(map[string]any)
	if doc["id"] != "DOC-0023" {
		t.Errorf("document.id = %v, want DOC-0023", doc["id"])
	}
	if doc["registered"] != true {
		t.Errorf("document.registered = %v, want true", doc["registered"])
	}
	if doc["owner_id"] != "FEAT-042" {
		t.Errorf("document.owner_id = %v, want FEAT-042", doc["owner_id"])
	}

	attn := res["attention"].([]any)
	if len(attn) != 0 {
		t.Errorf("attention len = %d, want 0", len(attn))
	}
}

// ─── FR-9.6 Unregistered Document ─────────────────────────────────────────────

func TestJSONRenderer_RenderDocument_Unregistered(t *testing.T) {
	r := &JSONRenderer{}
	d := &service.DocumentResult{
		Path: "work/spec/unregistered-spec.md",
	}

	out, err := r.RenderDocument(d)
	if err != nil {
		t.Fatalf("RenderDocument error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)
	res := wrapper.Results[0]

	doc := res["document"].(map[string]any)
	if doc["registered"] != false {
		t.Errorf("document.registered = %v, want false", doc["registered"])
	}
	if doc["id"] != nil {
		t.Errorf("document.id = %v, want nil", doc["id"])
	}
	if doc["type"] != nil {
		t.Errorf("document.type = %v, want nil", doc["type"])
	}
	if doc["status"] != nil {
		t.Errorf("document.status = %v, want nil", doc["status"])
	}
	if doc["owner_id"] != nil {
		t.Errorf("document.owner_id = %v, want nil", doc["owner_id"])
	}

	attn := res["attention"].([]any)
	if len(attn) == 0 {
		t.Error("attention is empty, want warning for unregistered doc")
	}
}

// ─── FR-9.9 Empty attention → [] ──────────────────────────────────────────────

func TestJSONRenderer_EmptyAttention(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.FeatureInput{
		ID:        "FEAT-001",
		DisplayID: "F-001",
		Slug:      "clean",
		Status:    "done",
	}

	out, err := r.RenderFeature(in)
	if err != nil {
		t.Fatalf("RenderFeature error: %v", err)
	}

	var wrapper struct {
		Results []map[string]any `json:"results"`
	}
	json.Unmarshal(out, &wrapper)

	attn := wrapper.Results[0]["attention"]
	if attn == nil {
		t.Error("attention is null, want empty array []")
	}
	arr, ok := attn.([]any)
	if !ok {
		t.Fatal("attention is not an array")
	}
	if len(arr) != 0 {
		t.Errorf("attention len = %d, want 0", len(arr))
	}
}

// ─── FR-10 Project Overview ───────────────────────────────────────────────────

func TestJSONRenderer_RenderProject(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.ProjectInput{
		Plans: []render.ProjectPlanInput{
			{DisplayID: "P1-main-plan", Status: "active", FeaturesActive: 2, FeaturesTotal: 5},
		},
		Health: &render.StatusHealthSummary{Errors: 0, Warnings: 2},
		Attention: []render.AttentionItem{
			{Severity: "warning", EntityID: "FEAT-042", Message: "No dev-plan document registered"},
		},
	}

	out, err := r.RenderProject(in)
	if err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// FR-10: project overview MUST NOT be wrapped in results array
	if _, exists := res["results"]; exists {
		t.Error("project output has 'results' key, must not be array-wrapped")
	}
	if res["scope"] != "project" {
		t.Errorf("scope = %v, want project", res["scope"])
	}

	plans, ok := res["plans"].([]any)
	if !ok {
		t.Fatal("plans is not an array")
	}
	if len(plans) != 1 {
		t.Fatalf("plans len = %d, want 1", len(plans))
	}

	health := res["health"].(map[string]any)
	if health["errors"] != float64(0) {
		t.Errorf("health.errors = %v, want 0", health["errors"])
	}
	if health["warnings"] != float64(2) {
		t.Errorf("health.warnings = %v, want 2", health["warnings"])
	}

	// FR-10.3: entity_id should be present for project attention items
	attn := res["attention"].([]any)
	if len(attn) != 1 {
		t.Fatalf("attention len = %d, want 1", len(attn))
	}
	attnItem := attn[0].(map[string]any)
	if _, exists := attnItem["entity_id"]; !exists {
		t.Error("attention[0].entity_id missing, must be present (FR-10.3)")
	}
}

// ─── FR-10.4 empty plans → [] ─────────────────────────────────────────────────

func TestJSONRenderer_RenderProject_EmptyPlans(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.ProjectInput{
		Plans:  nil,
		Health: &render.StatusHealthSummary{},
	}

	out, err := r.RenderProject(in)
	if err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	var res map[string]any
	json.Unmarshal(out, &res)

	plans := res["plans"]
	if plans == nil {
		t.Error("plans is null, want empty array []")
	}
}

// ─── FR-10.5 empty attention → [] ─────────────────────────────────────────────

func TestJSONRenderer_RenderProject_EmptyAttention(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.ProjectInput{
		Plans:  []render.ProjectPlanInput{},
		Health: &render.StatusHealthSummary{},
	}

	out, err := r.RenderProject(in)
	if err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	var res map[string]any
	json.Unmarshal(out, &res)

	attn := res["attention"]
	if attn == nil {
		t.Error("attention is null, want empty array []")
	}
}

// ─── Entity ID in attention ──────────────────────────────────────────────────

func TestJSONRenderer_AttentionEntityID_NullForProjectWide(t *testing.T) {
	r := &JSONRenderer{}
	in := &render.ProjectInput{
		Plans:  []render.ProjectPlanInput{},
		Health: &render.StatusHealthSummary{},
		Attention: []render.AttentionItem{
			{Severity: "warning", EntityID: "", Message: "Project-wide advisory"},
		},
	}

	out, err := r.RenderProject(in)
	if err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	var res map[string]any
	json.Unmarshal(out, &res)

	attn := res["attention"].([]any)
	attnItem := attn[0].(map[string]any)

	// FR-10.3: entity_id MUST be present (null) for project-wide items
	id, exists := attnItem["entity_id"]
	if !exists {
		t.Error("entity_id missing, must be present with null for project-wide items")
	} else if id != nil {
		t.Errorf("entity_id = %v, want nil for project-wide items", id)
	}
}

// ─── NFR-1.5 / AC-11: Schema contract test ─────────────────────────────────────

func TestJSONSchemaContract_RequiredFields(t *testing.T) {
	r := &JSONRenderer{}

	// Feature fields (FR-9.1)
	t.Run("feature", func(t *testing.T) {
		out, err := r.RenderFeature(&render.FeatureInput{
			ID: "x", DisplayID: "x", Slug: "x", Status: "x",
		})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONFields(t, out, []string{
			"results[0].scope",
			"results[0].feature.id",
			"results[0].feature.display_id",
			"results[0].feature.slug",
			"results[0].feature.status",
			"results[0].feature.plan_id",
			"results[0].documents.design",
			"results[0].documents.spec",
			"results[0].documents.dev-plan",
			"results[0].tasks.active",
			"results[0].tasks.ready",
			"results[0].tasks.done",
			"results[0].tasks.total",
			"results[0].attention",
		})
	})

	// Plan fields (FR-9.2)
	t.Run("plan", func(t *testing.T) {
		out, err := r.RenderPlan(&render.PlanInput{
			ID: "x", Slug: "x", Status: "x",
		})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONFields(t, out, []string{
			"results[0].scope",
			"results[0].plan.id",
			"results[0].plan.slug",
			"results[0].plan.status",
			"results[0].features.active",
			"results[0].features.done",
			"results[0].features.total",
			"results[0].attention",
		})
	})

	// Task fields (FR-9.3)
	t.Run("task", func(t *testing.T) {
		out, err := r.RenderTask("x", "x", "x", "x", nil)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONFields(t, out, []string{
			"results[0].scope",
			"results[0].task.id",
			"results[0].task.slug",
			"results[0].task.status",
			"results[0].task.parent_feature_id",
			"results[0].attention",
		})
	})

	// Bug fields (FR-9.4)
	t.Run("bug", func(t *testing.T) {
		out, err := r.RenderBug("x", "x", "x", "x", "x", nil)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONFields(t, out, []string{
			"results[0].scope",
			"results[0].bug.id",
			"results[0].bug.slug",
			"results[0].bug.status",
			"results[0].bug.severity",
			"results[0].bug.parent_feature_id",
			"results[0].attention",
		})
	})

	// Document fields (FR-9.5)
	t.Run("document", func(t *testing.T) {
		out, err := r.RenderDocument(&service.DocumentResult{
			ID: "x", Path: "x", Type: "x", Status: "x", Owner: "x",
		})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONFields(t, out, []string{
			"results[0].scope",
			"results[0].document.id",
			"results[0].document.path",
			"results[0].document.type",
			"results[0].document.status",
			"results[0].document.registered",
			"results[0].document.owner_id",
			"results[0].attention",
		})
	})

	// Project overview fields (FR-10)
	t.Run("project", func(t *testing.T) {
		out, err := r.RenderProject(&render.ProjectInput{
			Plans:  []render.ProjectPlanInput{},
			Health: &render.StatusHealthSummary{},
		})
		if err != nil {
			t.Fatal(err)
		}
		assertJSONFields(t, out, []string{
			"scope",
			"plans",
			"health.errors",
			"health.warnings",
			"attention",
		})
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// assertJSONFields verifies that each dot-separated path exists in the JSON output.
// Paths like "results[0].feature.id" navigate objects and array indices.
func assertJSONFields(t *testing.T, raw []byte, paths []string) {
	t.Helper()
	for _, path := range paths {
		if !jsonPathExists(t, raw, path) {
			t.Errorf("missing required JSON field: %s", path)
		}
	}
}

func jsonPathExists(t *testing.T, raw []byte, path string) bool {
	t.Helper()
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		// Handle array indexing: "results[0]" → key="results", idx=0
		key := part
		idx := -1
		if bra := strings.IndexByte(part, '['); bra >= 0 {
			key = part[:bra]
			rest := part[bra+1:]
			if ket := strings.IndexByte(rest, ']'); ket >= 0 {
				idxStr := rest[:ket]
				idx = 0
				for _, ch := range idxStr {
					if ch >= '0' && ch <= '9' {
						idx = idx*10 + int(ch-'0')
					}
				}
			}
		}

		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		val, exists := m[key]
		if !exists {
			return false
		}
		if idx >= 0 {
			arr, ok := val.([]any)
			if !ok || idx >= len(arr) {
				return false
			}
			current = arr[idx]
		} else {
			current = val
		}
	}
	return true
}
