// Package mcp entity_tool.go — consolidated entity CRUD for Kanbanzai 2.0 (Track H).
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

var entityCommitFunc = func(repoRoot, message string) (bool, error) {
	return git.CommitStateWithMessage(repoRoot, message)
}

func EntityTool(entitySvc *service.EntityService, docSvc *service.DocumentService, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) []server.ServerTool {
	return []server.ServerTool{entityTool(entitySvc, docSvc, gateRouter, checkpointStore, requiresHumanReview)}
}

func entityTool(entitySvc *service.EntityService, docSvc *service.DocumentService, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) server.ServerTool {
	tool := mcp.NewTool("entity",
		mcp.WithTitleAnnotation("Entity Manager"),
		mcp.WithDescription(
			"The primary tool for managing workflow entities (batches, features, tasks, bugs, decisions) — "+
				"use this whenever you need to create, query, modify, or advance entities through their lifecycle. "+
				"Actions: create, get, list, update, transition.",
		),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: create, get, list, update, transition")),
		mcp.WithString("type", mcp.Description("Entity type: batch, feature, task, bug, decision, strategic-plan")),
		mcp.WithString("id", mcp.Description("Entity ID")),
		mcp.WithString("status", mcp.Description("Target status (transition) or status filter (list)")),
		mcp.WithString("parent", mcp.Description("Parent batch/plan ID for features")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Tag filter (list) or tags to set (create/update)")),
		mcp.WithArray("entities", mcp.Items(map[string]any{"type": "object"}), mcp.Description("Batch create: array of entity objects")),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier")),
		mcp.WithString("summary", mcp.Description("Brief summary")),
		mcp.WithString("name", mcp.Description("Human-readable display name")),
		mcp.WithString("prefix", mcp.Description("Single-character Batch ID prefix")),
		mcp.WithString("parent_feature", mcp.Description("Parent feature ID (task create only)")),
		mcp.WithString("rationale", mcp.Description("Decision rationale")),
		mcp.WithString("reported_by", mcp.Description("Who reported it")),
		mcp.WithString("observed", mcp.Description("Observed behavior")),
		mcp.WithString("expected", mcp.Description("Expected behavior")),
		mcp.WithString("severity", mcp.Description("Bug severity: low, medium, high, critical")),
		mcp.WithString("priority", mcp.Description("Bug priority: low, medium, high, critical")),
		mcp.WithString("bug_type", mcp.Description("Bug type")),
		mcp.WithString("created_by", mcp.Description("Who created it")),
		mcp.WithString("design", mcp.Description("Design document reference")),
		mcp.WithArray("depends_on", mcp.WithStringItems(), mcp.Description("Task IDs this task depends on")),
		mcp.WithString("created_after", mcp.Description("Created-after filter, RFC3339")),
		mcp.WithString("created_before", mcp.Description("Created-before filter, RFC3339")),
		mcp.WithBoolean("advance", mcp.Description("When true, advance a feature through multiple lifecycle states")),
		mcp.WithBoolean("override", mcp.Description("Bypass a failing stage gate prerequisite")),
		mcp.WithString("override_reason", mcp.Description("Required when override is true")),
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

func entityCreateAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		entityType := strings.ToLower(entityArgStr(args, "type"))
		if entityType == "" {
			return nil, fmt.Errorf("type is required for create")
		}
		SignalMutation(ctx)
		if IsBatchInput(args, "entities") {
			items, _ := args["entities"].([]any)
			result, err := ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				m, ok := item.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("invalid entity object in batch")
				}
				r, e := entityCreateOne(entityType, m, entitySvc)
				return entityArgStr(m, "slug"), r, e
			})
			if _, err := entityCommitFunc(".", fmt.Sprintf("workflow: create %d %s entities", len(items), entityType)); err != nil {
				log.Printf("WARNING: commit after batch create failed: %v", err)
			}
			return result, err
		}
		return entityCreateOne(entityType, args, entitySvc)
	}
}

