package service

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/structural"
	"github.com/sambeau/kanbanzai/internal/validate"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// GateResult describes whether a stage gate prerequisite is satisfied.
type GateResult struct {
	Stage            string // the lifecycle stage being checked
	Satisfied        bool
	Reason           string                   // human-readable explanation
	StructuralChecks []structural.CheckResult // populated when structural checks ran
	ReviewCapReached bool                     // true when the review iteration cap was reached
	VerifierPrompt   string                   // handoff prompt for verifier sub-agent (verified→closed gate)
	NeedsVerifier    bool                     // true when the gate requires verifier sub-agent dispatch
}

// stageDocMapping maps feature lifecycle stages to their required document types.
var stageDocMapping = map[string]string{
	string(model.FeatureStatusDesigning):   string(model.DocumentTypeDesign),
	string(model.FeatureStatusSpecifying):  string(model.DocumentTypeSpec),
	string(model.FeatureStatusDevPlanning): string(model.DocumentTypeDevPlan),
}

// stageDocField maps feature lifecycle stages to the Feature struct field
// that holds a direct reference to the relevant document record.
var stageDocField = map[string]string{
	string(model.FeatureStatusDesigning):   "design",
	string(model.FeatureStatusSpecifying):  "spec",
	string(model.FeatureStatusDevPlanning): "dev_plan",
}

// featureDocRef returns the document record ID referenced by the feature's
// own field for the given stage, or empty string if none.
func featureDocRef(feature *model.Feature, stage string) string {
	switch stage {
	case string(model.FeatureStatusDesigning):
		return feature.Design
	case string(model.FeatureStatusSpecifying):
		return feature.Spec
	case string(model.FeatureStatusDevPlanning):
		return feature.DevPlan
	default:
		return ""
	}
}

