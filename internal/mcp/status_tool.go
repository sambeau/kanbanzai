// Package mcp status_tool.go — synthesis dashboard tool for Kanbanzai 2.0 (Track D).
//
// status() without an ID returns a project overview across all plans.
// status(plan_id) returns a plan dashboard with per-feature task rollup.
// status(feature_id) returns a feature detail view with task breakdown and documents.
// status(task_id / bug_id) returns a task/bug detail with dependency state.
//
// The tool is read-only: it produces no side effects.
//
// # Response shapes
//
// Project overview (no id):
//
//	{"scope":"project","plans":[...],"total":{...},"health":{...},"attention":[...],"generated_at":"..."}
//
// Plan dashboard (plan ID):
//
//	{"scope":"plan","plan":{...},"features":[...],"doc_gaps":[...],"health":{...},"attention":[...],"generated_at":"..."}
//
// Feature detail (FEAT-...):
//
//	{"scope":"feature","feature":{...},"tasks":[...],"task_summary":{...},"documents":[...],"estimate":...,"worktree":{...},"attention":[...],"generated_at":"..."}
//
// Task detail (TASK-... or T-...):
//
//	{"scope":"task","task":{...},"parent_feature":{...},"dependencies":[...],"dispatch":{...},"attention":[...],"generated_at":"..."}
//
// Bug detail (BUG-...):
//
//	{"scope":"bug","bug":{...},"attention":[...],"generated_at":"..."}
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/id"
	"kanbanzai/internal/model"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// StatusTools returns the status MCP tool registered in the core group.
// worktreeStore may be nil; feature detail will omit the worktree field in that case.
func StatusTools(entitySvc *service.EntityService, docSvc *service.DocumentService, worktreeStore *worktree.Store) []server.ServerTool {
	return []server.ServerTool{statusTool(entitySvc, docSvc, worktreeStore)}
}

