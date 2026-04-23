// Package mcp entity_tool.go — consolidated entity CRUD for Kanbanzai 2.0 (Track H).
//
// entity(action, ...) replaces 17+ entity-specific 1.0 tools with a single
// resource-oriented interface:
//
//	entity(action: "create", type: "task", parent_feature: "FEAT-...", slug: "...", summary: "...")
//	entity(action: "get", id: "FEAT-...")
//	entity(action: "list", type: "feature", parent: "P1-...", status: "developing")
//	entity(action: "update", id: "TASK-...", summary: "Updated summary")
//	entity(action: "transition", id: "FEAT-...", status: "developing")
//
// Type is inferred from the ID prefix for get/update/transition. Side effects
// are reported for status transitions (worktree creation, dependency unblocking).
package mcp

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/checkpoint"
	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/gate"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/knowledge"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// entityCommitFunc is the function called by entity create and transition
// handlers to commit state changes atomically. Package-level variable for
// test injection. Production value delegates to git.CommitStateWithMessage
// (FR-A10, FR-A11).
var entityCommitFunc = func(repoRoot, message string) (bool, error) {
	return git.CommitStateWithMessage(repoRoot, message)
}

// EntityTool returns the consolidated `entity` MCP tool registered in the core group.
func EntityTool(entitySvc *service.EntityService, docSvc *service.DocumentService, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) []server.ServerTool {
	return []server.ServerTool{entityTool(entitySvc, docSvc, gateRouter, checkpointStore, requiresHumanReview)}
}

func entityTool(entitySvc *service.EntityService, docSvc *service.DocumentService, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) server.ServerTool {
	tool := mcp.NewTool("entity",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Entity Manager"),
		mcp.WithDescription(
			"The primary tool for managing workflow entities (plans, features, tasks, bugs, decisions) — "+
				"use this whenever you need to create, query, modify, or advance entities through their lifecycle. "+
				"Use INSTEAD OF reading .kbz/state/ YAML files directly — action: get and action: list return "+
				"structured data with lifecycle state and cross-references that raw files do not provide. "+
				"For synthesised dashboard views (progress, attention items, what's blocked), use status instead. "+
				"Actions: create, get, list, update, transition. "+
				"For create and list: type is required. "+
				"For get, update, and transition: id is required (type is inferred from the ID prefix). "+
				"Supports batch creation via the entities array.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: create, get, list, update, transition"),
		),
		mcp.WithString("type", mcp.Description("Entity type: plan, feature, task, bug, decision (required for create and list)")),
		mcp.WithString("id", mcp.Description("Entity ID — type inferred from prefix (required for get, update, transition)")),
		mcp.WithString("status", mcp.Description("Target status (transition) or status filter (list)")),
		mcp.WithString("parent", mcp.Description(
			"Parent plan ID for features (required on feature create to associate the feature "+
				"with its plan; also used as a filter on list). "+
				"Note: tasks use parent_feature, not parent.")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Tag filter (list) or tags to set (create/update)")),
		mcp.WithArray("entities", mcp.Items(map[string]any{"type": "object"}), mcp.Description("Batch create: array of entity objects, each with the same fields as single create")),
		// Common entity fields (type-specific, all optional at top level).
		mcp.WithString("slug", mcp.Description("URL-friendly identifier")),
		mcp.WithString("summary", mcp.Description("Brief summary")),
		mcp.WithString("name", mcp.Description("Human-readable display name (required on create, ~4 words, no colon, no phase prefix).")),
		mcp.WithString("prefix", mcp.Description("Single-character Plan ID prefix (plan create only)")),
		mcp.WithString("parent_feature", mcp.Description("Parent feature ID (task create only)")),
		mcp.WithString("rationale", mcp.Description("Decision rationale (decision create only)")),
		mcp.WithString("reported_by", mcp.Description("Who reported it (bug create only)")),
		mcp.WithString("observed", mcp.Description("Observed behavior (bug create only)")),
		mcp.WithString("expected", mcp.Description("Expected behavior (bug create only)")),
		mcp.WithString("severity", mcp.Description("Bug severity: low, medium, high, critical")),
		mcp.WithString("priority", mcp.Description("Bug priority: low, medium, high, critical")),
		mcp.WithString("bug_type", mcp.Description("Bug type: implementation-defect, specification-defect, design-problem")),
		mcp.WithString("created_by", mcp.Description("Who created it. Auto-resolved from .kbz/local.yaml or git config if not provided.")),
		mcp.WithString("design", mcp.Description("Design document reference (feature or plan)")),
		mcp.WithArray("depends_on", mcp.WithStringItems(), mcp.Description("Task IDs this task depends on (task update only). Each must be a valid TASK-... ID.")),
		mcp.WithString("created_after", mcp.Description("Created-after filter, RFC3339 (list only)")),
		mcp.WithString("created_before", mcp.Description("Created-before filter, RFC3339 (list only)")),
		mcp.WithBoolean("advance", mcp.Description(
			"When true, advance a feature through multiple lifecycle states toward the target, "+
				"checking document prerequisites at each gate. Only supported for feature entities (transition only, default: false).",
		)),
		mcp.WithBoolean("override", mcp.Description(
			"When true, bypass a failing stage gate prerequisite. Requires override_reason to be set. "+
				"The bypass is logged as an override record on the feature entity (FR-011, FR-017).",
		)),
		mcp.WithString("override_reason", mcp.Description(
			"Required when override is true. Explains why the gate prerequisite is being bypassed. "+
				"Stored permanently on the feature as an override record (FR-012, FR-014).",
		)),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"create":     entityCreateAction(entitySvc),
			"get":        entityGetAction(entitySvc),
			"list":       entityListAction(entitySvc),
			"update":     entityUpdateAction(entitySvc),
			"transition": entityTransitionAction(entitySvc, docSvc, gateRouter, checkpointStore, requiresHumanReview),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── create ──────────────────────────────────────────────────────────────────

func entityCreateAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)

		entityType := strings.ToLower(entityArgStr(args, "type"))
		if entityType == "" {
			return nil, fmt.Errorf("Cannot create entity: type is missing.\n\nTo resolve:\n  Provide the entity type: entity(action: \"create\", type: \"task|feature|plan|bug|decision\", ...)")
		}

		// Signal mutation so side_effects: [] is always present in both
		// single and batch responses (spec §8.4: "The field is never omitted").
		SignalMutation(ctx)

		// Batch mode: entities array provided.
		if IsBatchInput(args, "entities") {
			items, _ := args["entities"].([]any)
			batchResult, batchErr := ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				m, ok := item.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("Cannot create entity in batch: each item in the entities array must be a JSON object with the entity fields.\n\nTo resolve:\n  Ensure entities is an array of objects: [{slug: \"...\", summary: \"...\"}, ...]")
				}
				result, err := entityCreateOne(entityType, m, entitySvc)
				return entityArgStr(m, "slug"), result, err
			})
			// Best-effort commit after all batch entities are created (FR-A10).
			batchCommitMsg := fmt.Sprintf("workflow: create %d %s entities", len(items), entityType)
			if _, commitErr := entityCommitFunc(".", batchCommitMsg); commitErr != nil {
				log.Printf("[entity] WARNING: batch auto-commit after create failed: %v", commitErr)
			}
			return batchResult, batchErr
		}

		// Single mode.
		return entityCreateOne(entityType, args, entitySvc)
	}
}

