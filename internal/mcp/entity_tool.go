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
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/id"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/model"
	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

// EntityTool returns the consolidated `entity` MCP tool registered in the core group.
func EntityTool(entitySvc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{entityTool(entitySvc)}
}

func entityTool(entitySvc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("entity",
		mcp.WithDescription(
			"Generic CRUD for all entity types (plan, feature, task, bug, epic, decision). "+
				"Replaces create_task, create_feature, create_plan, get_entity, list_entities, "+
				"update_entity, update_status, and related 1.0 tools. "+
				"Actions: create, get, list, update, transition. "+
				"For get/update/transition, entity type is inferred from the ID prefix "+
				"(FEAT-=feature, TASK-/T-=task, BUG-=bug, EPIC-=epic, DEC-=decision, plan prefix=plan). "+
				"create supports batch mode via the entities array parameter.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: create, get, list, update, transition"),
		),
		mcp.WithString("type", mcp.Description("Entity type: plan, feature, task, bug, epic, decision (required for create and list)")),
		mcp.WithString("id", mcp.Description("Entity ID — type inferred from prefix (required for get, update, transition)")),
		mcp.WithString("status", mcp.Description("Target status (transition) or status filter (list)")),
		mcp.WithString("parent", mcp.Description("Parent ID filter: plan ID for features, feature ID for tasks (list only)")),
		mcp.WithArray("tags", mcp.Description("Tag filter (list) or tags to set (create/update)")),
		mcp.WithArray("entities", mcp.Description("Batch create: array of entity objects, each with the same fields as single create")),
		// Common entity fields (type-specific, all optional at top level).
		mcp.WithString("slug", mcp.Description("URL-friendly identifier")),
		mcp.WithString("summary", mcp.Description("Brief summary")),
		mcp.WithString("title", mcp.Description("Human-readable title (plan, epic, bug)")),
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
		mcp.WithString("created_after", mcp.Description("Created-after filter, RFC3339 (list only)")),
		mcp.WithString("created_before", mcp.Description("Created-before filter, RFC3339 (list only)")),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"create":     entityCreateAction(entitySvc),
			"get":        entityGetAction(entitySvc),
			"list":       entityListAction(entitySvc),
			"update":     entityUpdateAction(entitySvc),
			"transition": entityTransitionAction(entitySvc),
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
			return nil, fmt.Errorf("type is required for create")
		}

		// Signal mutation so side_effects: [] is always present in both
		// single and batch responses (spec §8.4: "The field is never omitted").
		SignalMutation(ctx)

		// Batch mode: entities array provided.
		if IsBatchInput(args, "entities") {
			items, _ := args["entities"].([]any)
			return ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				m, ok := item.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("each entity must be an object")
				}
				result, err := entityCreateOne(entityType, m, entitySvc)
				return entityArgStr(m, "slug"), result, err
			})
		}

		// Single mode.
		return entityCreateOne(entityType, args, entitySvc)
	}
}