func entityCreateOne(entityType string, args map[string]any, entitySvc *service.EntityService) (any, error) {
	createdByRaw := entityArgStr(args, "created_by")
	createdBy, err := config.ResolveIdentity(createdByRaw)
	if err != nil {
		return nil, fmt.Errorf("cannot create %s: %v", entityType, err)
	}
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
			Slug: entityArgStr(args, "slug"), Parent: entityArgStr(args, "parent"),
			Summary: entityArgStr(args, "summary"), Design: entityArgStr(args, "design"),
			Tags: entityArgStringSlice(args, "tags"), CreatedBy: createdBy, Name: name,
		})
	case "batch", "plan":
		result, err = entitySvc.CreateBatch(service.CreateBatchInput{
			Prefix: entityArgStr(args, "prefix"), Slug: entityArgStr(args, "slug"),
			Name: name, Summary: entityArgStr(args, "summary"),
			Parent: entityArgStr(args, "parent"), Tags: entityArgStringSlice(args, "tags"), CreatedBy: createdBy,
		})
	case "strategic-plan":
		var order int
		if v, ok := args["order"].(int); ok {
			order = v
		}
		result, err = entitySvc.CreateStrategicPlan(service.CreateStrategicPlanInput{
			Prefix: entityArgStr(args, "prefix"), Slug: entityArgStr(args, "slug"),
			Name: name, Summary: entityArgStr(args, "summary"),
			Parent: entityArgStr(args, "parent"), DependsOn: entityArgStringSlice(args, "depends_on"),
			Order: order, Tags: entityArgStringSlice(args, "tags"), CreatedBy: createdBy,
		})
	case "bug":
		result, err = entitySvc.CreateBug(service.CreateBugInput{
			Slug: entityArgStr(args, "slug"), Name: name,
			ReportedBy: entityArgStr(args, "reported_by"), Observed: entityArgStr(args, "observed"),
			Expected: entityArgStr(args, "expected"), Severity: entityArgStr(args, "severity"),
			Priority: entityArgStr(args, "priority"), Type: entityArgStr(args, "bug_type"),
		})
	case "decision":
		result, err = entitySvc.CreateDecision(service.CreateDecisionInput{
			Slug: entityArgStr(args, "slug"), Name: name,
			Summary: entityArgStr(args, "summary"), Rationale: entityArgStr(args, "rationale"),
			DecidedBy: createdBy,
		})
	default:
		return nil, fmt.Errorf("unknown type %q", entityType)
	}
	if err != nil {
		return nil, err
	}

	entityOut := map[string]any{
		"display_id": id.FormatFullDisplay(result.ID), "id": result.ID, "type": result.Type,
		"slug": result.Slug, "name": entityStateStr(result.State, "name"),
		"status":     entityStateStr(result.State, "status"),
		"entity_ref": id.FormatEntityRef(id.FormatFullDisplay(result.ID), result.Slug, entityStateStr(result.State, "name")),
	}
	out := map[string]any{"entity": entityOut}
	if len(advisory) > 0 {
		out["duplicate_advisory"] = advisory
	}
	if _, err := entityCommitFunc(".", fmt.Sprintf("workflow(%s): create %s", result.ID, result.Type)); err != nil {
		log.Printf("WARNING: commit after create %s failed: %v", result.ID, err)
	}
	return out, nil
}

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
	if entityType == "batch" || entityType == "plan" {
		batches, err := entitySvc.ListBatches(service.BatchFilters{})
		if err == nil {
			for _, b := range batches {
				t, _ := b.State["name"].(string)
				s, _ := b.State["summary"].(string)
				existing = append(existing, knowledge.ExistingEntity{ID: b.ID, Type: "batch", Title: t, Summary: s})
			}
		}
	} else if entityType == "strategic-plan" {
		plans, err := entitySvc.ListStrategicPlans(service.StrategicPlanFilters{})
		if err == nil {
			for _, p := range plans {
				t, _ := p.State["name"].(string)
				s, _ := p.State["summary"].(string)
				existing = append(existing, knowledge.ExistingEntity{ID: p.ID, Type: "strategic-plan", Title: t, Summary: s})
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
				existing = append(existing, knowledge.ExistingEntity{ID: r.ID, Type: entityType, Title: t, Summary: s})
			}
		}
	}
	candidates := knowledge.FindDuplicateCandidates(title, summary, existing, 0.5)
	if len(candidates) == 0 {
		return nil
	}
	out := make([]map[string]any, len(candidates))
	for i, c := range candidates {
		out[i] = map[string]any{"entity_id": c.EntityID, "entity_type": c.EntityType, "title": c.Title, "similarity": c.Similarity}
	}
	return out
}

func entityGetAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		entityID := id.NormalizeID(entityArgStr(args, "id"))
		if entityID == "" {
			return nil, fmt.Errorf("id is required for get")
		}
		explicitType := entityArgStr(args, "type")
		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("unrecognised ID format %q", entityID)
		}
		if explicitType == "strategic-plan" {
			entityType = "strategic-plan"
		}
		if entityType == "strategic-plan" {
			r, err := entitySvc.GetStrategicPlan(entityID)
			if err != nil {
				return nil, fmt.Errorf("cannot get strategic plan %s: %w", entityID, err)
			}
			return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
		}
		if entityType == "batch" {
			r, err := entitySvc.GetBatch(entityID)
			if err != nil {
				return nil, fmt.Errorf("cannot get batch %s: %w", entityID, err)
			}
			return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
		}
		r, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			return nil, fmt.Errorf("cannot get %s %s: %w", entityType, entityID, err)
		}
		return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
	}
}

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
			if err == nil {
				createdAfter = &t
			}
		}
		if cbStr := entityArgStr(args, "created_before"); cbStr != "" {
			t, err := time.Parse(time.RFC3339, cbStr)
			if err == nil {
				createdBefore = &t
			}
		}
		if entityType == "strategic-plan" {
			plans, err := entitySvc.ListStrategicPlans(service.StrategicPlanFilters{Status: statusFilter, Tags: tagsFilter})
			if err != nil {
				return nil, fmt.Errorf("cannot list strategic plans: %w", err)
			}
			return entityListResponse(entityType, entitySummaries(plans)), nil
		}
		if entityType == "batch" || entityType == "plan" {
			batches, err := entitySvc.ListBatches(service.BatchFilters{Status: statusFilter, Tags: tagsFilter})
			if err != nil {
				return nil, fmt.Errorf("cannot list batches: %w", err)
			}
			return entityListResponse(entityType, entitySummaries(batches)), nil
		}
		results, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
			Type: entityType, Status: statusFilter, Parent: parentFilter,
			Tags: tagsFilter, CreatedAfter: createdAfter, CreatedBefore: createdBefore,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot list %s entities: %w", entityType, err)
		}
		return entityListResponse(entityType, entitySummaries(results)), nil
	}
}

func entityListResponse(entityType string, summaries []map[string]any) map[string]any {
	if summaries == nil {
		summaries = []map[string]any{}
	}
	return map[string]any{"entities": summaries, "total": len(summaries), "type": entityType}
}

func entitySummaries(results []service.ListResult) []map[string]any {
	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		summary, _ := r.State["summary"].(string)
		name, _ := r.State["name"].(string)
		status, _ := r.State["status"].(string)
		did := id.FormatFullDisplay(r.ID)
		out = append(out, map[string]any{
			"display_id": did, "id": r.ID, "type": r.Type, "slug": r.Slug,
			"name": name, "status": status, "summary": summary,
			"entity_ref": id.FormatEntityRef(did, r.Slug, name),
		})
	}
	return out
}

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

func entityUpdateAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)
		entityID := id.NormalizeID(entityArgStr(args, "id"))
		if entityID == "" {
			return nil, fmt.Errorf("id is required for update")
		}
		entityType, ok := entityInferType(entityID)
		if !ok {
			return nil, fmt.Errorf("unrecognised ID format %q", entityID)
		}
		explicitType := entityArgStr(args, "type")
		if explicitType == "strategic-plan" {
			entityType = "strategic-plan"
		}
		if entityType == "strategic-plan" {
			_, _, slug := model.ParseBatchID(entityID)
			input := service.UpdateStrategicPlanInput{ID: entityID, Slug: slug}
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
			if _, has := args["parent"]; has {
				v := entityArgStr(args, "parent")
				input.Parent = &v
			}
			if _, has := args["order"]; has {
				if v, ok := args["order"].(int); ok {
					input.Order = &v
				}
			}
			if _, has := args["depends_on"]; has {
				input.DependsOn = entityArgStringSlice(args, "depends_on")
			}
			if _, has := args["tags"]; has {
				input.Tags = entityArgStringSlice(args, "tags")
			}
			r, err := entitySvc.UpdateStrategicPlan(input)
			if err != nil {
				return nil, fmt.Errorf("cannot update strategic plan %s: %w", entityID, err)
			}
			return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
		}
		if entityType == "batch" {
			_, _, slug := model.ParseBatchID(entityID)
			input := service.UpdateBatchInput{ID: entityID, Slug: slug}
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
			r, err := entitySvc.UpdateBatch(input)
			if err != nil {
				return nil, fmt.Errorf("cannot update batch %s: %w", entityID, err)
			}
			return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
		}
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
		var listFields map[string][]string
		if deps := entityArgStringSlice(args, "depends_on"); len(deps) > 0 {
			if entityType != "task" {
				return nil, fmt.Errorf("depends_on is only valid for task entities")
			}
			listFields = map[string][]string{"depends_on": deps}
		}
		r, err := entitySvc.UpdateEntity(service.UpdateEntityInput{Type: entityType, ID: entityID, Fields: fields, ListFields: listFields})
		if err != nil {
			return nil, fmt.Errorf("cannot update %s %s: %w", entityType, entityID, err)
		}
		return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
	}
}