func entityCreateOne(entityType string, args map[string]any, entitySvc *service.EntityService) (any, error) {
	createdByRaw := entityArgStr(args, "created_by")
	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return nil, fmt.Errorf("Cannot create %s: failed to resolve identity.\n\nTo resolve:\n  Set created_by explicitly, or configure identity in .kbz/local.yaml", entityType)
	}

	// Advisory duplicate check runs before creation so it checks pre-existing entities only.
	advisory := entityDuplicateAdvisory(entityType, args, entitySvc)

	var result service.CreateResult

	nameRaw := entityArgStr(args, "name")
	name, nameErr := validate.ValidateName(nameRaw)
	if nameErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid name: %s", nameErr)), nil
	}

	switch entityType {
	case "task":
		result, err = entitySvc.CreateTask(service.CreateTaskInput{
			ParentFeature: entityArgStr(args, "parent_feature"),
			Slug:          entityArgStr(args, "slug"),
			Summary:       entityArgStr(args, "summary"),
			Name:          name,
		})

	case "feature":
		result, err = entitySvc.CreateFeature(service.CreateFeatureInput{
			Slug:      entityArgStr(args, "slug"),
			Parent:    entityArgStr(args, "parent"),
			Summary:   entityArgStr(args, "summary"),
			Design:    entityArgStr(args, "design"),
			Tags:      entityArgStringSlice(args, "tags"),
			CreatedBy: createdBy,
			Name:      name,
		})

	case "plan":
		result, err = entitySvc.CreatePlan(service.CreatePlanInput{
			Prefix:    entityArgStr(args, "prefix"),
			Slug:      entityArgStr(args, "slug"),
			Name:      name,
			Summary:   entityArgStr(args, "summary"),
			Tags:      entityArgStringSlice(args, "tags"),
			CreatedBy: createdBy,
		})

	case "bug":
		result, err = entitySvc.CreateBug(service.CreateBugInput{
			Slug:       entityArgStr(args, "slug"),
			Name:       name,
			ReportedBy: entityArgStr(args, "reported_by"),
			Observed:   entityArgStr(args, "observed"),
			Expected:   entityArgStr(args, "expected"),
			Severity:   entityArgStr(args, "severity"),
			Priority:   entityArgStr(args, "priority"),
			Type:       entityArgStr(args, "bug_type"),
		})

	case "decision":
		result, err = entitySvc.CreateDecision(service.CreateDecisionInput{
			Slug:      entityArgStr(args, "slug"),
			Name:      name,
			Summary:   entityArgStr(args, "summary"),
			Rationale: entityArgStr(args, "rationale"),
			DecidedBy: createdBy,
		})

	default:
		return nil, fmt.Errorf("Cannot create entity: unknown type %q.\n\nTo resolve:\n  Use one of: plan, feature, task, bug, decision", entityType)
	}

	if err != nil {
		return nil, err
	}

	displayID := id.FormatFullDisplay(result.ID)
	entityName := entityStateStr(result.State, "name")
	entityOut := map[string]any{
		"display_id": displayID,
		"id":         result.ID,
		"type":       result.Type,
		"slug":       result.Slug,
		"name":       entityName,
		"status":     entityStateStr(result.State, "status"),
		"entity_ref": id.FormatEntityRef(displayID, result.Slug, entityName),
	}
	out := map[string]any{
		"entity": entityOut,
	}

	if len(advisory) > 0 {
		out["duplicate_advisory"] = advisory
	}

	// Auto-commit the new entity's state file (FR-A10). Best-effort.
	commitMsg := fmt.Sprintf("workflow(%s): create %s", result.ID, result.Type)
	if _, commitErr := entityCommitFunc(".", commitMsg); commitErr != nil {
		log.Printf("[entity] WARNING: auto-commit after create %s failed: %v", result.ID, commitErr)
	}

	return out, nil
}

// entityDuplicateAdvisory runs an advisory (non-blocking) similarity check against
// pre-existing entities. Returns nil when no duplicates are found or the check fails.
func entityDuplicateAdvisory(entityType string, args map[string]any, entitySvc *service.EntityService) []map[string]any {
	title := entityArgStr(args, "name")
	if title == "" {
		title = entityArgStr(args, "slug")
	}
	summary := entityArgStr(args, "summary")
	if title == "" && summary == "" {
		return nil
	}

	var existing []knowledge.ExistingEntity

	if entityType == "plan" {
		plans, err := entitySvc.ListPlans(service.PlanFilters{})
		if err == nil {
			for _, p := range plans {
				t, _ := p.State["name"].(string)
				s, _ := p.State["summary"].(string)
				existing = append(existing, knowledge.ExistingEntity{
					ID:      p.ID,
					Type:    "plan",
					Title:   t,
					Summary: s,
				})
			}
		}
	} else {
		results, err := entitySvc.List(entityType)
		if err == nil {
			for _, r := range results {
				t, _ := r.State["name"].(string)
				if t == "" {
					t, _ = r.State["slug"].(string)
				}
				s, _ := r.State["summary"].(string)
				existing = append(existing, knowledge.ExistingEntity{
					ID:      r.ID,
					Type:    entityType,
					Title:   t,
					Summary: s,
				})
			}
		}
	}

	candidates := knowledge.FindDuplicateCandidates(title, summary, existing, 0.5)
	if len(candidates) == 0 {
		return nil
	}

	out := make([]map[string]any, len(candidates))
	for i, c := range candidates {
		out[i] = map[string]any{
			"entity_id":   c.EntityID,
			"entity_type": c.EntityType,
			"title":       c.Title,
			"similarity":  c.Similarity,
		}
	}
	return out
}