func entityCreateOne(entityType string, args map[string]any, entitySvc *service.EntityService) (any, error) {
	createdByRaw := entityArgStr(args, "created_by")
	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return nil, fmt.Errorf("resolve identity: %w", err)
	}

	// Advisory duplicate check runs before creation so it checks pre-existing entities only.
	advisory := entityDuplicateAdvisory(entityType, args, entitySvc)

	var result service.CreateResult

	switch entityType {
	case "task":
		result, err = entitySvc.CreateTask(service.CreateTaskInput{
			ParentFeature: entityArgStr(args, "parent_feature"),
			Slug:          entityArgStr(args, "slug"),
			Summary:       entityArgStr(args, "summary"),
		})

	case "feature":
		result, err = entitySvc.CreateFeature(service.CreateFeatureInput{
			Slug:      entityArgStr(args, "slug"),
			Parent:    entityArgStr(args, "parent"),
			Summary:   entityArgStr(args, "summary"),
			Design:    entityArgStr(args, "design"),
			Tags:      entityArgStringSlice(args, "tags"),
			CreatedBy: createdBy,
		})

	case "plan":
		result, err = entitySvc.CreatePlan(service.CreatePlanInput{
			Prefix:    entityArgStr(args, "prefix"),
			Slug:      entityArgStr(args, "slug"),
			Title:     entityArgStr(args, "title"),
			Summary:   entityArgStr(args, "summary"),
			Tags:      entityArgStringSlice(args, "tags"),
			CreatedBy: createdBy,
		})

	case "bug":
		result, err = entitySvc.CreateBug(service.CreateBugInput{
			Slug:       entityArgStr(args, "slug"),
			Title:      entityArgStr(args, "title"),
			ReportedBy: entityArgStr(args, "reported_by"),
			Observed:   entityArgStr(args, "observed"),
			Expected:   entityArgStr(args, "expected"),
			Severity:   entityArgStr(args, "severity"),
			Priority:   entityArgStr(args, "priority"),
			Type:       entityArgStr(args, "bug_type"),
		})

	case "epic":
		result, err = entitySvc.CreateEpic(service.CreateEpicInput{
			Slug:      entityArgStr(args, "slug"),
			Title:     entityArgStr(args, "title"),
			Summary:   entityArgStr(args, "summary"),
			CreatedBy: createdBy,
		})

	case "decision":
		result, err = entitySvc.CreateDecision(service.CreateDecisionInput{
			Slug:      entityArgStr(args, "slug"),
			Summary:   entityArgStr(args, "summary"),
			Rationale: entityArgStr(args, "rationale"),
			DecidedBy: createdBy,
		})

	default:
		return nil, fmt.Errorf("unknown entity type %q; valid: plan, feature, task, bug, epic, decision", entityType)
	}

	if err != nil {
		return nil, err
	}

	out := map[string]any{
		"entity": map[string]any{
			"id":         result.ID,
			"type":       result.Type,
			"slug":       result.Slug,
			"status":     entityStateStr(result.State, "status"),
			"display_id": id.FormatFullDisplay(result.ID),
		},
	}

	if len(advisory) > 0 {
		out["duplicate_advisory"] = advisory
	}

	return out, nil
}

