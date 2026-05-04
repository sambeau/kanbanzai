// Package validate validator_dispatch.go — validator dispatch abstraction for fast-track architecture.
//
// The ValidatorDispatcher interface decouples validator dispatch from the underlying
// agent invocation mechanism. Today, SpawnAgentDispatcher generates handoff prompts
// for the spawn_agent tool. When P44 model routing arrives, a different implementation
// can be injected without changing any code that calls ValidatorDispatcher.Dispatch.
//
// Spec: work/P43-fast-track-architecture/P43-spec-fast-track-architecture.md §1.2.1.9 (REQ-SESS-004)
// Dev-plan: work/P43-fast-track-architecture/P43-dev-plan-fast-track-architecture.md §1.2.10

package validate

import (
	"context"
	"fmt"
	"strings"
)

// Verdict is the overall result of a validator run.
type Verdict string

const (
	VerdictPass         Verdict = "pass"
	VerdictPassWithNotes Verdict = "pass_with_notes"
	VerdictFail         Verdict = "fail"
)

// ValidatorContext carries everything a validator needs to know about the
// document under validation and its surrounding context.
type ValidatorContext struct {
	// DocumentPath is the filesystem path to the document being validated.
	DocumentPath string
	// DocumentType is the type of document (specification, dev-plan, design).
	DocumentType string
	// ParentDocPath is the filesystem path to the parent document (e.g., the
	// design for a spec, or the spec for a dev-plan). May be empty.
	ParentDocPath string
	// RubricPath is the filesystem path to the checklist/rubric file the
	// validator should apply.
	RubricPath string
	// FeatureID is the feature entity this validation is scoped to.
	FeatureID string
}

// ValidatorSummary is the compact result returned to the orchestrator.
// It fits within a single tool response (REQ-SESS-003). The full report is
// retrieved via doc(action: "content") for human audit (REQ-SESS-002).
type ValidatorSummary struct {
	// Verdict is the overall result: pass, pass_with_notes, or fail.
	Verdict Verdict
	// BlockingCount is the number of blocking findings.
	BlockingCount int
	// NonBlockingCount is the number of non-blocking findings.
	NonBlockingCount int
	// EvidenceScore is a coarse 0.0–1.0 measure of how well-evidenced the
	// validator's conclusions are. Higher is better.
	EvidenceScore float64
	// ReportDocID is the document record ID of the full report written to the
	// document store by the validator.
	ReportDocID string
}

// ValidatorDispatcher dispatches a validator sub-agent for a document.
//
// The interface abstracts the invocation mechanism so that today's
// spawn_agent-based dispatch can be replaced by P44 model routing without
// changes to callers. Implementations receive the role, skill, and validation
// context, and return a summary for the orchestrator.
//
// The full report must be written to the document store by the implementation;
// the caller does not manage report storage.
type ValidatorDispatcher interface {
	// Dispatch triggers a validator run. The role and skill identify which
	// validator to invoke (e.g. role="spec-validator", skill="validate-spec").
	// vctx carries the document paths and feature context.
	//
	// On success the full report is registered in the document store and
	// ValidatorSummary.ReportDocID holds its document record ID.
	Dispatch(ctx context.Context, role, skill string, vctx ValidatorContext) (ValidatorSummary, error)
}

// SpawnAgentDispatcher implements ValidatorDispatcher by generating a handoff
// prompt for the spawn_agent tool. It does NOT invoke spawn_agent itself —
// the orchestrator (caller) passes the generated prompt to its spawn_agent tool.
//
// The prompt assembly follows the same fresh-session principle as handoff:
// only the document content, parent document, and rubric checklist are included.
// The conversation that produced the document is NOT included (REQ-SESS-001).
type SpawnAgentDispatcher struct {
	// DocContentFunc reads a file's content. May be nil; falls back to a
	// placeholder when the file cannot be read.
	DocContentFunc func(path string) (string, error)
	// RegisterReportFunc registers the validator's full report as a document
	// record and returns the doc ID. Must be non-nil.
	RegisterReportFunc func(reportPath, reportContent, docType, title string, featureID string) (string, error)
}

// NewSpawnAgentDispatcher creates a SpawnAgentDispatcher with the required
// document registration callback. docContentFunc may be nil.
func NewSpawnAgentDispatcher(registerFn func(string, string, string, string, string) (string, error)) *SpawnAgentDispatcher {
	return &SpawnAgentDispatcher{
		RegisterReportFunc: registerFn,
	}
}