// ─── get ─────────────────────────────────────────────────────────────────────

func entityGetAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		entityID := id.NormalizeID(entityArgStr(args, "id"))
		if entityID == "" {
			return nil, fmt.Errorf("Cannot get entity: no ID provided.\n\nTo resolve:\n  Pass id with a prefixed entity ID (e.g. FEAT-001, TASK-042, BUG-003, P1-my-plan).")
		}

		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("Cannot get entity %q: unrecognised ID format.\n\nTo resolve:\n  Use a prefixed ID such as FEAT-..., TASK-..., T-..., BUG-..., or a plan ID like P1-slug.", entityID)
		}

		if entityType == "plan" {
			result, err := entitySvc.GetPlan(entityID)
			if err != nil {
				return nil, fmt.Errorf("Cannot get plan %s: %w.\n\nTo resolve:\n  Verify the plan ID exists with entity(action: \"list\", type: \"plan\").", entityID, err)
			}
			return map[string]any{
				"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
			}, nil
		}

		result, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			return nil, fmt.Errorf("Cannot get %s %s: %w.\n\nTo resolve:\n  Verify the ID exists with entity(action: \"list\", type: %q).", entityType, entityID, err, entityType)
		}
		return map[string]any{
			"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
		}, nil
	}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func entityListAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		entityType := strings.ToLower(entityArgStr(args, "type"))
		if entityType == "" {
			return nil, fmt.Errorf("Cannot list entities: no type provided.\n\nTo resolve:\n  Pass type with one of: plan, feature, task, bug, decision.")
		}

		statusFilter := entityArgStr(args, "status")
		parentFilter := entityArgStr(args, "parent")
		tagsFilter := entityArgStringSlice(args, "tags")

		var createdAfter, createdBefore *time.Time
		if caStr := entityArgStr(args, "created_after"); caStr != "" {
			t, err := time.Parse(time.RFC3339, caStr)
			if err != nil {
				return nil, fmt.Errorf("Cannot list entities: invalid created_after value.\n\nTo resolve:\n  Use RFC 3339 format, e.g. \"2024-01-15T00:00:00Z\".")
			}
			createdAfter = &t
		}
		if cbStr := entityArgStr(args, "created_before"); cbStr != "" {
			t, err := time.Parse(time.RFC3339, cbStr)
			if err != nil {
				return nil, fmt.Errorf("Cannot list entities: invalid created_before value.\n\nTo resolve:\n  Use RFC 3339 format, e.g. \"2024-12-31T23:59:59Z\".")
			}
			createdBefore = &t
		}

		// Plans have their own listing path.
		if entityType == "plan" {
			plans, err := entitySvc.ListPlans(service.PlanFilters{
				Status: statusFilter,
				Tags:   tagsFilter,
			})
			if err != nil {
				return nil, fmt.Errorf("Cannot list plans: %w.\n\nTo resolve:\n  Check project health with the health tool and verify .kbz/state/ is intact.", err)
			}
			return entityListResponse(entityType, entitySummaries(plans)), nil
		}

		// Generic path via ListEntitiesFiltered.
		results, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
			Type:          entityType,
			Status:        statusFilter,
			Parent:        parentFilter,
			Tags:          tagsFilter,
			CreatedAfter:  createdAfter,
			CreatedBefore: createdBefore,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot list %s entities: %w.\n\nTo resolve:\n  Check that %q is a valid entity type (plan, feature, task, bug, decision).", entityType, err, entityType)
		}
		return entityListResponse(entityType, entitySummaries(results)), nil
	}
}

func entityListResponse(entityType string, summaries []map[string]any) map[string]any {
	if summaries == nil {
		summaries = []map[string]any{}
	}
	return map[string]any{
		"entities": summaries,
		"total":    len(summaries),
		"type":     entityType,
	}
}

func entitySummaries(results []service.ListResult) []map[string]any {
	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		summary, _ := r.State["summary"].(string)
		entityName, _ := r.State["name"].(string)
		status, _ := r.State["status"].(string)
		displayID := id.FormatFullDisplay(r.ID)
		item := map[string]any{
			"display_id": displayID,
			"id":         r.ID,
			"type":       r.Type,
			"slug":       r.Slug,
			"name":       entityName,
			"status":     status,
			"summary":    summary,
			"entity_ref": id.FormatEntityRef(displayID, r.Slug, entityName),
		}
		out = append(out, item)
	}
	return out
}

// entityHasAnyTag returns true if the entity state contains at least one of the given tags.
func entityHasAnyTag(state map[string]any, filterTags []string) bool {
	rawTags := state["tags"]
	var entityTags []string
	switch v := rawTags.(type) {
	case []string:
		entityTags = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				entityTags = append(entityTags, s)
			}
		}
	}
	for _, ft := range filterTags {
		for _, et := range entityTags {
			if strings.EqualFold(ft, et) {
				return true
			}
		}
	}
	return false
}

// ─── update ───────────────────────────────────────────────────────────────────

func entityUpdateAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		// Signal mutation so side_effects: [] is always present in the response (spec §8.4).
		SignalMutation(ctx)

		args, _ := req.Params.Arguments.(map[string]any)
		entityID := id.NormalizeID(entityArgStr(args, "id"))
		if entityID == "" {
			return nil, fmt.Errorf("Cannot update entity: no ID provided.\n\nTo resolve:\n  Pass id with a prefixed entity ID (e.g. FEAT-001, TASK-042, BUG-003, P1-my-plan).")
		}

		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("Cannot update entity %q: unrecognised ID format.\n\nTo resolve:\n  Use a prefixed ID such as FEAT-..., TASK-..., T-..., BUG-..., or a plan ID like P1-slug.", entityID)
		}

		// Plans use their own update path (supports name, summary, design, tags).
		if entityType == "plan" {
			_, _, slug := model.ParsePlanID(entityID)
			input := service.UpdatePlanInput{
				ID:   entityID,
				Slug: slug,
			}
			if _, has := args["name"]; has {
				v := entityArgStr(args, "name")
				input.Name = &v
			}
			if _, has := args["summary"]; has {
				v := entityArgStr(args, "summary")
				input.Summary = &v
			}
			if _, has := args["design"]; has {
				v := entityArgStr(args, "design")
				input.Design = &v
			}
			if _, has := args["tags"]; has {
				input.Tags = entityArgStringSlice(args, "tags")
			}
			result, err := entitySvc.UpdatePlan(input)
			if err != nil {
				return nil, fmt.Errorf("Cannot update plan %s: %w.\n\nTo resolve:\n  Verify the plan exists with entity(action: \"get\", id: %q) and check the field values.", entityID, err, entityID)
			}
			return map[string]any{
				"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
			}, nil
		}

		// Regular entities: collect string-valued fields to update.
		fields := make(map[string]string)
		for _, key := range []string{"slug", "summary", "name", "design", "rationale", "observed", "expected", "severity", "priority"} {
			if v, exists := args[key]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					fields[key] = strings.TrimSpace(s)
				}
			}
		}
		if nameVal, hasName := fields["name"]; hasName {
			if _, err := validate.ValidateName(nameVal); err != nil {
				return nil, fmt.Errorf("invalid name: %w", err)
			}
		}

		// List-valued fields (e.g. depends_on for tasks).
		var listFields map[string][]string
		if deps := entityArgStringSlice(args, "depends_on"); len(deps) > 0 {
			if entityType != "task" {
				return nil, fmt.Errorf("Cannot update %s %s: depends_on is only valid for task entities.\n\nTo resolve:\n  Remove the depends_on parameter, or target a TASK-... entity instead.", entityType, entityID)
			}
			for _, dep := range deps {
				if !strings.HasPrefix(dep, "TASK-") {
					return nil, fmt.Errorf("Cannot update task %s: invalid depends_on entry %q.\n\nTo resolve:\n  Each depends_on value must be a TASK-... ID (e.g. TASK-001).", entityID, dep)
				}
			}
			listFields = map[string][]string{"depends_on": deps}
		}

		result, err := entitySvc.UpdateEntity(service.UpdateEntityInput{
			Type:       entityType,
			ID:         entityID,
			Fields:     fields,
			ListFields: listFields,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot update %s %s: %w.\n\nTo resolve:\n  Verify the entity exists with entity(action: \"get\", id: %q) and check the field values.", entityType, entityID, err, entityID)
		}
		return map[string]any{
			"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
		}, nil
	}
}

// ─── transition ───────────────────────────────────────────────────────────────