// entityDuplicateAdvisory runs an advisory (non-blocking) similarity check against
// pre-existing entities. Returns nil when no duplicates are found or the check fails.
func entityDuplicateAdvisory(entityType string, args map[string]any, entitySvc *service.EntityService) []map[string]any {
	title := entityArgStr(args, "title")
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
				t, _ := p.State["title"].(string)
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
				t, _ := r.State["title"].(string)
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
		entityID := entityArgStr(args, "id")
		if entityID == "" {
			return nil, fmt.Errorf("id is required for get")
		}

		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("cannot infer entity type from ID %q; use a prefixed ID (FEAT-..., TASK-..., BUG-..., etc.)", entityID)
		}

		if entityType == "plan" {
			result, err := entitySvc.GetPlan(entityID)
			if err != nil {
				return nil, fmt.Errorf("get plan %s: %w", entityID, err)
			}
			return map[string]any{
				"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
			}, nil
		}

		result, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			return nil, fmt.Errorf("get %s %s: %w", entityType, entityID, err)
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
			return nil, fmt.Errorf("type is required for list")
		}

		statusFilter := entityArgStr(args, "status")
		parentFilter := entityArgStr(args, "parent")
		tagsFilter := entityArgStringSlice(args, "tags")

		var createdAfter, createdBefore *time.Time
		if caStr := entityArgStr(args, "created_after"); caStr != "" {
			t, err := time.Parse(time.RFC3339, caStr)
			if err != nil {
				return nil, fmt.Errorf("invalid created_after: %w", err)
			}
			createdAfter = &t
		}
		if cbStr := entityArgStr(args, "created_before"); cbStr != "" {
			t, err := time.Parse(time.RFC3339, cbStr)
			if err != nil {
				return nil, fmt.Errorf("invalid created_before: %w", err)
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
				return nil, fmt.Errorf("list plans: %w", err)
			}
			return entityListResponse(entityType, entitySummaries(plans)), nil
		}

		// For tasks with a parent filter: ListEntitiesFiltered checks r.State["parent"]
		// but tasks store their parent in r.State["parent_feature"]. Filter manually.
		if entityType == "task" && parentFilter != "" {
			allTasks, err := entitySvc.List("task")
			if err != nil {
				return nil, fmt.Errorf("list tasks: %w", err)
			}
			var filtered []service.ListResult
			for _, t := range allTasks {
				pf, _ := t.State["parent_feature"].(string)
				if pf != parentFilter {
					continue
				}
				if statusFilter != "" {
					st, _ := t.State["status"].(string)
					if st != statusFilter {
						continue
					}
				}
				if len(tagsFilter) > 0 && !entityHasAnyTag(t.State, tagsFilter) {
					continue
				}
				filtered = append(filtered, t)
			}
			return entityListResponse(entityType, entitySummaries(filtered)), nil
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
			return nil, fmt.Errorf("list %s: %w", entityType, err)
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
		if summary == "" {
			summary, _ = r.State["title"].(string)
		}
		status, _ := r.State["status"].(string)
		out = append(out, map[string]any{
			"id":         r.ID,
			"type":       r.Type,
			"slug":       r.Slug,
			"status":     status,
			"summary":    summary,
			"display_id": id.FormatFullDisplay(r.ID),
		})
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
		entityID := entityArgStr(args, "id")
		if entityID == "" {
			return nil, fmt.Errorf("id is required for update")
		}

		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("cannot infer entity type from ID %q", entityID)
		}

		// Plans use their own update path (supports title, summary, design, tags).
		if entityType == "plan" {
			_, _, slug := model.ParsePlanID(entityID)
			input := service.UpdatePlanInput{
				ID:   entityID,
				Slug: slug,
			}
			if _, has := args["title"]; has {
				v := entityArgStr(args, "title")
				input.Title = &v
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
				return nil, fmt.Errorf("update plan %s: %w", entityID, err)
			}
			return map[string]any{
				"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
			}, nil
		}

		// Regular entities: collect string-valued fields to update.
		fields := make(map[string]string)
		for _, key := range []string{"slug", "summary", "title", "design", "rationale", "observed", "expected", "severity", "priority"} {
			if v, exists := args[key]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					fields[key] = strings.TrimSpace(s)
				}
			}
		}

		result, err := entitySvc.UpdateEntity(service.UpdateEntityInput{
			Type:   entityType,
			ID:     entityID,
			Fields: fields,
		})
		if err != nil {
			return nil, fmt.Errorf("update %s %s: %w", entityType, entityID, err)
		}
		return map[string]any{
			"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
		}, nil
	}
}

// ─── transition ───────────────────────────────────────────────────────────────

func entityTransitionAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		// Signal mutation so side_effects: [] is always present in the response (spec §8.4).
		SignalMutation(ctx)

		args, _ := req.Params.Arguments.(map[string]any)
		entityID := entityArgStr(args, "id")
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

		// Plans use their own status update path.
		if entityType == "plan" {
			_, _, slug := model.ParsePlanID(entityID)
			result, err := entitySvc.UpdatePlanStatus(entityID, slug, newStatus)
			if err != nil {
				return entityTransitionError(entitySvc, "plan", entityID, newStatus, err), nil
			}
			return map[string]any{
				"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
			}, nil
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
					ToStatus:   u.Status,
					Trigger:    fmt.Sprintf("Dependencies of %s now in terminal state", u.TaskID),
				})
			}
		}

		return map[string]any{
			"entity": entityFullRecord(result.ID, result.Type, result.Slug, result.State),
		}, nil
	}
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
	case "epic":
		return validate.EntityEpic, true
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
	case strings.HasPrefix(upper, "EPIC-"):
		return "epic", true
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
// It merges the entity ID, type, slug, and display_id on top of the full state.
func entityFullRecord(entityID, entityType, slug string, state map[string]any) map[string]any {
	out := make(map[string]any, len(state)+4)
	for k, v := range state {
		out[k] = v
	}
	out["id"] = entityID
	out["type"] = entityType
	out["slug"] = slug
	out["display_id"] = id.FormatFullDisplay(entityID)
	return out
}
