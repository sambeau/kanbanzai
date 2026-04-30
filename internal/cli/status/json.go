// Package status provides machine-readable status output renderers for the kbz CLI.
package status

import (
	"encoding/json"

	"github.com/sambeau/kanbanzai/internal/cli/render"
	"github.com/sambeau/kanbanzai/internal/service"
)

// JSONRenderer produces RFC 8259 JSON output from service-layer synthesis structs.
// Entity and document queries are wrapped in a "results" array.
// The project overview uses a distinct "scope":"project" top-level shape.
type JSONRenderer struct{}

// ─── Intermediate JSON schema structs ─────────────────────────────────────────

type documentSlot struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

type featureResult struct {
	Scope     string          `json:"scope"`
	Feature   jsonFeature     `json:"feature"`
	Documents jsonDocs        `json:"documents"`
	Tasks     jsonTaskSummary `json:"tasks"`
	Attention []jsonAttn      `json:"attention"`
}

type jsonFeature struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
	PlanID any    `json:"plan_id"`
}

type jsonDocs struct {
	Design  *documentSlot `json:"design"`
	Spec    *documentSlot `json:"spec"`
	DevPlan *documentSlot `json:"dev_plan"`
}

type jsonTaskSummary struct {
	Active int `json:"active"`
	Ready  int `json:"ready"`
	Done   int `json:"done"`
	Total  int `json:"total"`
}

type planResult struct {
	Scope     string       `json:"scope"`
	Plan      jsonPlanHdr  `json:"plan"`
	Features  jsonFeatCnt  `json:"features"`
	Attention []jsonAttn   `json:"attention"`
}

type jsonPlanHdr struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

type jsonFeatCnt struct {
	Active int `json:"active"`
	Done   int `json:"done"`
	Total  int `json:"total"`
}

type taskResult struct {
	Scope     string    `json:"scope"`
	Task      jsonTask  `json:"task"`
	Attention []jsonAttn `json:"attention"`
}

type jsonTask struct {
	ID              string `json:"id"`
	Slug            string `json:"slug"`
	Status          string `json:"status"`
	ParentFeatureID string `json:"parent_feature_id"`
}

type bugResult struct {
	Scope     string    `json:"scope"`
	Bug       jsonBug   `json:"bug"`
	Attention []jsonAttn `json:"attention"`
}

type jsonBug struct {
	ID              string `json:"id"`
	Slug            string `json:"slug"`
	Status          string `json:"status"`
	Severity        string `json:"severity"`
	ParentFeatureID any    `json:"parent_feature_id"`
}

type docResult struct {
	Scope     string    `json:"scope"`
	Document  jsonDoc   `json:"document"`
	Attention []jsonAttn `json:"attention"`
}

type jsonDoc struct {
	ID         any  `json:"id"`
	Path       string `json:"path"`
	Type       any    `json:"type"`
	Status     any    `json:"status"`
	Registered bool   `json:"registered"`
	OwnerID    any    `json:"owner_id"`
}

type projectResult struct {
	Scope     string            `json:"scope"`
	Plans     []jsonProjectPlan `json:"plans"`
	Health    jsonHealth        `json:"health"`
	Attention []jsonAttn        `json:"attention"`
}

type jsonProjectPlan struct {
	ID       string      `json:"id"`
	Slug     string      `json:"slug"`
	Status   string      `json:"status"`
	Features jsonFeatCnt `json:"features"`
}

type jsonHealth struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

type jsonAttn struct {
	Severity string `json:"severity"`
	EntityID any    `json:"entity_id,omitempty"`
	Message  string `json:"message"`
}

// ─── RenderFeature ────────────────────────────────────────────────────────────

func (r *JSONRenderer) RenderFeature(in *render.FeatureInput) ([]byte, error) {
	byType := map[string]*documentSlot{}
	for _, d := range in.Documents {
		byType[d.Type] = &documentSlot{ID: d.Path, Path: d.Path, Status: d.Status}
	}

	var planID any
	if in.PlanID != "" {
		planID = in.PlanID
	}

	fr := featureResult{
		Scope: "feature",
		Feature: jsonFeature{
			ID:     in.ID,
			Slug:   in.Slug,
			Status: in.Status,
			PlanID: planID,
		},
		Documents: jsonDocs{
			Design:  byType["design"],
			Spec:    byType["specification"],
			DevPlan: byType["dev-plan"],
		},
		Tasks: jsonTaskSummary{
			Active: in.TasksActive,
			Ready:  in.TasksReady,
			Done:   in.TasksDone,
			Total:  in.TasksTotal,
		},
		Attention: attnToJSON(in.Attention),
	}

	return marshalResults(fr)
}