func entityTransitionAction(entitySvc *service.EntityService, docSvc *service.DocumentService, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		// Signal mutation so side_effects: [] is always present in the response (spec §8.4).
		SignalMutation(ctx)

		args, _ := req.Params.Arguments.(map[string]any)
		entityID := id.NormalizeID(entityArgStr(args, "id"))
		if entityID == "" {
			return nil, fmt.Errorf("id is required for transition")
		}
		newStatus := entityArgStr(args, "status")
		if newStatus == "" {
			return nil, fmt.Errorf("status is required for transition")
		}

		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("cannot infer entity type from ID %q", entityID)
		}

		override, _ := args["override"].(bool)
		overrideReason := entityArgStr(args, "override_reason")

		advance, _ := args["advance"].(bool)

		// Advance mode: walk a feature through multiple states toward the target.
		if advance {
			if entityType != "feature" {
				return nil, fmt.Errorf("advance is only supported for feature entities")
			}
			if override && strings.TrimSpace(overrideReason) == "" {
				return map[string]any{
					"error": "override_reason is required when override is true; cannot bypass gate without a reason",
				}, nil
			}
			return entityAdvanceFeature(ctx, entitySvc, docSvc, entityID, newStatus, override, overrideReason, gateRouter, checkpointStore, requiresHumanReview)
		}

		// Plans use their own status update path (no gate enforcement).
		if entityType == "plan" {
			_, _, slug := model.ParsePlanID(entityID)
			// Load current status before transition for commit message (FR-A11).
			var planFromStatus string
			if preResult, preErr := entitySvc.Get("plan", entityID, ""); preErr == nil {
				planFromStatus, _ = preResult.State["status"].(string)
			}

			// Lifecycle gate: block terminal plan transitions when non-terminal features exist.
			isPlanTerminalTarget := newStatus == string(model.PlanStatusDone) ||
				newStatus == "cancelled" || newStatus == "superseded"
			if isPlanTerminalTarget && !override {
				count, countErr := entitySvc.CountNonTerminalFeatures(entityID)
				if countErr == nil && count > 0 {
					return map[string]any{
						"error": fmt.Sprintf("cannot transition plan %s to %q: %d non-terminal feature(s) must be resolved first", entityID, newStatus, count),
						"gate_failed": map[string]any{
							"from_status": planFromStatus,
							"to_status":   newStatus,
							"reason":      fmt.Sprintf("%d non-terminal feature(s)", count),
						},
					}, nil
				}
			}

			result, err := entitySvc.UpdatePlanStatus(entityID, slug, newStatus)
			if err != nil {
				return entityTransitionError(entitySvc, "plan", entityID, newStatus, err), nil
			}
			// Auto-commit state change (FR-A11). Best-effort.
			planCommitMsg := fmt.Sprintf("workflow(%s): transition %s \u2192 %s", entityID, planFromStatus, newStatus)
			if _, commitErr := entityCommitFunc(".", planCommitMsg); commitErr != nil {
				log.Printf("[entity] WARNING: auto-commit after plan transition %s failed: %v", entityID, commitErr)
			}
			return map[string]any{
				"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
			}, nil
		}

		// structuralChecks holds any structural check results from the gate evaluation,
		// to be included in the response when the transition succeeds.
		var structuralChecks interface{}

		// Feature entities: lifecycle gate for terminal transitions (AC-001 through AC-005).
		// Block done/superseded/cancelled when non-terminal tasks exist.
		if entityType == "feature" && !override {
			isFeatureTerminalTarget := newStatus == string(model.FeatureStatusDone) ||
				newStatus == string(model.FeatureStatusSuperseded) ||
				newStatus == string(model.FeatureStatusCancelled)
			if isFeatureTerminalTarget {
				preResult, preErr := entitySvc.Get("feature", entityID, "")
				var currentStatusForGate string
				if preErr == nil {
					currentStatusForGate, _ = preResult.State["status"].(string)
				}
				count, countErr := entitySvc.CountNonTerminalTasks(entityID)
				if countErr == nil && count > 0 {
					return map[string]any{
						"error": fmt.Sprintf("cannot transition feature %s to %q: %d non-terminal task(s) must be resolved first", entityID, newStatus, count),
						"gate_failed": map[string]any{
							"from_status": currentStatusForGate,
							"to_status":   newStatus,
							"reason":      fmt.Sprintf("%d non-terminal task(s)", count),
						},
					}, nil
				}
			}
		}

		// Feature entities on Phase 2 transitions: evaluate the stage gate (FR-001).
		if entityType == "feature" {
			getResult, err := entitySvc.Get("feature", entityID, "")
			if err != nil {
				return entityTransitionError(entitySvc, entityType, entityID, newStatus, err), nil
			}
			feature := featureFromState(getResult.ID, getResult.Slug, getResult.State)
			currentStatus := string(feature.Status)

			if isPhase2Transition(currentStatus, newStatus) {
				// Evaluate gate through the router (registry → hardcoded fallback).
				var gateResult service.GateResult
				overridePolicy := "agent"
				if gateRouter != nil {
					routerCtx := buildGateEvalContext(feature, docSvc, entitySvc)
					routerResult := gateRouter.CheckGate(currentStatus, newStatus, routerCtx)
					overridePolicy = gateRouter.OverridePolicy(newStatus)

					if routerResult.Source == "registry" {
						// Registry provided prerequisites — use the router result.
						gateResult = service.GateResult{
							Stage:     routerResult.Stage,
							Satisfied: routerResult.Satisfied,
							Reason:    routerResult.Reason,
						}
					} else {
						// Hardcoded fallback — call CheckTransitionGate directly
						// to preserve StructuralChecks and ReviewCapReached.
						gateResult = service.CheckTransitionGate(currentStatus, newStatus, feature, docSvc, entitySvc)
					}
				} else {
					gateResult = service.CheckTransitionGate(currentStatus, newStatus, feature, docSvc, entitySvc)
				}

				if len(gateResult.StructuralChecks) > 0 {
					structuralChecks = gateResult.StructuralChecks
				}
				if !gateResult.Satisfied {
					// Handle review iteration cap (FR-005, FR-006, FR-007).
					if gateResult.ReviewCapReached {
						blockedReason := gateResult.Reason
						_ = entitySvc.PersistFeatureBlockedReason(entityID, "", blockedReason)
						chkStore := checkpoint.NewStore(entitySvc.Root())
						chk, chkErr := chkStore.Create(checkpoint.Record{
							Question:  fmt.Sprintf("Feature %s has reached the review iteration cap (%d/%d). What should happen next?", entityID, feature.ReviewCycle, service.DefaultMaxReviewCycles),
							Context:   fmt.Sprintf("Feature: %s | Review cycle: %d/%d | Options: (1) accept with known issues and transition to done, (2) rework with revised scope and create focused rework tasks, (3) cancel the feature.", entityID, feature.ReviewCycle, service.DefaultMaxReviewCycles),
							Status:    checkpoint.StatusPending,
							CreatedAt: time.Now().UTC(),
							CreatedBy: "system",
						})
						resp := map[string]any{
							"error":          blockedReason,
							"blocked_reason": blockedReason,
							"feature_id":     entityID,
						}
						if chkErr == nil {
							resp["checkpoint_id"] = chk.ID
						}
						return resp, nil
					}
					if !override {
						var nonTerminal []service.TaskStatusPair
						if (currentStatus == "developing" && newStatus == "reviewing") ||
							(currentStatus == "needs-rework" && newStatus == "reviewing") {
							nonTerminal = nonTerminalTasksForFeature(entityID, entitySvc)
						}
						return service.GateFailureResponse(entityID, currentStatus, newStatus, gateResult, nonTerminal), nil
					}
					if strings.TrimSpace(overrideReason) == "" {
						return map[string]any{
							"error": fmt.Sprintf(
								"override_reason is required when override is true; cannot bypass gate on %s→%s without a reason",
								currentStatus, newStatus,
							),
						}, nil
					}
					// Branch on override policy (FR-010 through FR-014).
					if overridePolicy == "checkpoint" && checkpointStore != nil {
						chkResult, chkErr := gate.HandleCheckpointOverride(gate.CheckpointOverrideParams{
							FeatureID:       entityID,
							FromStatus:      currentStatus,
							ToStatus:        newStatus,
							GateDescription: gateResult.Reason,
							OverrideReason:  overrideReason,
							AgentIdentity:   "agent",
							CheckpointStore: checkpointStore,
						})
						if chkErr != nil {
							return nil, fmt.Errorf("creating checkpoint override: %w", chkErr)
						}
						or := model.OverrideRecord{
							FromStatus:   currentStatus,
							ToStatus:     newStatus,
							Reason:       overrideReason,
							Timestamp:    time.Now(),
							CheckpointID: chkResult.CheckpointID,
						}
						feature.Overrides = append(feature.Overrides, or)
						if err := entitySvc.PersistFeatureOverrides(feature.ID, feature.Slug, feature.Overrides); err != nil {
							return nil, fmt.Errorf("persisting override record: %w", err)
						}
						return map[string]any{
							"checkpoint_created": true,
							"checkpoint_id":      chkResult.CheckpointID,
							"message":            chkResult.Message,
							"feature_id":         entityID,
						}, nil
					}
					// Agent policy: log the bypass and continue (FR-014).
					or := model.OverrideRecord{
						FromStatus: currentStatus,
						ToStatus:   newStatus,
						Reason:     overrideReason,
						Timestamp:  time.Now(),
					}
					feature.Overrides = append(feature.Overrides, or)
					if err := entitySvc.PersistFeatureOverrides(feature.ID, feature.Slug, feature.Overrides); err != nil {
						return nil, fmt.Errorf("persisting override record: %w", err)
					}
				}
			}
		}

		// Capture current status before transition for commit message (FR-A11).
		var fromStatusBeforeTransition string
		if preResult, preErr := entitySvc.Get(entityType, entityID, ""); preErr == nil {
			fromStatusBeforeTransition, _ = preResult.State["status"].(string)
		}

		result, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   entityType,
			ID:     entityID,
			Status: newStatus,
		})
		if err != nil {
			return entityTransitionError(entitySvc, entityType, entityID, newStatus, err), nil
		}

		// Report side effects from the status transition hook.
		if wt := result.WorktreeHookResult; wt != nil {
			if wt.Created {
				PushSideEffect(ctx, SideEffect{
					Type:       SideEffectWorktreeCreated,
					EntityID:   entityID,
					EntityType: entityType,
					Extra: map[string]string{
						"worktree_id": wt.WorktreeID,
						"branch":      wt.Branch,
						"path":        wt.Path,
					},
				})
			}
			for _, u := range wt.UnblockedTasks {
				PushSideEffect(ctx, SideEffect{
					Type:       SideEffectTaskUnblocked,
					EntityID:   u.TaskID,
					EntityType: "task",
					FromStatus: u.PreviousStatus,
					ToStatus:   u.Status,
					Trigger:    fmt.Sprintf("All dependencies of %s are now in terminal state", u.TaskID),
				})
			}
		}

		// Increment review_cycle when a feature transitions into reviewing (FR-002).
		if entityType == "feature" && newStatus == string(model.FeatureStatusReviewing) {
			if err := entitySvc.IncrementFeatureReviewCycle(entityID, ""); err != nil {
				return nil, fmt.Errorf("incrementing review_cycle: %w", err)
			}
		}

		resp := map[string]any{
			"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
		}
		if structuralChecks != nil {
			resp["structural_checks"] = structuralChecks
		}

		// Auto-advance: when a task reaches a terminal state, check whether all
		// sibling tasks are done and auto-advance the parent feature to reviewing.
		if entityType == "task" {
			isTaskTerminal := newStatus == string(model.TaskStatusDone) ||
				newStatus == string(model.TaskStatusNotPlanned) ||
				newStatus == "duplicate"
			if isTaskTerminal {
				parentFeatureID := ""
				if taskResult, taskErr := entitySvc.Get("task", entityID, ""); taskErr == nil {
					parentFeatureID, _ = taskResult.State["parent_feature"].(string)
				}
				if parentFeatureID != "" {
					// Capture feature status before the advance fires.
					featFromStatus := "developing"
					if preFeat, preFeatErr := entitySvc.Get("feature", parentFeatureID, ""); preFeatErr == nil {
						if s, _ := preFeat.State["status"].(string); s != "" {
							featFromStatus = s
						}
					}
					advanced, advErr := entitySvc.MaybeAutoAdvanceFeature(parentFeatureID)
					if advanced {
						PushSideEffect(ctx, SideEffect{
							Type:       SideEffectFeatureAutoAdvanced,
							EntityID:   parentFeatureID,
							EntityType: "feature",
							FromStatus: featFromStatus,
							ToStatus:   string(model.FeatureStatusReviewing),
							Trigger:    fmt.Sprintf("all tasks for %s are terminal", parentFeatureID),
						})
						// Also check whether the feature's parent plan can auto-advance.
						planID := entitySvc.FeatureParentPlan(parentFeatureID)
						if planID != "" {
							planAdvanced, planAdvErr := entitySvc.MaybeAutoAdvancePlan(planID)
							if planAdvanced {
								PushSideEffect(ctx, SideEffect{
									Type:       SideEffectPlanAutoAdvanced,
									EntityID:   planID,
									EntityType: "plan",
									FromStatus: string(model.PlanStatusActive),
									ToStatus:   string(model.PlanStatusDone),
									Trigger:    fmt.Sprintf("all features for %s are terminal", planID),
								})
							} else if planAdvErr != nil {
								log.Printf("[entity] WARNING: plan auto-advance check for %s failed: %v", planID, planAdvErr)
							}
						}
					} else if advErr != nil {
						// Surface the auto-advance failure as a warning side effect (AC-017).
						PushSideEffect(ctx, SideEffect{
							Type:       SideEffectFeatureAutoAdvanced,
							EntityID:   parentFeatureID,
							EntityType: "feature",
							Trigger:    fmt.Sprintf("auto-advance failed: %v", advErr),
						})
					}
				}
			}
		}

		// Auto-advance: when a feature reaches a terminal state, check whether all
		// sibling features are terminal and auto-advance the parent plan to done.
		if entityType == "feature" {
			isFeatureTerminal := newStatus == string(model.FeatureStatusDone) ||
				newStatus == string(model.FeatureStatusSuperseded) ||
				newStatus == string(model.FeatureStatusCancelled)
			if isFeatureTerminal {
				planID := entitySvc.FeatureParentPlan(entityID)
				if planID != "" {
					planAdvanced, planAdvErr := entitySvc.MaybeAutoAdvancePlan(planID)
					if planAdvanced {
						PushSideEffect(ctx, SideEffect{
							Type:       SideEffectPlanAutoAdvanced,
							EntityID:   planID,
							EntityType: "plan",
							FromStatus: string(model.PlanStatusActive),
							ToStatus:   string(model.PlanStatusDone),
							Trigger:    fmt.Sprintf("all features for %s are terminal", planID),
						})
					} else if planAdvErr != nil {
						log.Printf("[entity] WARNING: plan auto-advance check for %s failed: %v", planID, planAdvErr)
					}
				}
			}
		}

		// Auto-commit the entity's state file after transition (FR-A11). Best-effort.
		transitionCommitMsg := fmt.Sprintf("workflow(%s): transition %s \u2192 %s", entityID, fromStatusBeforeTransition, newStatus)
		if _, commitErr := entityCommitFunc(".", transitionCommitMsg); commitErr != nil {
			log.Printf("[entity] WARNING: auto-commit after transition %s failed: %v", entityID, commitErr)
		}

		return resp, nil
	}
}

