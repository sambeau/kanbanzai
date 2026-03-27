package service

import (
	"fmt"

	"kanbanzai/internal/model"
)

// GateResult describes whether a stage gate prerequisite is satisfied.
type GateResult struct {
	Stage     string // the lifecycle stage being checked
	Satisfied bool
	Reason    string // human-readable explanation
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