// ─── RenderPlan ───────────────────────────────────────────────────────────────

func (r *JSONRenderer) RenderPlan(in *render.PlanInput) ([]byte, error) {
	active, done, total := 0, 0, len(in.Features)
	for _, f := range in.Features {
		switch f.Status {
		case "done", "closed", "merged":
			done++
		case "active", "developing", "reviewing":
			active++
		}
	}

	pr := planResult{
		Scope: "plan",
		Plan: jsonPlanHdr{
			ID:     in.ID,
			Slug:   in.Slug,
			Status: in.Status,
		},
		Features: jsonFeatCnt{
			Active: active,
			Done:   done,
			Total:  total,
		},
		Attention: attnToJSON(in.Attention),
	}

	return marshalResults(pr)
}

// ─── RenderTask ───────────────────────────────────────────────────────────────

func (r *JSONRenderer) RenderTask(id, slug, status, parentFeature string, attention []render.AttentionItem) ([]byte, error) {
	tr := taskResult{
		Scope: "task",
		Task: jsonTask{
			ID:              id,
			Slug:            slug,
			Status:          status,
			ParentFeatureID: parentFeature,
		},
		Attention: attnToJSON(attention),
	}
	return marshalResults(tr)
}

// ─── RenderBug ────────────────────────────────────────────────────────────────

func (r *JSONRenderer) RenderBug(id, slug, status, severity string, attention []render.AttentionItem) ([]byte, error) {
	br := bugResult{
		Scope: "bug",
		Bug: jsonBug{
			ID:              id,
			Slug:            slug,
			Status:          status,
			Severity:        severity,
			ParentFeatureID: nil,
		},
		Attention: attnToJSON(attention),
	}
	return marshalResults(br)
}

// ─── RenderDocument ───────────────────────────────────────────────────────────

func (r *JSONRenderer) RenderDocument(d *service.DocumentResult) ([]byte, error) {
	registered := d.ID != ""

	var docID any
	if registered {
		docID = d.ID
	}

	var docType any
	if d.Type != "" {
		docType = d.Type
	}

	var docStatus any
	if d.Status != "" {
		docStatus = d.Status
	}

	var ownerID any
	if d.Owner != "" {
		ownerID = d.Owner
	}

	var attention []jsonAttn
	if !registered {
		attention = []jsonAttn{
			{Severity: "warning", Message: "Document is not registered in the Kanbanzai document store"},
		}
	}

	dr := docResult{
		Scope: "document",
		Document: jsonDoc{
			ID:         docID,
			Path:       d.Path,
			Type:       docType,
			Status:     docStatus,
			Registered: registered,
			OwnerID:    ownerID,
		},
		Attention: attention,
	}

	return marshalResults(dr)
}

// ─── RenderProject ────────────────────────────────────────────────────────────

func (r *JSONRenderer) RenderProject(in *render.ProjectInput) ([]byte, error) {
	plans := make([]jsonProjectPlan, 0, len(in.Plans))
	for _, p := range in.Plans {
		plans = append(plans, jsonProjectPlan{
			ID:     p.DisplayID,
			Slug:   p.DisplayID,
			Status: p.Status,
			Features: jsonFeatCnt{
				Active: p.FeaturesActive,
				Done:   p.FeaturesTotal - p.FeaturesActive,
				Total:  p.FeaturesTotal,
			},
		})
	}

	health := jsonHealth{}
	if in.Health != nil {
		health.Errors = in.Health.Errors
		health.Warnings = in.Health.Warnings
	}

	pr := projectResult{
		Scope:     "project",
		Plans:     plans,
		Health:    health,
		Attention: attnToJSON(in.Attention),
	}

	return json.Marshal(pr)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func marshalResults(v any) ([]byte, error) {
	wrapper := struct {
		Results []any `json:"results"`
	}{
		Results: []any{v},
	}
	return json.Marshal(wrapper)
}

func attnToJSON(items []render.AttentionItem) []jsonAttn {
	if items == nil {
		return []jsonAttn{}
	}
	out := make([]jsonAttn, 0, len(items))
	for _, a := range items {
		var eid any
		if a.EntityID != "" {
			eid = a.EntityID
		}
		out = append(out, jsonAttn{
			Severity: a.Severity,
			EntityID: eid,
			Message:  a.Message,
		})
	}
	return out
}