func entityTransitionAction(entitySvc *service.EntityService, docSvc *service.DocumentService, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
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
		explicitType := entityArgStr(args, "type")
		if explicitType == "strategic-plan" {
			entityType = "strategic-plan"
		}
		override, _ := args["override"].(bool)
		overrideReason := entityArgStr(args, "override_reason")
		advance, _ := args["advance"].(bool)
		if advance {
			if entityType != "feature" {
				return nil, fmt.Errorf("advance is only supported for feature entities")
			}
			if override && strings.TrimSpace(overrideReason) == "" {
				return map[string]any{"error": "override_reason is required when override is true"}, nil
			}
			return entityAdvanceFeature(ctx, entitySvc, docSvc, entityID, newStatus, override, overrideReason, gateRouter, checkpointStore, requiresHumanReview)
		}
		if entityType == "strategic-plan" {
			_, _, slug := model.ParsePlanID(entityID)
			var fromStatus string
			if pre, preErr := entitySvc.GetStrategicPlan(entityID); preErr == nil {
				fromStatus, _ = pre.State["status"].(string)
			}
			r, err := entitySvc.UpdateStrategicPlanStatus(entityID, slug, newStatus)
			if err != nil {
				return entityTransitionError(entitySvc, "strategic-plan", entityID, newStatus, err), nil
			}
			if _, err := entityCommitFunc(".", fmt.Sprintf("workflow(%s): transition %s → %s", entityID, fromStatus, newStatus)); err != nil {
				log.Printf("WARNING: commit after strategic-plan transition failed: %v", err)
			}
			return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
		}
		if entityType == "batch" {
			_, _, slug := model.ParseBatchID(entityID)
			var batchFromStatus string
			if pre, preErr := entitySvc.GetBatch(entityID); preErr == nil {
				batchFromStatus, _ = pre.State["status"].(string)
			}
			isTerminal := newStatus == string(model.BatchStatusDone) || newStatus == "cancelled" || newStatus == "superseded"
			if isTerminal && !override {
				if count, countErr := entitySvc.CountNonTerminalFeatures(entityID); countErr == nil && count > 0 {
					return map[string]any{
						"error":       fmt.Sprintf("cannot transition batch %s to %q: %d non-terminal feature(s)", entityID, newStatus, count),
						"gate_failed": map[string]any{"from_status": batchFromStatus, "to_status": newStatus, "reason": fmt.Sprintf("%d non-terminal feature(s)", count)},
					}, nil
				}
			}
			r, err := entitySvc.UpdateBatchStatus(entityID, slug, newStatus)
			if err != nil {
				return entityTransitionError(entitySvc, "batch", entityID, newStatus, err), nil
			}
			if _, err := entityCommitFunc(".", fmt.Sprintf("workflow(%s): transition %s → %s", entityID, batchFromStatus, newStatus)); err != nil {
				log.Printf("WARNING: commit after batch transition failed: %v", err)
			}
			return map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}, nil
		}
		var structuralChecks interface{}
		if entityType == "feature" && !override {
			isTerminal := newStatus == string(model.FeatureStatusDone) || newStatus == string(model.FeatureStatusSuperseded) || newStatus == string(model.FeatureStatusCancelled)
			if isTerminal {
				pre, preErr := entitySvc.Get("feature", entityID, "")
				var curStatus string
				if preErr == nil {
					curStatus, _ = pre.State["status"].(string)
				}
				if count, countErr := entitySvc.CountNonTerminalTasks(entityID); countErr == nil && count > 0 {
					return map[string]any{
						"error":       fmt.Sprintf("cannot transition feature %s to %q: %d non-terminal task(s)", entityID, newStatus, count),
						"gate_failed": map[string]any{"from_status": curStatus, "to_status": newStatus, "reason": fmt.Sprintf("%d non-terminal task(s)", count)},
					}, nil
				}
			}
		}
		if entityType == "feature" {
			getR, err := entitySvc.Get("feature", entityID, "")
			if err != nil {
				return entityTransitionError(entitySvc, entityType, entityID, newStatus, err), nil
			}
			feature := featureFromState(getR.ID, getR.Slug, getR.State)
			currentStatus := string(feature.Status)
			if isPhase2Transition(currentStatus, newStatus) {
				var gateResult service.GateResult
				overridePolicy := "agent"
				if gateRouter != nil {
					routerCtx := buildGateEvalContext(feature, docSvc, entitySvc)
					routerResult := gateRouter.CheckGate(currentStatus, newStatus, routerCtx)
					overridePolicy = gateRouter.OverridePolicy(newStatus)
					if routerResult.Source == "registry" {
						gateResult = service.GateResult{Stage: routerResult.Stage, Satisfied: routerResult.Satisfied, Reason: routerResult.Reason}
					} else {
						gateResult = service.CheckTransitionGate(currentStatus, newStatus, feature, docSvc, entitySvc)
					}
				} else {
					gateResult = service.CheckTransitionGate(currentStatus, newStatus, feature, docSvc, entitySvc)
				}
				if len(gateResult.StructuralChecks) > 0 {
					structuralChecks = gateResult.StructuralChecks
				}
				if !gateResult.Satisfied {
					if gateResult.ReviewCapReached {
						_ = entitySvc.PersistFeatureBlockedReason(entityID, "", gateResult.Reason)
						chkStore := checkpoint.NewStore(entitySvc.Root())
						chk, chkErr := chkStore.Create(checkpoint.Record{
							Question: fmt.Sprintf("Feature %s has reached the review iteration cap (%d/%d). What should happen next?", entityID, feature.ReviewCycle, service.DefaultMaxReviewCycles),
							Context:  fmt.Sprintf("Review cycle: %d/%d", feature.ReviewCycle, service.DefaultMaxReviewCycles),
							Status:   checkpoint.StatusPending, CreatedAt: time.Now().UTC(), CreatedBy: "system",
						})
						resp := map[string]any{"error": gateResult.Reason, "blocked_reason": gateResult.Reason, "feature_id": entityID}
						if chkErr == nil {
							resp["checkpoint_id"] = chk.ID
						}
						return resp, nil
					}
					if !override {
						var nonTerm []service.TaskStatusPair
						if (currentStatus == "developing" && newStatus == "reviewing") || (currentStatus == "needs-rework" && newStatus == "reviewing") {
							nonTerm = nonTerminalTasksForFeature(entityID, entitySvc)
						}
						return service.GateFailureResponse(entityID, currentStatus, newStatus, gateResult, nonTerm), nil
					}
					if strings.TrimSpace(overrideReason) == "" {
						return map[string]any{"error": "override_reason is required when override is true"}, nil
					}
					if overridePolicy == "checkpoint" && checkpointStore != nil {
						chkR, chkErr := gate.HandleCheckpointOverride(gate.CheckpointOverrideParams{
							FeatureID: entityID, FromStatus: currentStatus, ToStatus: newStatus,
							GateDescription: gateResult.Reason, OverrideReason: overrideReason, AgentIdentity: "agent",
							CheckpointStore: checkpointStore,
						})
						if chkErr != nil {
							return nil, fmt.Errorf("creating checkpoint: %w", chkErr)
						}
						feature.Overrides = append(feature.Overrides, model.OverrideRecord{
							FromStatus: currentStatus, ToStatus: newStatus, Reason: overrideReason, Timestamp: time.Now(), CheckpointID: chkR.CheckpointID,
						})
						entitySvc.PersistFeatureOverrides(feature.ID, feature.Slug, feature.Overrides)
						return map[string]any{"checkpoint_created": true, "checkpoint_id": chkR.CheckpointID, "message": chkR.Message, "feature_id": entityID}, nil
					}
					feature.Overrides = append(feature.Overrides, model.OverrideRecord{
						FromStatus: currentStatus, ToStatus: newStatus, Reason: overrideReason, Timestamp: time.Now(),
					})
					entitySvc.PersistFeatureOverrides(feature.ID, feature.Slug, feature.Overrides)
				}
			}
		}
		var fromStatus string
		if pre, preErr := entitySvc.Get(entityType, entityID, ""); preErr == nil {
			fromStatus, _ = pre.State["status"].(string)
		}
		r, err := entitySvc.UpdateStatus(service.UpdateStatusInput{Type: entityType, ID: entityID, Status: newStatus})
		if err != nil {
			return entityTransitionError(entitySvc, entityType, entityID, newStatus, err), nil
		}
		if wt := r.WorktreeHookResult; wt != nil {
			if wt.Created {
				PushSideEffect(ctx, SideEffect{Type: SideEffectWorktreeCreated, EntityID: entityID, EntityType: entityType, Extra: map[string]string{"worktree_id": wt.WorktreeID, "branch": wt.Branch, "path": wt.Path}})
			}
			for _, u := range wt.UnblockedTasks {
				PushSideEffect(ctx, SideEffect{Type: SideEffectTaskUnblocked, EntityID: u.TaskID, EntityType: "task", FromStatus: u.PreviousStatus, ToStatus: u.Status, Trigger: fmt.Sprintf("dependencies resolved for %s", u.TaskID)})
			}
		}
		if entityType == "feature" && newStatus == string(model.FeatureStatusReviewing) {
			if err := entitySvc.IncrementFeatureReviewCycle(entityID, ""); err != nil {
				log.Printf("ERROR: failed to increment review cycle for %s: %v", entityID, err)
			}
		}
		resp := map[string]any{"entity": entityFullRecord(r.ID, r.Type, r.Slug, r.State)}
		if structuralChecks != nil {
			resp["structural_checks"] = structuralChecks
		}
		if entityType == "task" {
			isTerminal := newStatus == string(model.TaskStatusDone) || newStatus == string(model.TaskStatusNotPlanned) || newStatus == "duplicate"
			if isTerminal {
				parentFeatureID := ""
				if taskR, taskErr := entitySvc.Get("task", entityID, ""); taskErr == nil {
					parentFeatureID, _ = taskR.State["parent_feature"].(string)
				}
				if parentFeatureID != "" {
					featFrom := "developing"
					if preFeat, preFeatErr := entitySvc.Get("feature", parentFeatureID, ""); preFeatErr == nil {
						if s, _ := preFeat.State["status"].(string); s != "" {
							featFrom = s
						}
					}
					if advanced, advErr := entitySvc.MaybeAutoAdvanceFeature(parentFeatureID); advanced {
						PushSideEffect(ctx, SideEffect{Type: SideEffectFeatureAutoAdvanced, EntityID: parentFeatureID, EntityType: "feature", FromStatus: featFrom, ToStatus: string(model.FeatureStatusReviewing), Trigger: fmt.Sprintf("all tasks for %s are terminal", parentFeatureID)})
						if batchID := entitySvc.FeatureParentPlan(parentFeatureID); batchID != "" {
							if batchAdvanced, _ := entitySvc.MaybeAutoAdvancePlan(batchID); batchAdvanced {
								PushSideEffect(ctx, SideEffect{Type: SideEffectPlanAutoAdvanced, EntityID: batchID, EntityType: "batch", FromStatus: string(model.BatchStatusActive), ToStatus: string(model.BatchStatusDone), Trigger: fmt.Sprintf("all features for %s are terminal", batchID)})
							}
						}
					} else if advErr != nil {
						PushSideEffect(ctx, SideEffect{Type: SideEffectFeatureAutoAdvanced, EntityID: parentFeatureID, EntityType: "feature", Trigger: fmt.Sprintf("auto-advance failed: %v", advErr)})
					}
				}
			}
		}
		if entityType == "feature" {
			isTerminal := newStatus == string(model.FeatureStatusDone) || newStatus == string(model.FeatureStatusSuperseded) || newStatus == string(model.FeatureStatusCancelled)
			if isTerminal {
				if batchID := entitySvc.FeatureParentPlan(entityID); batchID != "" {
					if batchAdvanced, _ := entitySvc.MaybeAutoAdvancePlan(batchID); batchAdvanced {
						PushSideEffect(ctx, SideEffect{Type: SideEffectPlanAutoAdvanced, EntityID: batchID, EntityType: "batch", FromStatus: string(model.BatchStatusActive), ToStatus: string(model.BatchStatusDone), Trigger: fmt.Sprintf("all features for %s are terminal", batchID)})
					}
				}
			}
		}
		if _, err := entityCommitFunc(".", fmt.Sprintf("workflow(%s): transition %s → %s", entityID, fromStatus, newStatus)); err != nil {
			log.Printf("WARNING: commit after transition failed: %v", err)
		}
		return resp, nil
	}
}