// Dispatch generates a validator handoff prompt for spawn_agent.
//
// The prompt includes:
//   - The validator's role identity and skill procedure
//   - The document under validation (full content)
//   - The parent document (full content, if provided)
//   - The rubric checklist
//   - Instructions to produce a summary and write the full report
//
// It returns a ValidatorSummary with ReportDocID empty — the caller is
// expected to use the returned prompt with spawn_agent, and the spawned
// agent will register the report and return the summary.
func (d *SpawnAgentDispatcher) Dispatch(ctx context.Context, role, skill string, vctx ValidatorContext) (ValidatorSummary, error) {
	if d.RegisterReportFunc == nil {
		return ValidatorSummary{}, fmt.Errorf("SpawnAgentDispatcher: RegisterReportFunc is nil")
	}

	prompt := d.buildPrompt(role, skill, vctx)

	// The prompt is the primary output. The orchestrator passes it to
	// spawn_agent. We also produce a provisional summary that the
	// orchestrator can use while waiting for the sub-agent to complete.
	return ValidatorSummary{
		Verdict:          VerdictPass, // provisional; sub-agent overrides
		BlockingCount:    0,
		NonBlockingCount: 0,
		EvidenceScore:    0.0,
		ReportDocID:      "",
	}, fmt.Errorf("not yet implemented: spawn_agent dispatch requires the orchestrator to pass the generated prompt to spawn_agent. Prompt:\n%s", prompt)
}

// buildPrompt assembles the validator handoff prompt.
func (d *SpawnAgentDispatcher) buildPrompt(role, skill string, vctx ValidatorContext) string {
	var sb strings.Builder

	sb.WriteString("# Validator Dispatch\n\n")
	sb.WriteString(fmt.Sprintf("**Role:** %s\n\n", role))
	sb.WriteString(fmt.Sprintf("**Skill:** %s\n\n", skill))
	sb.WriteString(fmt.Sprintf("**Document under validation:** %s (%s)\n\n", vctx.DocumentPath, vctx.DocumentType))

	if vctx.ParentDocPath != "" {
		sb.WriteString(fmt.Sprintf("**Parent document:** %s\n\n", vctx.ParentDocPath))
	}
	if vctx.FeatureID != "" {
		sb.WriteString(fmt.Sprintf("**Feature:** %s\n\n", vctx.FeatureID))
	}

	// Document content
	sb.WriteString("## Document Under Validation\n\n")
	if content, err := d.readDoc(vctx.DocumentPath); err == nil {
		sb.WriteString(content)
	} else {
		sb.WriteString(fmt.Sprintf("<!-- could not read %s: %v -->\n\n", vctx.DocumentPath, err))
	}

	// Parent document content
	if vctx.ParentDocPath != "" {
		sb.WriteString("\n## Parent Document\n\n")
		if content, err := d.readDoc(vctx.ParentDocPath); err == nil {
			sb.WriteString(content)
		} else {
			sb.WriteString(fmt.Sprintf("<!-- could not read %s: %v -->\n\n", vctx.ParentDocPath, err))
		}
	}

	// Rubric content
	if vctx.RubricPath != "" {
		sb.WriteString("\n## Validation Rubric\n\n")
		if content, err := d.readDoc(vctx.RubricPath); err == nil {
			sb.WriteString(content)
		} else {
			sb.WriteString(fmt.Sprintf("<!-- could not read %s: %v -->\n\n", vctx.RubricPath, err))
		}
	}

	// Instructions
	sb.WriteString("\n## Instructions\n\n")
	sb.WriteString("1. Adopt the role identity and follow the skill procedure listed above.\n")
	sb.WriteString("2. Validate the document under validation against the rubric.\n")
	sb.WriteString("3. Classify each finding as **blocking** or **non-blocking**.\n")
	sb.WriteString("4. Produce two outputs:\n")
	sb.WriteString("   - A **summary** (verdict, blocking/non-blocking counts, evidence score 0.0–1.0).\n")
	sb.WriteString("   - A **full report** with detailed per-check analysis and evidence citations.\n")
	sb.WriteString("5. Write the full report to a file in the feature's worktree.\n")
	sb.WriteString("6. Register the report with `doc(action: \"register\", type: \"report\", ...)`.\n")
	sb.WriteString("7. Return the summary with the report's document ID.\n")

	return sb.String()
}

// readDoc reads a document's content, using the configured DocContentFunc if
// available, otherwise returning a placeholder.
func (d *SpawnAgentDispatcher) readDoc(path string) (string, error) {
	if d.DocContentFunc != nil {
		return d.DocContentFunc(path)
	}
	return "", fmt.Errorf("DocContentFunc not configured; cannot read %s", path)
}