// CheckFeatureGate checks the prerequisite for a single stage gate.
// It returns a GateResult indicating whether the gate is satisfied and why.
func CheckFeatureGate(stage string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult {
	// The reviewing stage is never skippable.
	if stage == "reviewing" {
		return GateResult{
			Stage:     stage,
			Satisfied: false,
			Reason:    "reviewing stage cannot be skipped",
		}
	}

	// The developing stage requires at least one child task.
	if stage == string(model.FeatureStatusDeveloping) {
		return checkDevelopingGate(feature, entitySvc)
	}

	// Document-driven gates.
	docType, ok := stageDocMapping[stage]
	if !ok {
		return GateResult{
			Stage:     stage,
			Satisfied: false,
			Reason:    fmt.Sprintf("unknown stage %q", stage),
		}
	}

	return checkDocumentGate(stage, docType, feature, docSvc, entitySvc)
}

// CheckFeatureGates checks all document-driven stage gates for a feature.
// Returns a GateResult for each skippable stage in lifecycle order.
func CheckFeatureGates(feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) []GateResult {
	stages := []string{
		string(model.FeatureStatusDesigning),
		string(model.FeatureStatusSpecifying),
		string(model.FeatureStatusDevPlanning),
		string(model.FeatureStatusDeveloping),
		"reviewing",
	}

	results := make([]GateResult, 0, len(stages))
	for _, stage := range stages {
		results = append(results, CheckFeatureGate(stage, feature, docSvc, entitySvc))
	}
	return results
}

// checkDocumentGate checks whether an approved document of the given type
// exists for the feature, following the three-level lookup order:
//  1. Feature's own document field reference
//  2. Documents owned by the feature
//  3. Documents owned by the parent plan
func checkDocumentGate(stage, docType string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult {
	fieldName := stageDocField[stage]

	// 1. Check feature's own document field reference.
	docRef := featureDocRef(feature, stage)
	if docRef != "" {
		doc, err := docSvc.GetDocument(docRef, false)
		if err == nil && doc.Status == string(model.DocumentStatusApproved) {
			return GateResult{
				Stage:     stage,
				Satisfied: true,
				Reason:    fmt.Sprintf("approved %s document referenced by feature.%s: %s", docType, fieldName, docRef),
			}
		}
	}

	// 2. Check documents owned by the feature.
	featureDocs, err := docSvc.ListDocuments(DocumentFilters{
		Owner:  feature.ID,
		Type:   docType,
		Status: string(model.DocumentStatusApproved),
	})
	if err == nil && len(featureDocs) > 0 {
		return GateResult{
			Stage:     stage,
			Satisfied: true,
			Reason:    fmt.Sprintf("approved %s document owned by feature: %s", docType, featureDocs[0].ID),
		}
	}

	// 3. Check documents owned by the parent batch/plan.
	if feature.Parent != "" {
		parentDocs, err := docSvc.ListDocuments(DocumentFilters{
			Owner:  feature.Parent,
			Type:   docType,
			Status: string(model.DocumentStatusApproved),
		})
		if err == nil && len(parentDocs) > 0 {
			return GateResult{
				Stage:     stage,
				Satisfied: true,
				Reason:    fmt.Sprintf("approved %s document owned by parent %s: %s", docType, feature.Parent, parentDocs[0].ID),
			}
		}

		// 4. Check documents owned by the grandparent plan (batch → plan chain).
		if batch, batchErr := entitySvc.GetBatch(feature.Parent); batchErr == nil {
			if batch.Parent != "" {
				gpDocs, gpErr := docSvc.ListDocuments(DocumentFilters{
					Owner:  batch.Parent,
					Type:   docType,
					Status: string(model.DocumentStatusApproved),
				})
				if gpErr == nil && len(gpDocs) > 0 {
					return GateResult{
						Stage:     stage,
						Satisfied: true,
						Reason:    fmt.Sprintf("approved %s document owned by grandparent plan %s: %s", docType, batch.Parent, gpDocs[0].ID),
					}
				}
			}
		}
	}

	return GateResult{
		Stage:     stage,
		Satisfied: false,
		Reason:    fmt.Sprintf("no approved %s document found", docType),
	}
}

// CheckTransitionGate checks the gate prerequisite for a specific (from, to)
// feature lifecycle transition. It returns a satisfied GateResult for ungated
// transitions (terminal targets, Phase 1 transitions, proposed→designing,
// reviewing→needs-rework) and an unsatisfied GateResult when prerequisites
// are not met. This is the primary entry point for mandatory gate enforcement
// (FR-001 through FR-010).
func CheckTransitionGate(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult {
	// Terminal state transitions are always ungated (FR-002).
	if to == string(model.FeatureStatusSuperseded) || to == string(model.FeatureStatusCancelled) {
		return GateResult{Stage: to, Satisfied: true}
	}

	transition := from + "→" + to
	switch transition {
	case string(model.FeatureStatusProposed) + "→" + string(model.FeatureStatusDesigning):
		// proposed→designing: ungated by design (FR-003)
		return GateResult{Stage: to, Satisfied: true}

	case string(model.FeatureStatusDesigning) + "→" + string(model.FeatureStatusSpecifying):
		// designing→specifying: requires approved design document (FR-004)
		docResult := checkDocumentGate(string(model.FeatureStatusDesigning), string(model.DocumentTypeDesign), feature, docSvc, entitySvc)
		if !docResult.Satisfied {
			return docResult
		}
		structChecks, hardFail := runStructuralChecksForGate(from, to, feature, docSvc)
		docResult.StructuralChecks = structChecks
		if hardFail {
			docResult.Satisfied = false
			docResult.Reason = buildStructuralFailureReason(structChecks)
		}
		return docResult

	case string(model.FeatureStatusSpecifying) + "→" + string(model.FeatureStatusDevPlanning):
		// specifying→dev-planning: requires approved specification document (FR-005)
		docResult := checkDocumentGate(string(model.FeatureStatusSpecifying), string(model.DocumentTypeSpec), feature, docSvc, entitySvc)
		if !docResult.Satisfied {
			return docResult
		}
		structChecks, hardFail := runStructuralChecksForGate(from, to, feature, docSvc)
		docResult.StructuralChecks = structChecks
		if hardFail {
			docResult.Satisfied = false
			docResult.Reason = buildStructuralFailureReason(structChecks)
		}
		return docResult

	case string(model.FeatureStatusDevPlanning) + "→" + string(model.FeatureStatusDeveloping):
		// dev-planning→developing: requires approved dev-plan AND at least one child task (FR-006)
		docResult := checkDocumentGate(string(model.FeatureStatusDevPlanning), string(model.DocumentTypeDevPlan), feature, docSvc, entitySvc)
		if !docResult.Satisfied {
			return docResult
		}
		structChecks, hardFail := runStructuralChecksForGate(from, to, feature, docSvc)
		docResult.StructuralChecks = structChecks
		if hardFail {
			docResult.Satisfied = false
			docResult.Reason = buildStructuralFailureReason(structChecks)
			return docResult
		}
		taskResult := checkDevelopingGate(feature, entitySvc)
		if !taskResult.Satisfied {
			return taskResult
		}
		return docResult

	case string(model.FeatureStatusDeveloping) + "→" + string(model.FeatureStatusReviewing):
		// developing→reviewing: all child tasks must be in terminal state (FR-007)
		return checkAllTasksTerminal(feature, entitySvc)

	case string(model.FeatureStatusReviewing) + "→" + string(model.FeatureStatusDone):
		// reviewing→done: a review report document must be registered (FR-008)
		return checkReviewReportExists(feature, docSvc)

	case string(model.FeatureStatusReviewing) + "→" + string(model.FeatureStatusNeedsRework):
		// Check review iteration cap (FR-005).
		if feature.ReviewCycle >= DefaultMaxReviewCycles {
			return GateResult{
				Stage:            to,
				Satisfied:        false,
				Reason:           fmt.Sprintf("Review iteration cap reached (%d/%d). Human decision required: accept with known issues, rework with revised scope, or cancel.", feature.ReviewCycle, DefaultMaxReviewCycles),
				ReviewCapReached: true,
			}
		}
		// reviewing→needs-rework: ungated by design (FR-003)
		return GateResult{Stage: to, Satisfied: true}

	case string(model.FeatureStatusNeedsRework) + "→" + string(model.FeatureStatusDeveloping):
		// needs-rework→developing: at least one non-terminal child task must exist (FR-009)
		return checkReworkTaskExists(feature, entitySvc)

	case string(model.FeatureStatusNeedsRework) + "→" + string(model.FeatureStatusReviewing):
		// needs-rework→reviewing: all child tasks must be in terminal state (FR-010)
		return checkAllTasksTerminal(feature, entitySvc)

	default:
		// All other transitions (Phase 1, backward, unknown) are ungated.
		return GateResult{Stage: to, Satisfied: true}
	}
}

// buildStructuralFailureReason builds a gate failure reason from hard_gate structural check failures.
func buildStructuralFailureReason(checks []structural.CheckResult) string {
	var msgs []string
	for _, c := range checks {
		if !c.Passed && c.Mode == "hard_gate" {
			msg := fmt.Sprintf("%s check failed for %s", c.CheckType, c.DocumentType)
			if len(c.Details) > 0 {
				msg += ": " + strings.Join(c.Details, "; ")
			}
			msgs = append(msgs, msg)
		}
	}
	if len(msgs) == 0 {
		return "structural check failed"
	}
	return strings.Join(msgs, "; ")
}

// checkAllTasksTerminal verifies that all child tasks of the feature are in a
// terminal state (done, not-planned, or duplicate). Used by developing→reviewing
// (FR-007) and needs-rework→reviewing (FR-010).
func checkAllTasksTerminal(feature *model.Feature, entitySvc *EntityService) GateResult {
	tasks, err := entitySvc.List("task")
	if err != nil {
		return GateResult{
			Satisfied: false,
			Reason:    fmt.Sprintf("error listing tasks: %v", err),
		}
	}

	termStates := validate.DependencyTerminalStates()
	var nonTerminal []string
	for _, t := range tasks {
		if stringFromState(t.State, "parent_feature") != feature.ID {
			continue
		}
		status := stringFromState(t.State, "status")
		if _, ok := termStates[status]; !ok {
			nonTerminal = append(nonTerminal, fmt.Sprintf("%s (%s)", t.ID, status))
		}
	}

	if len(nonTerminal) == 0 {
		return GateResult{
			Satisfied: true,
			Reason:    "all child tasks are in terminal state",
		}
	}

	return GateResult{
		Satisfied: false,
		Reason:    fmt.Sprintf("non-terminal child tasks: %s", strings.Join(nonTerminal, ", ")),
	}
}

// checkAllTasksHaveVerification verifies that all child tasks of the feature
// have a non-empty verification field. Returns nil if no child tasks exist
// (vacuously true). Used as a prereq helper for agentic review auto-advance.
func checkAllTasksHaveVerification(feature *model.Feature, entitySvc *EntityService) error {
	tasks, err := entitySvc.List("task")
	if err != nil {
		return fmt.Errorf("error listing tasks: %v", err)
	}

	for _, t := range tasks {
		if stringFromState(t.State, "parent_feature") != feature.ID {
			continue
		}
		if stringFromState(t.State, "verification") == "" {
			return fmt.Errorf("task %s has no recorded verification", t.ID)
		}
	}

	return nil
}

// checkReviewReportExists verifies that at least one report document is
// registered and owned by the feature. The report need not be approved.
// Used by reviewing→done (FR-008).
func checkReviewReportExists(feature *model.Feature, docSvc *DocumentService) GateResult {
	docs, err := docSvc.ListDocuments(DocumentFilters{
		Owner: feature.ID,
		Type:  string(model.DocumentTypeReport),
	})
	if err == nil && len(docs) > 0 {
		return GateResult{
			Satisfied: true,
			Reason:    fmt.Sprintf("review report document found: %s", docs[0].ID),
		}
	}

	return GateResult{
		Satisfied: false,
		Reason:    "no review report document registered for this feature",
	}
}

// checkReworkTaskExists verifies that at least one non-terminal child task
// exists for the feature. Used by needs-rework→developing (FR-009).
func checkReworkTaskExists(feature *model.Feature, entitySvc *EntityService) GateResult {
	tasks, err := entitySvc.List("task")
	if err != nil {
		return GateResult{
			Satisfied: false,
			Reason:    fmt.Sprintf("error listing tasks: %v", err),
		}
	}

	termStates := validate.DependencyTerminalStates()
	for _, t := range tasks {
		if stringFromState(t.State, "parent_feature") != feature.ID {
			continue
		}
		status := stringFromState(t.State, "status")
		if _, ok := termStates[status]; !ok {
			return GateResult{
				Satisfied: true,
				Reason:    fmt.Sprintf("non-terminal rework task found: %s (%s)", t.ID, status),
			}
		}
	}

	return GateResult{
		Satisfied: false,
		Reason:    "no non-terminal rework tasks found; create a rework task before resuming development",
	}
}

// CheckBugTransitionGate checks the gate prerequisite for a specific (from, to)
// bug lifecycle transition. It returns a satisfied GateResult for ungated
// transitions (terminal targets, pre-in-progress transitions) and an
// unsatisfied GateResult when prerequisites are not met (FR-005 through FR-011).
func CheckBugTransitionGate(from, to string, bug *model.Bug, docSvc *DocumentService, entitySvc *EntityService) GateResult {
	// Terminal state transitions are always ungated.
	if to == string(model.BugStatusClosed) || to == string(model.BugStatusDuplicate) || to == string(model.BugStatusNotPlanned) {
		return GateResult{Stage: to, Satisfied: true}
	}

	transition := from + "→" + to
	switch transition {
	case string(model.BugStatusInProgress) + "→" + string(model.BugStatusNeedsReview):
		// FR-006: in-progress→needs-review requires worktree commits beyond base.
		return checkBugWorktreeHasCommits(bug, docSvc)

	case string(model.BugStatusNeedsReview) + "→" + string(model.BugStatusVerifying):
		// FR-007: needs-review→verifying requires review report AND passing tests.
		return checkBugReviewReportAndTests(bug, docSvc)

	case string(model.BugStatusNeedsReview) + "→" + string(model.BugStatusNeedsRework):
		// FR-008: needs-review→needs-rework (first step back to in-progress).
		// Increment review_cycle, check cap, escalate if reached.
		return checkBugReviewCap(bug, entitySvc)

	case string(model.BugStatusVerifying) + "→" + string(model.BugStatusClosed):
		// FR-009: verifying→closed is a pass-through placeholder until F4.
		log.Printf("[bug-gate] verifying→closed: verifier not yet implemented — see F4")
		return GateResult{Stage: to, Satisfied: true, Reason: "verifier not yet implemented — see F4"}

	default:
		// All other bug transitions (pre-in-progress, backward, unknown) are ungated.
		return GateResult{Stage: to, Satisfied: true}
	}
}

// checkBugWorktreeHasCommits verifies that the bug's worktree branch has at
// least one commit beyond the base branch (FR-006).
func checkBugWorktreeHasCommits(bug *model.Bug, docSvc *DocumentService) GateResult {
	repoRoot := docSvc.RepoRoot()
	branchName := worktree.GenerateBranchName(bug.ID, bug.Slug)

	// GetCommitsBehindAhead returns an error if the branch doesn't exist, which
	// we treat as "no fix commits found on worktree".
	_, ahead, err := git.GetCommitsBehindAhead(repoRoot, branchName, "")
	if err != nil {
		return GateResult{
			Stage:     string(model.BugStatusNeedsReview),
			Satisfied: false,
			Reason:    "no fix commits found on worktree",
		}
	}

	if ahead == 0 {
		return GateResult{
			Stage:     string(model.BugStatusNeedsReview),
			Satisfied: false,
			Reason:    "no fix commits found on worktree",
		}
	}

	return GateResult{
		Stage:     string(model.BugStatusNeedsReview),
		Satisfied: true,
		Reason:    fmt.Sprintf("worktree has %d commit(s) ahead of base", ahead),
	}
}

// checkBugReviewReportAndTests verifies that a review report document exists
// for the bug and that go test passes on the worktree (FR-007).
func checkBugReviewReportAndTests(bug *model.Bug, docSvc *DocumentService) GateResult {
	// Check 1: Review report document exists and is owned by the bug.
	docs, err := docSvc.ListDocuments(DocumentFilters{
		Owner: bug.ID,
		Type:  string(model.DocumentTypeReport),
	})
	if err != nil || len(docs) == 0 {
		return GateResult{
			Stage:     string(model.BugStatusVerifying),
			Satisfied: false,
			Reason:    "no review report document registered for this bug",
		}
	}

	// Check 2: go test ./... passes on the worktree.
	repoRoot := docSvc.RepoRoot()
	wtPath := worktree.GenerateWorktreePath(bug.ID, bug.Slug)
	absWTPath := filepath.Join(repoRoot, wtPath)

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = absWTPath
	if out, testErr := cmd.CombinedOutput(); testErr != nil {
		return GateResult{
			Stage:     string(model.BugStatusVerifying),
			Satisfied: false,
			Reason:    fmt.Sprintf("go test failed: %v\n%s", testErr, string(out)),
		}
	}

	return GateResult{
		Stage:     string(model.BugStatusVerifying),
		Satisfied: true,
		Reason:    fmt.Sprintf("review report found (%s) and go test passes", docs[0].ID),
	}
}

// checkBugReviewCap increments the bug's review_cycle, checks the tier cap,
// and blocks the transition if the cap is reached (FR-008/FR-014).
func checkBugReviewCap(bug *model.Bug, entitySvc *EntityService) GateResult {
	// Increment review_cycle (persisted before gate evaluation).
	newCycle := bug.ReviewCycle + 1
	if err := entitySvc.IncrementBugReviewCycle(bug.ID, bug.Slug); err != nil {
		log.Printf("[bug-gate] WARNING: failed to increment review cycle for %s: %v", bug.ID, err)
		// Continue with in-memory value even if persist fails.
	}

	// Resolve tier's MaxCycles.
	tierCfg := ResolveBugTierConfig(bug.Tier)

	if newCycle >= tierCfg.MaxCycles {
		return GateResult{
			Stage:            string(model.BugStatusNeedsRework),
			Satisfied:        false,
			Reason:           fmt.Sprintf("Review iteration cap reached (%d/%d). Human decision required: accept with known issues, rework with revised scope, or cancel.", newCycle, tierCfg.MaxCycles),
			ReviewCapReached: true,
		}
	}

	return GateResult{
		Stage:     string(model.BugStatusNeedsRework),
		Satisfied: true,
		Reason:    fmt.Sprintf("review cycle %d/%d", newCycle, tierCfg.MaxCycles),
	}
}

// ResolveBugTierConfig returns the TierConfig for the bug's tier, defaulting
// to the bug_fix tier config if the bug has no tier or the tier is unknown.
func ResolveBugTierConfig(tier string) config.TierConfig {
	ft := config.DefaultFastTrackConfig()
	if tier == "" {
		tier = config.TierBugFix
	}
	if cfg, ok := ft.Tiers[tier]; ok {
		return cfg
	}
	return ft.Tiers[config.TierBugFix]
}

// checkDevelopingGate checks whether the feature has at least one child task.
func checkDevelopingGate(feature *model.Feature, entitySvc *EntityService) GateResult {
	stage := string(model.FeatureStatusDeveloping)

	tasks, err := entitySvc.List("task")
	if err != nil {
		return GateResult{
			Stage:     stage,
			Satisfied: false,
			Reason:    fmt.Sprintf("error listing tasks: %v", err),
		}
	}

	for _, t := range tasks {
		parentFeature := stringFromState(t.State, "parent_feature")
		if parentFeature == feature.ID {
			return GateResult{
				Stage:     stage,
				Satisfied: true,
				Reason:    fmt.Sprintf("feature has child task: %s", t.ID),
			}
		}
	}

	return GateResult{
		Stage:     stage,
		Satisfied: false,
		Reason:    "feature has no child tasks",
	}
}