func entityAdvanceFeature(ctx context.Context, entitySvc *service.EntityService, docSvc *service.DocumentService, entityID, targetStatus string, override bool, overrideReason string, gateRouter *gate.GateRouter, checkpointStore *checkpoint.Store, requiresHumanReview func() bool) (any, error) {
	getR, err := entitySvc.Get("feature", entityID, "")
	if err != nil {
		return nil, fmt.Errorf("loading feature %s: %w", entityID, err)
	}
	feature := featureFromState(getR.ID, getR.Slug, getR.State)
	startStatus := string(feature.Status)
	advCfg := &service.AdvanceConfig{RequiresHumanReview: requiresHumanReview}
	if gateRouter != nil {
		advCfg.CheckGate = func(from, to string, f *model.Feature, ds *service.DocumentService, es *service.EntityService) service.GateResult {
			routerCtx := buildGateEvalContext(f, ds, es)
			routerResult := gateRouter.CheckGate(from, to, routerCtx)
			if routerResult.Source == "registry" {
				return service.GateResult{Stage: routerResult.Stage, Satisfied: routerResult.Satisfied, Reason: routerResult.Reason}
			}
			return service.CheckTransitionGate(from, to, f, ds, es)
		}
		advCfg.OverridePolicy = func(to string) string { return gateRouter.OverridePolicy(to) }
		if checkpointStore != nil {
			advCfg.OnCheckpoint = func(featureID, fromStatus, toStatus, gateReason, overrideReason string) (string, error) {
				r, err := gate.HandleCheckpointOverride(gate.CheckpointOverrideParams{FeatureID: featureID, FromStatus: fromStatus, ToStatus: toStatus, GateDescription: gateReason, OverrideReason: overrideReason, AgentIdentity: "agent", CheckpointStore: checkpointStore})
				if err != nil {
					return "", err
				}
				return r.CheckpointID, nil
			}
		}
	}
	advResult, err := service.AdvanceFeatureStatus(feature, targetStatus, entitySvc, docSvc, override, overrideReason, advCfg)
	if err != nil {
		return nil, fmt.Errorf("advance feature %s: %w", entityID, err)
	}
	stagesSkipped := len(advResult.AdvancedThrough)
	resp := map[string]any{"status": advResult.FinalStatus, "advanced_through": advResult.AdvancedThrough}
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
		stageWord := "stages"
		if stagesSkipped == 1 {
			stageWord = "stage"
		}
		resp["stopped_reason"] = advResult.StoppedReason
		resp["message"] = fmt.Sprintf("Advanced from %s to %s (%d %s). Stopped: %s", startStatus, advResult.FinalStatus, stagesSkipped, stageWord, advResult.StoppedReason)
	} else if stagesSkipped == 0 {
		resp["message"] = fmt.Sprintf("Already at %s", advResult.FinalStatus)
	} else {
		resp["message"] = fmt.Sprintf("Advanced from %s to %s (skipped %d stages)", startStatus, advResult.FinalStatus, stagesSkipped)
	}
	return resp, nil
}