// entityAdvanceFeature loads a feature and calls AdvanceFeatureStatus, returning
// a structured response with the stages advanced through and any stop reason.
func entityAdvanceFeature(ctx context.Context, entitySvc *service.EntityService, docSvc *service.DocumentService, entityID, targetStatus string, override bool, overrideReason string, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) (any, error) {
	// Load the feature entity to get the model struct needed by AdvanceFeatureStatus.
	getResult, err := entitySvc.Get("feature", entityID, "")
	if err != nil {
		return nil, fmt.Errorf("loading feature %s: %w", entityID, err)
	}

	feature := featureFromState(getResult.ID, getResult.Slug, getResult.State)
	startStatus := string(feature.Status)

	advCfg := &service.AdvanceConfig{
		RequiresHumanReview: requiresHumanReview,
	}
	if gateRouter != nil {
		advCfg.CheckGate = func(from, to string, f *model.Feature, ds *service.DocumentService, es *service.EntityService) service.GateResult {
			routerCtx := buildGateEvalContext(f, ds, es)
			routerResult := gateRouter.CheckGate(from, to, routerCtx)
			if routerResult.Source == "registry" {
				return service.GateResult{
					Stage:     routerResult.Stage,
					Satisfied: routerResult.Satisfied,
					Reason:    routerResult.Reason,
				}
			}
			return service.CheckTransitionGate(from, to, f, ds, es)
		}
		advCfg.OverridePolicy = func(to string) string {
			return gateRouter.OverridePolicy(to)
		}
		if checkpointStore != nil {
			advCfg.OnCheckpoint = func(featureID, fromStatus, toStatus, gateReason, overrideReason string) (string, error) {
				result, err := gate.HandleCheckpointOverride(gate.CheckpointOverrideParams{
					FeatureID:       featureID,
					FromStatus:      fromStatus,
					ToStatus:        toStatus,
					GateDescription: gateReason,
					OverrideReason:  overrideReason,
					AgentIdentity:   "agent",
					CheckpointStore: checkpointStore,
				})
				if err != nil {
					return "", err
				}
				return result.CheckpointID, nil
			}
		}
	}
	advResult, err := service.AdvanceFeatureStatus(feature, targetStatus, entitySvc, docSvc, override, overrideReason, advCfg)
	if err != nil {
		return nil, fmt.Errorf("advance feature %s: %w", entityID, err)
	}

	stagesSkipped := len(advResult.AdvancedThrough)
	resp := map[string]any{
		"status":           advResult.FinalStatus,
		"advanced_through": advResult.AdvancedThrough,
	}

	if len(advResult.OverriddenGates) > 0 {
		resp["overridden_gates"] = advResult.OverriddenGates
	}
	if len(advResult.StructuralChecks) > 0 {
		resp["structural_checks"] = advResult.StructuralChecks
	}
	if advResult.CheckpointID != "" {
		resp["checkpoint_created"] = true
		resp["checkpoint_id"] = advResult.CheckpointID
		resp["checkpoint_gate"] = advResult.CheckpointGate
	}

	if advResult.StoppedReason != "" {
		resp["stopped_reason"] = advResult.StoppedReason
		stageWord := "stages"
		if stagesSkipped == 1 {
			stageWord = "stage"
		}
		resp["message"] = fmt.Sprintf(
			"Advanced from %s to %s (%d %s). Stopped: %s",
			startStatus, advResult.FinalStatus, stagesSkipped, stageWord, advResult.StoppedReason,
		)
	} else if stagesSkipped == 0 {
		resp["message"] = fmt.Sprintf("Already at %s", advResult.FinalStatus)
	} else {
		resp["message"] = fmt.Sprintf(
			"Advanced from %s to %s (skipped %d stages with satisfied prerequisites)",
			startStatus, advResult.FinalStatus, stagesSkipped,
		)
	}

	return resp, nil
}

