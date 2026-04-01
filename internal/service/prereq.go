package service

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/structural"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// GateResult describes whether a stage gate prerequisite is satisfied.
type GateResult struct {
	Stage            string // the lifecycle stage being checked
	Satisfied        bool
	Reason           string                   // human-readable explanation
	StructuralChecks []structural.CheckResult // populated when structural checks ran
}

// stageDocMapping maps feature lifecycle stages to their required document types.
var stageDocMapping = map[string]string{
	string(model.FeatureStatusDesigning):   string(model.DocumentTypeDesign),
	string(model.FeatureStatusSpecifying):  string(model.DocumentTypeSpecification),
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

	return checkDocumentGate(stage, docType, feature, docSvc)
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
func checkDocumentGate(stage, docType string, feature *model.Feature, docSvc *DocumentService) GateResult {
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

	// 3. Check documents owned by the parent plan.
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
				Reason:    fmt.Sprintf("approved %s document owned by parent plan %s: %s", docType, feature.Parent, parentDocs[0].ID),
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
		docResult := checkDocumentGate(string(model.FeatureStatusDesigning), string(model.DocumentTypeDesign), feature, docSvc)
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
		docResult := checkDocumentGate(string(model.FeatureStatusSpecifying), string(model.DocumentTypeSpecification), feature, docSvc)
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
		docResult := checkDocumentGate(string(model.FeatureStatusDevPlanning), string(model.DocumentTypeDevPlan), feature, docSvc)
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
