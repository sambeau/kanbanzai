// Package testexternal verifies that the kbzschema public package can be
// imported and used by an external Go module without pulling in any
// kanbanzai/internal/ packages.
//
// This file is compiled as part of the AC-13 external compilation check.
// It is intentionally minimal: it just exercises the public API surface.
package main

import (
	"fmt"

	"kanbanzai/kbzschema"
)

func main() {
	// Exercise exported constants.
	_ = kbzschema.SchemaVersion
	_ = kbzschema.PlanStatusProposed
	_ = kbzschema.PlanStatusDesigning
	_ = kbzschema.PlanStatusActive
	_ = kbzschema.PlanStatusDone
	_ = kbzschema.PlanStatusSuperseded
	_ = kbzschema.PlanStatusCancelled
	_ = kbzschema.FeatureStatusProposed
	_ = kbzschema.FeatureStatusDeveloping
	_ = kbzschema.TaskStatusQueued
	_ = kbzschema.TaskStatusDone
	_ = kbzschema.BugStatusReported
	_ = kbzschema.BugStatusClosed
	_ = kbzschema.SeverityLow
	_ = kbzschema.SeverityCritical
	_ = kbzschema.PriorityHigh
	_ = kbzschema.BugTypeImplementationDefect
	_ = kbzschema.DecisionStatusAccepted
	_ = kbzschema.DocTypeDesign
	_ = kbzschema.DocStatusDraft
	_ = kbzschema.KnowledgeStatusContributed
	_ = kbzschema.KnowledgeTier2
	_ = kbzschema.KnowledgeTier3
	_ = kbzschema.CheckpointStatusPending
	_ = kbzschema.CheckpointStatusResponded

	// Exercise exported struct types.
	var plan kbzschema.Plan
	plan.Status = kbzschema.PlanStatusActive
	_ = plan

	var feature kbzschema.Feature
	feature.Status = kbzschema.FeatureStatusDeveloping
	_ = feature

	var task kbzschema.Task
	task.Status = kbzschema.TaskStatusReady
	_ = task

	var bug kbzschema.Bug
	bug.Severity = kbzschema.SeverityHigh
	bug.Priority = kbzschema.PriorityMedium
	_ = bug

	var decision kbzschema.Decision
	decision.Status = kbzschema.DecisionStatusAccepted
	_ = decision

	var doc kbzschema.DocumentRecord
	doc.Type = kbzschema.DocTypeSpecification
	doc.Status = kbzschema.DocStatusApproved
	_ = doc

	var ke kbzschema.KnowledgeEntry
	ke.Status = kbzschema.KnowledgeStatusConfirmed
	ke.Tier = kbzschema.KnowledgeTier2
	_ = ke

	var chk kbzschema.HumanCheckpoint
	chk.Status = kbzschema.CheckpointStatusPending
	_ = chk

	var cfg kbzschema.ProjectConfig
	cfg.SchemaVersion = kbzschema.SchemaVersion
	_ = cfg

	// Exercise NewReader (expected to fail for a non-existent path, but
	// it must compile and the error must be non-nil).
	r, err := kbzschema.NewReader("/tmp/nonexistent-kanbanzai-repo")
	if err == nil {
		// If it somehow succeeds, exercise the read methods.
		_, _ = r.ListPlans()
	}

	// Exercise GenerateSchema.
	schemaBytes, err := kbzschema.GenerateSchema()
	if err != nil {
		fmt.Printf("GenerateSchema error: %v\n", err)
		return
	}
	if len(schemaBytes) == 0 {
		fmt.Println("GenerateSchema returned empty bytes")
	}
}