func featureFromState(entityID, slug string, state map[string]any) *model.Feature {
	rc, _ := state["review_cycle"].(int)
	br, _ := state["blocked_reason"].(string)
	return &model.Feature{
		ID: entityID, Slug: slug, Parent: entityStateStr(state, "parent"),
		Status: model.FeatureStatus(entityStateStr(state, "status")), ReviewCycle: rc,
		BlockedReason: br, Design: entityStateStr(state, "design"), Spec: entityStateStr(state, "spec"),
		DevPlan: entityStateStr(state, "dev_plan"), Overrides: overridesFromState(state),
	}
}

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
		result = append(result, model.OverrideRecord{
			FromStatus: entityStateStr(m, "from_status"), ToStatus: entityStateStr(m, "to_status"),
			Reason: entityStateStr(m, "reason"), Timestamp: time.Now(),
		})
	}
	return result
}

func buildGateEvalContext(feature *model.Feature, docSvc *service.DocumentService, entitySvc *service.EntityService) gate.PrereqEvalContext {
	return gate.PrereqEvalContext{Feature: feature, DocSvc: &gateDocAdapter{svc: docSvc}, EntitySvc: &gateEntityAdapter{svc: entitySvc}}
}

type gateDocAdapter struct{ svc *service.DocumentService }