func statusTool(entitySvc *service.EntityService, docSvc *service.DocumentService, worktreeStore *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("status",
		mcp.WithDescription(
			"Synthesis dashboard. Aggregates project, plan, feature, or task state into a concise view. "+
				"Call with no id for a project overview across all plans. "+
				"Call with a plan ID (e.g. P1-my-plan) for a plan dashboard. "+
				"Call with a FEAT-... ID for a feature detail view. "+
				"Call with a TASK-..., T-..., or BUG-... ID for a task/bug detail view. "+
				"Returns attention items highlighting the most actionable next steps.",
		),
		mcp.WithString("id", mcp.Description(
			"Optional entity ID to scope the dashboard. "+
				"Omit for project overview. "+
				"Plan ID (e.g. P1-my-plan) for plan dashboard. "+
				"FEAT-... for feature detail. TASK-... or BUG-... for task/bug detail.",
		)),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		id = strings.TrimSpace(id)

		var result any
		var err error

		switch inferIDType(id) {
		case idTypeNone:
			result, err = synthesiseProject(entitySvc, docSvc)
		case idTypePlan:
			result, err = synthesisePlan(id, entitySvc, docSvc)
		case idTypeFeature:
			result, err = synthesiseFeature(id, entitySvc, docSvc, worktreeStore)
		case idTypeTask:
			result, err = synthesiseTask(id, entitySvc)
		case idTypeBug:
			result, err = synthesiseBug(id, entitySvc)
		default:
			return ActionError("unknown_id_format",
				fmt.Sprintf("Cannot determine entity type from ID %q. "+
					"Use a plan ID (e.g. P1-slug), FEAT-..., TASK-..., T-..., or BUG-...", id),
				nil), nil
		}

		if err != nil {
			if isNotFound(err) {
				return ActionError("not_found",
					fmt.Sprintf("Entity %q not found", id), nil), nil
			}
			return ActionError("status_error", err.Error(), nil), nil
		}

		b, jsonErr := json.Marshal(result)
		if jsonErr != nil {
			return ActionError("serialisation_error", jsonErr.Error(), nil), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── ID type inference ────────────────────────────────────────────────────────

type idType int

const (
	idTypeNone    idType = iota // no ID provided
	idTypePlan                  // plan prefix + number + slug (e.g. P1-foo)
	idTypeFeature               // FEAT-...
	idTypeTask                  // TASK-... or T-...
	idTypeBug                   // BUG-...
	idTypeUnknown               // non-empty but unrecognised format
)

// inferIDType returns the entity type implied by the ID string.
func inferIDType(id string) idType {
	if id == "" {
		return idTypeNone
	}
	upper := strings.ToUpper(id)
	switch {
	case strings.HasPrefix(upper, "FEAT-"):
		return idTypeFeature
	case strings.HasPrefix(upper, "TASK-") || strings.HasPrefix(upper, "T-"):
		return idTypeTask
	case strings.HasPrefix(upper, "BUG-"):
		return idTypeBug
	case model.IsPlanID(id):
		return idTypePlan
	default:
		return idTypeUnknown
	}
}

func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

// ─── Shared types ─────────────────────────────────────────────────────────────

// statusHealthSummary is the compact health block included in project and plan views.
// It summarises errors and warnings without full category detail, keeping the response
// compact for agents that only need a quick signal.
type statusHealthSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// buildHealthSummary runs the entity health check and returns a compact summary.
// On error it returns an empty summary rather than failing the whole status call,
// because health-check failure should not block the dashboard.
func buildHealthSummary(entitySvc *service.EntityService) *statusHealthSummary {
	report, err := entitySvc.HealthCheck()
	if err != nil {
		return &statusHealthSummary{}
	}
	return &statusHealthSummary{
		Errors:   report.Summary.ErrorCount,
		Warnings: report.Summary.WarningCount,
	}
}

// worktreeInfo is the compact worktree block included in feature detail.
type worktreeInfo struct {
	Status string `json:"status"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// dispatchInfo is the dispatch block included in task detail for active tasks.
type dispatchInfo struct {
	DispatchedTo string `json:"dispatched_to,omitempty"`
	DispatchedAt string `json:"dispatched_at,omitempty"`
	DispatchedBy string `json:"dispatched_by,omitempty"`
}

// featureInfo is a compact feature record used in feature detail and task parent context.
// plan_id holds the parent plan ID (e.g. "P1-my-plan").
type featureInfo struct {
	DisplayID string `json:"display_id"`
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status"`
	PlanID    string `json:"plan_id,omitempty"`
}

// ─── Project overview synthesis ───────────────────────────────────────────────

type projectOverview struct {
	Scope       string               `json:"scope"`
	Plans       []planSummary        `json:"plans"`
	Total       planAggregate        `json:"total"`
	Health      *statusHealthSummary `json:"health,omitempty"`
	Attention   []string             `json:"attention,omitempty"`
	GeneratedAt string               `json:"generated_at"`
}

type planAggregate struct {
	Plans    int `json:"plans"`
	Features int `json:"features"`
	Tasks    struct {
		Ready  int `json:"ready"`
		Active int `json:"active"`
		Done   int `json:"done"`
		Total  int `json:"total"`
	} `json:"tasks"`
}

type planSummary struct {
	DisplayID string `json:"display_id"`
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status"`
	Features  int    `json:"features"`
	Tasks     struct {
		Ready  int `json:"ready"`
		Active int `json:"active"`
		Done   int `json:"done"`
		Total  int `json:"total"`
	} `json:"tasks"`
}

func synthesiseProject(entitySvc *service.EntityService, docSvc *service.DocumentService) (*projectOverview, error) {
	plans, err := entitySvc.ListPlans(service.PlanFilters{})
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}

	allFeatures, err := entitySvc.List("feature")
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	allTasks, err := entitySvc.List("task")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	// Index features by their plan (stored as "parent" field on feature records).
	featuresByPlan := make(map[string][]service.ListResult)
	for _, f := range allFeatures {
		parent, _ := f.State["parent"].(string)
		featuresByPlan[parent] = append(featuresByPlan[parent], f)
	}

	// Index task status by parent feature.
	type taskCounts struct{ ready, active, done, total int }
	tasksByFeature := make(map[string]taskCounts)
	for _, t := range allTasks {
		pf, _ := t.State["parent_feature"].(string)
		status, _ := t.State["status"].(string)
		tc := tasksByFeature[pf]
		tc.total++
		switch status {
		case "ready":
			tc.ready++
		case "active":
			tc.active++
		case "done":
			tc.done++
		}
		tasksByFeature[pf] = tc
	}

	summaries := make([]planSummary, 0, len(plans))
	var agg planAggregate
	agg.Plans = len(plans)

	for _, p := range plans {
		status, _ := p.State["status"].(string)
		title, _ := p.State["title"].(string)

		features := featuresByPlan[p.ID]
		agg.Features += len(features)

		var planTasks taskCounts
		for _, f := range features {
			tc := tasksByFeature[f.ID]
			planTasks.ready += tc.ready
			planTasks.active += tc.active
			planTasks.done += tc.done
			planTasks.total += tc.total
		}
		agg.Tasks.Ready += planTasks.ready
		agg.Tasks.Active += planTasks.active
		agg.Tasks.Done += planTasks.done
		agg.Tasks.Total += planTasks.total

		ps := planSummary{
			DisplayID: id.FormatFullDisplay(p.ID),
			ID:        p.ID,
			Slug:      p.Slug,
			Title:     title,
			Status:    status,
			Features:  len(features),
		}
		ps.Tasks.Ready = planTasks.ready
		ps.Tasks.Active = planTasks.active
		ps.Tasks.Done = planTasks.done
		ps.Tasks.Total = planTasks.total
		summaries = append(summaries, ps)
	}

	attention := generateProjectAttention(summaries, allTasks)
	health := buildHealthSummary(entitySvc)

	return &projectOverview{
		Scope:       "project",
		Plans:       summaries,
		Total:       agg,
		Health:      health,
		Attention:   attention,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ─── Plan dashboard synthesis ─────────────────────────────────────────────────

type planDashboard struct {
	Scope       string               `json:"scope"`
	Plan        planHeader           `json:"plan"`
	Features    []featureSummary     `json:"features"`
	HasLabels   bool                 `json:"has_labels,omitempty"`
	DocGaps     []string             `json:"doc_gaps,omitempty"`
	Health      *statusHealthSummary `json:"health,omitempty"`
	Attention   []string             `json:"attention,omitempty"`
	GeneratedAt string               `json:"generated_at"`
}

type planHeader struct {
	DisplayID string `json:"display_id"`
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status"`
}

type featureSummary struct {
	DisplayID string `json:"display_id"`
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status"`
	Label     string `json:"label,omitempty"`
	Tasks     struct {
		Queued int `json:"queued"`
		Ready  int `json:"ready"`
		Active int `json:"active"`
		Done   int `json:"done"`
		Total  int `json:"total"`
	} `json:"tasks"`
	HasSpec    bool `json:"has_spec"`
	HasDevPlan bool `json:"has_dev_plan"`
}

func synthesisePlan(planID string, entitySvc *service.EntityService, docSvc *service.DocumentService) (*planDashboard, error) {
	plan, err := entitySvc.GetPlan(planID)
	if err != nil {
		return nil, err
	}

	allFeatures, err := entitySvc.List("feature")
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	allTasks, err := entitySvc.List("task")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	// Filter features owned by this plan (stored as "parent" field on feature records).
	var features []service.ListResult
	for _, f := range allFeatures {
		parent, _ := f.State["parent"].(string)
		if parent == planID {
			features = append(features, f)
		}
	}

	// Build task status index.
	type taskCounts struct{ queued, ready, active, done, total int }
	tasksByFeature := make(map[string]taskCounts)
	for _, t := range allTasks {
		pf, _ := t.State["parent_feature"].(string)
		status, _ := t.State["status"].(string)
		tc := tasksByFeature[pf]
		tc.total++
		switch status {
		case "queued":
			tc.queued++
		case "ready":
			tc.ready++
		case "active":
			tc.active++
		case "done":
			tc.done++
		}
		tasksByFeature[pf] = tc
	}

	// Collect document info per feature.
	docsPerFeature := make(map[string][]service.DocumentResult)
	if docSvc != nil {
		for _, f := range features {
			docs, _ := docSvc.ListDocumentsByOwner(f.ID)
			docsPerFeature[f.ID] = docs
		}
	}

	var docGaps []string
	featureSummaries := make([]featureSummary, 0, len(features))

	for _, f := range features {
		fstatus, _ := f.State["status"].(string)
		fsummary, _ := f.State["summary"].(string)
		tc := tasksByFeature[f.ID]

		docs := docsPerFeature[f.ID]
		hasSpec := hasDocType(docs, "specification")
		hasDevPlan := hasDocType(docs, "dev-plan")

		flabel, _ := f.State["label"].(string)

		fs := featureSummary{
			DisplayID:  id.FormatFullDisplay(f.ID),
			ID:         f.ID,
			Slug:       f.Slug,
			Summary:    fsummary,
			Status:     fstatus,
			Label:      flabel,
			HasSpec:    hasSpec,
			HasDevPlan: hasDevPlan,
		}
		fs.Tasks.Queued = tc.queued
		fs.Tasks.Ready = tc.ready
		fs.Tasks.Active = tc.active
		fs.Tasks.Done = tc.done
		fs.Tasks.Total = tc.total

		featureSummaries = append(featureSummaries, fs)

		// Document gap detection.
		if !hasSpec {
			docGaps = append(docGaps, fmt.Sprintf("%s (%s): missing specification", f.ID, f.Slug))
		}
	}

	title, _ := plan.State["title"].(string)
	planStatus, _ := plan.State["status"].(string)

	// Detect whether any feature has a label set.
	hasLabels := false
	for _, fs := range featureSummaries {
		if fs.Label != "" {
			hasLabels = true
			break
		}
	}

	attention := generatePlanAttention(featureSummaries, docGaps)
	health := buildHealthSummary(entitySvc)

	return &planDashboard{
		Scope: "plan",
		Plan: planHeader{
			DisplayID: id.FormatFullDisplay(plan.ID),
			ID:        plan.ID,
			Slug:      plan.Slug,
			Title:     title,
			Status:    planStatus,
		},
		Features:    featureSummaries,
		HasLabels:   hasLabels,
		DocGaps:     docGaps,
		Health:      health,
		Attention:   attention,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ─── Feature detail synthesis ─────────────────────────────────────────────────

type featureDetail struct {
	Scope       string      `json:"scope"`
	Feature     featureInfo `json:"feature"`
	Tasks       []taskInfo  `json:"tasks"`
	HasLabels   bool        `json:"has_labels,omitempty"`
	TaskSummary struct {
		Queued int `json:"queued"`
		Ready  int `json:"ready"`
		Active int `json:"active"`
		Done   int `json:"done"`
		Total  int `json:"total"`
	} `json:"task_summary"`
	Documents   []docInfo     `json:"documents,omitempty"`
	Estimate    *float64      `json:"estimate,omitempty"`
	Worktree    *worktreeInfo `json:"worktree,omitempty"`
	Attention   []string      `json:"attention,omitempty"`
	GeneratedAt string        `json:"generated_at"`
}

type taskInfo struct {
	DisplayID string   `json:"display_id"`
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Summary   string   `json:"summary,omitempty"`
	Status    string   `json:"status"`
	Label     string   `json:"label,omitempty"`
	Estimate  *float64 `json:"estimate,omitempty"`
}

type docInfo struct {
	DisplayID string `json:"display_id"`
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status"`
	Path      string `json:"path,omitempty"`
}

func synthesiseFeature(featID string, entitySvc *service.EntityService, docSvc *service.DocumentService, worktreeStore *worktree.Store) (*featureDetail, error) {
	feat, err := entitySvc.Get("feature", featID, "")
	if err != nil {
		return nil, err
	}

	allTasks, err := entitySvc.List("task")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	// Filter tasks for this feature.
	var tasks []taskInfo
	var taskSummary struct{ queued, ready, active, done, total int }
	for _, t := range allTasks {
		pf, _ := t.State["parent_feature"].(string)
		if pf != featID {
			continue
		}
		status, _ := t.State["status"].(string)
		summary, _ := t.State["summary"].(string)
		taskSummary.total++
		switch status {
		case "queued":
			taskSummary.queued++
		case "ready":
			taskSummary.ready++
		case "active":
			taskSummary.active++
		case "done":
			taskSummary.done++
		}
		var est *float64
		if ev, ok := t.State["estimate"]; ok && ev != nil {
			if ef, ok := ev.(float64); ok {
				est = &ef
			}
		}
		tlabel, _ := t.State["label"].(string)
		tasks = append(tasks, taskInfo{
			DisplayID: id.FormatFullDisplay(t.ID),
			ID:        t.ID,
			Slug:      t.Slug,
			Summary:   summary,
			Status:    status,
			Label:     tlabel,
			Estimate:  est,
		})
	}

	// Load documents for this feature.
	var docs []docInfo
	if docSvc != nil {
		ownerDocs, _ := docSvc.ListDocumentsByOwner(featID)
		for _, d := range ownerDocs {
			docs = append(docs, docInfo{
				DisplayID: id.FormatFullDisplay(d.ID),
				ID:        d.ID,
				Type:      d.Type,
				Title:     d.Title,
				Status:    d.Status,
				Path:      d.Path,
			})
		}
	}

	// Look up worktree for this feature, if a store is provided.
	var wt *worktreeInfo
	if worktreeStore != nil {
		if record, err := worktreeStore.GetByEntityID(featID); err == nil {
			wt = &worktreeInfo{
				Status: string(record.Status),
				Branch: record.Branch,
				Path:   record.Path,
			}
		}
	}

	fstatus, _ := feat.State["status"].(string)
	fsummary, _ := feat.State["summary"].(string)
	fplanID, _ := feat.State["parent"].(string) // "parent" is the plan ID field on feature records

	var est *float64
	if ev, ok := feat.State["estimate"]; ok && ev != nil {
		if ef, ok := ev.(float64); ok {
			est = &ef
		}
	}

	// Detect whether any task has a label set.
	hasLabels := false
	for _, ti := range tasks {
		if ti.Label != "" {
			hasLabels = true
			break
		}
	}

	attention := generateFeatureAttention(tasks, docs, taskSummary.total)

	d := &featureDetail{
		Scope: "feature",
		Feature: featureInfo{
			DisplayID: id.FormatFullDisplay(feat.ID),
			ID:        feat.ID,
			Slug:      feat.Slug,
			Summary:   fsummary,
			Status:    fstatus,
			PlanID:    fplanID,
		},
		Tasks:       tasks,
		HasLabels:   hasLabels,
		Documents:   docs,
		Estimate:    est,
		Worktree:    wt,
		Attention:   attention,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	d.TaskSummary.Queued = taskSummary.queued
	d.TaskSummary.Ready = taskSummary.ready
	d.TaskSummary.Active = taskSummary.active
	d.TaskSummary.Done = taskSummary.done
	d.TaskSummary.Total = taskSummary.total

	return d, nil
}

// ─── Task detail synthesis ────────────────────────────────────────────────────

type taskDetail struct {
	Scope         string        `json:"scope"`
	Task          taskFullInfo  `json:"task"`
	ParentFeature *featureInfo  `json:"parent_feature,omitempty"`
	Dependencies  []depInfo     `json:"dependencies,omitempty"`
	Dispatch      *dispatchInfo `json:"dispatch,omitempty"`
	Attention     []string      `json:"attention,omitempty"`
	GeneratedAt   string        `json:"generated_at"`
}

type taskFullInfo struct {
	DisplayID     string   `json:"display_id"`
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Summary       string   `json:"summary,omitempty"`
	Status        string   `json:"status"`
	ParentFeature string   `json:"parent_feature,omitempty"`
	Estimate      *float64 `json:"estimate,omitempty"`
	FilesPlanned  []string `json:"files_planned,omitempty"`
}

type depInfo struct {
	DisplayID string `json:"display_id"`
	TaskID    string `json:"task_id"`
	Slug      string `json:"slug,omitempty"`
	Status    string `json:"status"`
	Blocking  bool   `json:"blocking"`
}

func synthesiseTask(taskID string, entitySvc *service.EntityService) (*taskDetail, error) {
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return nil, err
	}

	tstatus, _ := task.State["status"].(string)
	tsummary, _ := task.State["summary"].(string)
	tparent, _ := task.State["parent_feature"].(string)

	var est *float64
	if ev, ok := task.State["estimate"]; ok && ev != nil {
		if ef, ok := ev.(float64); ok {
			est = &ef
		}
	}

	var filesPlanned []string
	if fp, ok := task.State["files_planned"]; ok {
		if fpArr, ok := fp.([]any); ok {
			for _, f := range fpArr {
				if s, ok := f.(string); ok {
					filesPlanned = append(filesPlanned, s)
				}
			}
		}
	}

	ti := taskFullInfo{
		DisplayID:     id.FormatFullDisplay(task.ID),
		ID:            task.ID,
		Slug:          task.Slug,
		Summary:       tsummary,
		Status:        tstatus,
		ParentFeature: tparent,
		Estimate:      est,
		FilesPlanned:  filesPlanned,
	}

	// Load parent feature for context.
	var parentFeat *featureInfo
	if tparent != "" {
		if pf, err := entitySvc.Get("feature", tparent, ""); err == nil {
			pfstatus, _ := pf.State["status"].(string)
			pfsummary, _ := pf.State["summary"].(string)
			// Features store their parent plan in the "parent" field, not "owner".
			pfplanID, _ := pf.State["parent"].(string)
			parentFeat = &featureInfo{
				DisplayID: id.FormatFullDisplay(pf.ID),
				ID:        pf.ID,
				Slug:      pf.Slug,
				Summary:   pfsummary,
				Status:    pfstatus,
				PlanID:    pfplanID,
			}
		}
	}

	// Resolve dependencies.
	deps := resolveDependencies(task.State, entitySvc)

	// Build dispatch info if the task has been dispatched.
	var dispatch *dispatchInfo
	if to, _ := task.State["dispatched_to"].(string); to != "" {
		dispatch = &dispatchInfo{
			DispatchedTo: to,
			DispatchedAt: stringFromTaskState(task.State, "dispatched_at"),
			DispatchedBy: stringFromTaskState(task.State, "dispatched_by"),
		}
	}

	attention := generateTaskAttention(ti, deps)

	return &taskDetail{
		Scope:         "task",
		Task:          ti,
		ParentFeature: parentFeat,
		Dependencies:  deps,
		Dispatch:      dispatch,
		Attention:     attention,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// stringFromTaskState is a small helper to safely read a string field from task state.
func stringFromTaskState(state map[string]any, key string) string {
	v, _ := state[key].(string)
	return v
}

// ─── Bug detail synthesis ─────────────────────────────────────────────────────

type bugDetail struct {
	Scope       string   `json:"scope"`
	Bug         bugInfo  `json:"bug"`
	Attention   []string `json:"attention,omitempty"`
	GeneratedAt string   `json:"generated_at"`
}

type bugInfo struct {
	DisplayID string `json:"display_id"`
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title,omitempty"`
	Status    string `json:"status"`
	Severity  string `json:"severity,omitempty"`
	Priority  string `json:"priority,omitempty"`
}

func synthesiseBug(bugID string, entitySvc *service.EntityService) (*bugDetail, error) {
	bug, err := entitySvc.Get("bug", bugID, "")
	if err != nil {
		return nil, err
	}

	bstatus, _ := bug.State["status"].(string)
	btitle, _ := bug.State["title"].(string)
	bseverity, _ := bug.State["severity"].(string)
	bpriority, _ := bug.State["priority"].(string)

	var attention []string
	if bseverity == "critical" || bseverity == "high" {
		attention = append(attention, fmt.Sprintf("High-severity bug (%s) — prioritise resolution", bseverity))
	}
	if bstatus == "reported" {
		attention = append(attention, "Bug not yet triaged — run triage to confirm severity and assign")
	}

	return &bugDetail{
		Scope: "bug",
		Bug: bugInfo{
			DisplayID: id.FormatFullDisplay(bug.ID),
			ID:        bug.ID,
			Slug:      bug.Slug,
			Title:     btitle,
			Status:    bstatus,
			Severity:  bseverity,
			Priority:  bpriority,
		},
		Attention:   attention,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ─── Attention item generators ────────────────────────────────────────────────

const maxAttentionItems = 5

func generateProjectAttention(plans []planSummary, allTasks []service.ListResult) []string {
	var items []string

	// Count active plans.
	activePlans := 0
	for _, p := range plans {
		if p.Status == "active" {
			activePlans++
		}
	}

	// Find plans with ready tasks.
	for _, p := range plans {
		if p.Tasks.Ready > 0 {
			items = append(items, fmt.Sprintf("%d task(s) ready to claim in plan %s", p.Tasks.Ready, p.DisplayID))
		}
		if len(items) >= maxAttentionItems {
			break
		}
	}

	// Find stalled active tasks (no update in >3 days).
	stalledCount := 0
	threshold := time.Now().UTC().Add(-3 * 24 * time.Hour)
	for _, t := range allTasks {
		status, _ := t.State["status"].(string)
		if status != "active" {
			continue
		}
		if updatedStr, ok := t.State["updated"].(string); ok {
			if updated, err := time.Parse(time.RFC3339, updatedStr); err == nil {
				if updated.Before(threshold) {
					stalledCount++
				}
			}
		}
	}
	if stalledCount > 0 && len(items) < maxAttentionItems {
		items = append(items, fmt.Sprintf("%d active task(s) stalled for >3 days — check progress", stalledCount))
	}

	if activePlans == 0 && len(plans) > 0 && len(items) < maxAttentionItems {
		items = append(items, "No active plans — advance a plan from designing or proposed to active")
	}

	return items
}

func generatePlanAttention(features []featureSummary, docGaps []string) []string {
	var items []string

	// Features with ready tasks.
	for _, f := range features {
		if f.Tasks.Ready > 0 && len(items) < maxAttentionItems {
			items = append(items, fmt.Sprintf("%d task(s) ready in %s (%s)", f.Tasks.Ready, f.DisplayID, f.Slug))
		}
	}

	// Missing specs.
	for _, gap := range docGaps {
		if len(items) >= maxAttentionItems {
			break
		}
		items = append(items, gap)
	}

	// Features with no tasks.
	for _, f := range features {
		if f.Tasks.Total == 0 && f.Status != "done" && len(items) < maxAttentionItems {
			items = append(items, fmt.Sprintf("%s has no tasks — decompose the feature to start work", f.DisplayID))
		}
	}

	return items
}

func generateFeatureAttention(tasks []taskInfo, docs []docInfo, totalTasks int) []string {
	var items []string

	// Ready tasks available.
	readyCount := 0
	for _, t := range tasks {
		if t.Status == "ready" {
			readyCount++
		}
	}
	if readyCount > 0 {
		items = append(items, fmt.Sprintf("%d task(s) ready to claim", readyCount))
	}

	// Missing spec.
	hasSpec := false
	hasDevPlan := false
	for _, d := range docs {
		switch d.Type {
		case "specification":
			hasSpec = true
		case "dev-plan":
			hasDevPlan = true
		}
	}
	if !hasSpec && len(items) < maxAttentionItems {
		items = append(items, "Missing specification document — create work/spec/*.md and register it")
	}
	if !hasDevPlan && len(items) < maxAttentionItems {
		items = append(items, "Missing dev-plan document — create work/plan/*.md and register it")
	}

	// No tasks.
	if totalTasks == 0 && len(items) < maxAttentionItems {
		items = append(items, "No tasks exist — run decompose to generate the task breakdown")
	}

	return items
}

func generateTaskAttention(task taskFullInfo, deps []depInfo) []string {
	var items []string

	// Blocking dependencies.
	blockingCount := 0
	for _, d := range deps {
		if d.Blocking {
			blockingCount++
		}
	}
	if blockingCount > 0 {
		items = append(items, fmt.Sprintf("%d blocking dependency(ies) not yet done — task cannot start", blockingCount))
	}

	// Task is ready.
	if task.Status == "ready" {
		items = append(items, "Task is ready — use next(task_id) to claim and receive full context")
	}

	// Missing files_planned.
	if len(task.FilesPlanned) == 0 && task.Status != "done" {
		items = append(items, "No files_planned set — add planned files for better conflict detection")
	}

	return items
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// hasDocType reports whether the slice contains a document of the given type
// with a non-superseded status.
func hasDocType(docs []service.DocumentResult, docType string) bool {
	for _, d := range docs {
		if d.Type == docType && d.Status != "superseded" {
			return true
		}
	}
	return false
}

// resolveDependencies looks up each depends_on entry and checks its current status.
func resolveDependencies(taskState map[string]any, entitySvc *service.EntityService) []depInfo {
	rawDeps, ok := taskState["depends_on"]
	if !ok {
		return nil
	}
	depIDs, ok := rawDeps.([]any)
	if !ok {
		return nil
	}

	var result []depInfo
	for _, d := range depIDs {
		depID, _ := d.(string)
		if depID == "" {
			continue
		}
		dep, err := entitySvc.Get("task", depID, "")
		if err != nil {
			result = append(result, depInfo{DisplayID: id.FormatFullDisplay(depID), TaskID: depID, Status: "unknown", Blocking: true})
			continue
		}
		depStatus, _ := dep.State["status"].(string)
		blocking := !isTerminalStatus(depStatus)
		result = append(result, depInfo{
			DisplayID: id.FormatFullDisplay(dep.ID),
			TaskID:    dep.ID,
			Slug:      dep.Slug,
			Status:    depStatus,
			Blocking:  blocking,
		})
	}
	return result
}

// isTerminalStatus returns true for statuses that satisfy a dependency.
func isTerminalStatus(status string) bool {
	switch status {
	case "done", "not-planned", "duplicate":
		return true
	}
	return false
}