// featureFromState constructs a model.Feature from an entity state map.
func featureFromState(entityID, slug string, state map[string]any) *model.Feature {
	rc, _ := state["review_cycle"].(int)
	blockedReason, _ := state["blocked_reason"].(string)
	return &model.Feature{
		ID:            entityID,
		Slug:          slug,
		Parent:        entityStateStr(state, "parent"),
		Status:        model.FeatureStatus(entityStateStr(state, "status")),
		ReviewCycle:   rc,
		BlockedReason: blockedReason,
		Design:        entityStateStr(state, "design"),
		Spec:          entityStateStr(state, "spec"),
		DevPlan:       entityStateStr(state, "dev_plan"),
		Overrides:     overridesFromState(state),
	}
}

// overridesFromState parses the "overrides" list from a feature entity state map.
func overridesFromState(state map[string]any) []model.OverrideRecord {
	rawSlice, ok := state["overrides"].([]any)
	if !ok || len(rawSlice) == 0 {
		return nil
	}
	result := make([]model.OverrideRecord, 0, len(rawSlice))
	for _, item := range rawSlice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fromStatus, _ := m["from_status"].(string)
		toStatus, _ := m["to_status"].(string)
		reason, _ := m["reason"].(string)
		tsStr, _ := m["timestamp"].(string)
		ts, _ := time.Parse(time.RFC3339, tsStr)
		result = append(result, model.OverrideRecord{
			FromStatus: fromStatus,
			ToStatus:   toStatus,
			Reason:     reason,
			Timestamp:  ts,
		})
	}
	return result
}

// buildGateEvalContext creates a gate.PrereqEvalContext from service types,
// bridging the service layer to the gate evaluator interfaces.
func buildGateEvalContext(feature *model.Feature, docSvc *service.DocumentService, entitySvc *service.EntityService) gate.PrereqEvalContext {
	return gate.PrereqEvalContext{
		Feature:   feature,
		DocSvc:    &gateDocAdapter{svc: docSvc},
		EntitySvc: &gateEntityAdapter{svc: entitySvc},
	}
}

// gateDocAdapter wraps *service.DocumentService to implement gate.DocumentService.
type gateDocAdapter struct {
	svc *service.DocumentService
}

func (a *gateDocAdapter) GetDocument(id string, loadContent bool) (*gate.DocumentRecord, error) {
	result, err := a.svc.GetDocument(id, loadContent)
	if err != nil {
		return nil, err
	}
	return &gate.DocumentRecord{
		ID:     result.ID,
		Status: result.Status,
		Type:   result.Type,
		Owner:  result.Owner,
	}, nil
}

func (a *gateDocAdapter) ListDocuments(filters gate.DocumentFilters) ([]*gate.DocumentRecord, error) {
	results, err := a.svc.ListDocuments(service.DocumentFilters{
		Owner:  filters.Owner,
		Type:   filters.Type,
		Status: filters.Status,
	})
	if err != nil {
		return nil, err
	}
	records := make([]*gate.DocumentRecord, len(results))
	for i, r := range results {
		records[i] = &gate.DocumentRecord{
			ID:     r.ID,
			Status: r.Status,
			Type:   r.Type,
			Owner:  r.Owner,
		}
	}
	return records, nil
}