func (a *gateDocAdapter) GetDocument(id string, loadContent bool) (*gate.DocumentRecord, error) {
	r, err := a.svc.GetDocument(id, loadContent)
	if err != nil {
		return nil, err
	}
	return &gate.DocumentRecord{ID: r.ID, Status: r.Status, Type: r.Type, Owner: r.Owner}, nil
}

func (a *gateDocAdapter) ListDocuments(filters gate.DocumentFilters) ([]*gate.DocumentRecord, error) {
	rs, err := a.svc.ListDocuments(service.DocumentFilters{Owner: filters.Owner, Type: filters.Type, Status: filters.Status})
	if err != nil {
		return nil, err
	}
	out := make([]*gate.DocumentRecord, len(rs))
	for i, r := range rs {
		out[i] = &gate.DocumentRecord{ID: r.ID, Status: r.Status, Type: r.Type, Owner: r.Owner}
	}
	return out, nil
}

type gateEntityAdapter struct{ svc *service.EntityService }

func (a *gateEntityAdapter) List(entityType string) ([]gate.EntityResult, error) {
	rs, err := a.svc.List(entityType)
	if err != nil {
		return nil, err
	}
	out := make([]gate.EntityResult, len(rs))
	for i, r := range rs {
		out[i] = gate.EntityResult{ID: r.ID, State: r.State}
	}
	return out, nil
}