// gateEntityAdapter wraps *service.EntityService to implement gate.EntityService.
type gateEntityAdapter struct {
	svc *service.EntityService
}

func (a *gateEntityAdapter) List(entityType string) ([]gate.EntityResult, error) {
	results, err := a.svc.List(entityType)
	if err != nil {
		return nil, err
	}
	gateResults := make([]gate.EntityResult, len(results))
	for i, r := range results {
		gateResults[i] = gate.EntityResult{
			ID:    r.ID,
			State: r.State,
		}
	}
	return gateResults, nil
}

// phase2Statuses is the set of Phase 2 feature lifecycle states subject to
// mandatory gate enforcement (NFR-002).
var phase2Statuses = map[string]bool{
	"proposed":     true,
	"designing":    true,
	"specifying":   true,
	"dev-planning": true,
	"developing":   true,
	"reviewing":    true,
	"needs-rework": true,
	"done":         true,
}

// isPhase2Transition reports whether a feature transition between two statuses
// is in the Phase 2 document-driven lifecycle, and therefore subject to gate
// enforcement (NFR-002). Phase 1 statuses (draft, in-review, approved,
// in-progress, review) are excluded.
func isPhase2Transition(from, to string) bool {
	return phase2Statuses[from] && phase2Statuses[to]
}

// nonTerminalTasksForFeature returns the list of child tasks for a feature
// that are not in a terminal state. Used to enrich gate failure messages for
// task-completeness gates (FR-022, FR-025).
func nonTerminalTasksForFeature(featureID string, entitySvc *service.EntityService) []service.TaskStatusPair {
	tasks, err := entitySvc.List("task")
	if err != nil {
		return nil
	}
	var result []service.TaskStatusPair
	for _, t := range tasks {
		parentFeature, _ := t.State["parent_feature"].(string)
		if parentFeature != featureID {
			continue
		}
		status, _ := t.State["status"].(string)
		if !validate.IsTaskDependencySatisfied(status) {
			result = append(result, service.TaskStatusPair{ID: t.ID, Status: status})
		}
	}
	return result
}

// ─── Transition error enrichment ─────────────────────────────────────────────

// entityTransitionError builds a structured error response for a failed lifecycle
// transition. It enriches the error with the entity's current status and the valid
// next states, giving agents the context they need to correct the request
// (spec §14.7: "Invalid transitions return an error naming the current status,
// the requested status, and the valid transitions from the current state").
//
// Returns (errorMap, nil) so that WithSideEffects merges side_effects into the
// response — mutation responses always include side_effects (spec §8.4).
func entityTransitionError(entitySvc *service.EntityService, entityType, entityID, requested string, cause error) map[string]any {
	details := map[string]any{
		"requested_status": requested,
	}

	// Best-effort enrichment: fetch current status and compute valid next states.
	// The extra read only happens on the error path, not the hot path.
	if currentStatus, err := entityCurrentStatus(entitySvc, entityType, entityID); err == nil {
		details["current_status"] = currentStatus
		if kind, ok := entityKindFromType(entityType); ok {
			if next := validate.NextStates(kind, currentStatus); len(next) > 0 {
				sort.Strings(next)
				details["valid_transitions"] = next
			}
		}
	}

	return map[string]any{
		"error": map[string]any{
			"code":    "invalid_transition",
			"message": cause.Error(),
			"details": details,
		},
	}
}

// entityCurrentStatus fetches only the current lifecycle status of an entity.
func entityCurrentStatus(entitySvc *service.EntityService, entityType, entityID string) (string, error) {
	if entityType == "plan" {
		r, err := entitySvc.GetPlan(entityID)
		if err != nil {
			return "", err
		}
		status, _ := r.State["status"].(string)
		return status, nil
	}
	r, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return "", err
	}
	status, _ := r.State["status"].(string)
	return status, nil
}

// entityKindFromType maps an entity type string to its validate.EntityKind.
// Returns (kind, true) for known types, ("", false) for unknown types.
func entityKindFromType(entityType string) (validate.EntityKind, bool) {
	switch entityType {
	case "plan":
		return validate.EntityPlan, true
	case "feature":
		return validate.EntityFeature, true
	case "task":
		return validate.EntityTask, true
	case "bug":
		return validate.EntityBug, true
	case "decision":
		return validate.EntityDecision, true
	case "incident":
		return validate.EntityIncident, true
	default:
		return validate.EntityKind(""), false
	}
}

// ─── Type inference from ID prefix (§14.8) ───────────────────────────────────

// entityTypeFromID infers the entity type from an ID string using its prefix.
func entityInferType(entityID string) (entityType string, ok bool) {
	upper := strings.ToUpper(entityID)
	switch {
	case strings.HasPrefix(upper, "FEAT-"):
		return "feature", true
	case strings.HasPrefix(upper, "TASK-"), strings.HasPrefix(upper, "T-"):
		return "task", true
	case strings.HasPrefix(upper, "BUG-"):
		return "bug", true
	case strings.HasPrefix(upper, "DEC-"):
		return "decision", true
	case strings.HasPrefix(upper, "INC-"):
		return "incident", true
	case model.IsPlanID(entityID):
		return "plan", true
	default:
		return "", false
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// entityArgStr extracts a trimmed string from an MCP args map.
func entityArgStr(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

// entityArgStringSlice extracts a []string from an MCP args map.
func entityArgStringSlice(args map[string]any, key string) []string {
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// entityStateStr safely reads a string field from an entity state map.
func entityStateStr(state map[string]any, key string) string {
	if state == nil {
		return ""
	}
	s, _ := state[key].(string)
	return s
}

// entityFullRecord builds the full response map for a get/update/transition response.
// It merges the entity ID, type, slug, display_id, and entity_ref on top of the full state.
// display_id and entity_ref are placed first conceptually (JSON key order is not
// guaranteed, but callers reading the map will see all fields).
func entityFullRecord(entityID, entityType, slug string, state map[string]any) map[string]any {
	out := make(map[string]any, len(state)+6)
	for k, v := range state {
		out[k] = v
	}
	displayID := id.FormatFullDisplay(entityID)
	entityName, _ := state["name"].(string)
	out["display_id"] = displayID
	out["id"] = entityID
	out["type"] = entityType
	out["slug"] = slug
	out["entity_ref"] = id.FormatEntityRef(displayID, slug, entityName)
	return out
}