var phase2Statuses = map[string]bool{
	"proposed": true, "designing": true, "specifying": true, "dev-planning": true,
	"developing": true, "reviewing": true, "needs-rework": true, "done": true,
}

func isPhase2Transition(from, to string) bool { return phase2Statuses[from] && phase2Statuses[to] }

func nonTerminalTasksForFeature(featureID string, entitySvc *service.EntityService) []service.TaskStatusPair {
	tasks, err := entitySvc.List("task")
	if err != nil {
		return nil
	}
	var result []service.TaskStatusPair
	for _, t := range tasks {
		pf, _ := t.State["parent_feature"].(string)
		if pf != featureID {
			continue
		}
		status, _ := t.State["status"].(string)
		if !validate.IsTaskDependencySatisfied(status) {
			result = append(result, service.TaskStatusPair{ID: t.ID, Status: status})
		}
	}
	return result
}

func entityTransitionError(entitySvc *service.EntityService, entityType, entityID, requested string, cause error) map[string]any {
	details := map[string]any{"requested_status": requested}
	if currentStatus, err := entityCurrentStatus(entitySvc, entityType, entityID); err == nil {
		details["current_status"] = currentStatus
		if kind, ok := entityKindFromType(entityType); ok {
			if next := validate.NextStates(kind, currentStatus); len(next) > 0 {
				sort.Strings(next)
				details["valid_transitions"] = next
			}
		}
	}
	return map[string]any{"error": map[string]any{"code": "invalid_transition", "message": cause.Error(), "details": details}}
}

func entityCurrentStatus(entitySvc *service.EntityService, entityType, entityID string) (string, error) {
	switch entityType {
	case "batch", "plan":
		if r, err := entitySvc.GetBatch(entityID); err == nil {
			status, _ := r.State["status"].(string)
			return status, nil
		}
		r, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			return "", err
		}
		status, _ := r.State["status"].(string)
		return status, nil
	case "strategic-plan":
		r, err := entitySvc.GetStrategicPlan(entityID)
		if err != nil {
			return "", err
		}
		status, _ := r.State["status"].(string)
		return status, nil
	default:
		r, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			return "", err
		}
		status, _ := r.State["status"].(string)
		return status, nil
	}
}

func entityKindFromType(entityType string) (validate.EntityKind, bool) {
	switch entityType {
	case "batch", "plan":
		return validate.EntityBatch, true
	case "strategic-plan":
		return validate.EntityStrategicPlan, true
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
		return "", false
	}
}

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
	case model.IsBatchID(entityID):
		// Distinguish strategic plans (P...) from batches (B...).
		prefix, _, _ := model.ParseBatchID(entityID)
		if prefix == "P" {
			return "strategic-plan", true
		}
		return "batch", true
	default:
		return "", false
	}
}

func entityArgStr(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

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

func entityStateStr(state map[string]any, key string) string {
	if state == nil {
		return ""
	}
	s, _ := state[key].(string)
	return s
}

func entityFullRecord(entityID, entityType, slug string, state map[string]any) map[string]any {
	out := make(map[string]any, len(state)+6)
	for k, v := range state {
		out[k] = v
	}
	displayID := id.FormatFullDisplay(entityID)
	if did, ok := state["display_id"].(string); ok && service.IsFeatureDisplayID(did) {
		displayID = did
	}
	name, _ := state["name"].(string)
	out["display_id"] = displayID
	out["id"] = entityID
	out["type"] = entityType
	out["slug"] = slug
	out["entity_ref"] = id.FormatEntityRef(displayID, slug, name)
	return out
}
