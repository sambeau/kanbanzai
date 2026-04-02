package gate

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

func init() {
	RegisterEvaluator("documents", evalDocuments)
}

// docTypeToField maps document prerequisite types to the corresponding
// field accessor on a Feature (the direct document reference).
var docTypeToField = map[string]func(*model.Feature) string{
	"design":        func(f *model.Feature) string { return f.Design },
	"specification": func(f *model.Feature) string { return f.Spec },
	"dev-plan":      func(f *model.Feature) string { return f.DevPlan },
}

// evalDocuments evaluates document prerequisites using a three-level lookup:
//  1. Feature's own document field reference (Design, Spec, DevPlan)
//  2. Documents owned by the feature
//  3. Documents owned by the parent plan
func evalDocuments(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult {
	var results []GateResult

	for _, dp := range prereqs.Documents {
		results = append(results, evalOneDocument(dp, stage, ctx))
	}

	return results
}

func evalOneDocument(dp binding.DocumentPrereq, stage string, ctx PrereqEvalContext) GateResult {
	// Level 1: check the feature's direct document field reference.
	if fieldFn, ok := docTypeToField[dp.Type]; ok {
		ref := fieldFn(ctx.Feature)
		if ref != "" {
			doc, err := ctx.DocSvc.GetDocument(ref, false)
			if err == nil && doc != nil && doc.Status == dp.Status {
				return GateResult{
					Stage:     stage,
					Satisfied: true,
					Reason:    fmt.Sprintf("%s document %s is %s", dp.Type, doc.ID, dp.Status),
					Source:    "registry",
				}
			}
		}
	}

	// Level 2: documents owned by the feature.
	docs, err := ctx.DocSvc.ListDocuments(DocumentFilters{
		Owner:  ctx.Feature.ID,
		Type:   dp.Type,
		Status: dp.Status,
	})
	if err == nil && len(docs) > 0 {
		return GateResult{
			Stage:     stage,
			Satisfied: true,
			Reason:    fmt.Sprintf("%s document %s owned by feature is %s", dp.Type, docs[0].ID, dp.Status),
			Source:    "registry",
		}
	}

	// Level 3: documents owned by the parent plan.
	if ctx.Feature.Parent != "" {
		docs, err = ctx.DocSvc.ListDocuments(DocumentFilters{
			Owner:  ctx.Feature.Parent,
			Type:   dp.Type,
			Status: dp.Status,
		})
		if err == nil && len(docs) > 0 {
			return GateResult{
				Stage:     stage,
				Satisfied: true,
				Reason:    fmt.Sprintf("%s document %s owned by parent plan is %s", dp.Type, docs[0].ID, dp.Status),
				Source:    "registry",
			}
		}
	}

	return GateResult{
		Stage:     stage,
		Satisfied: false,
		Reason:    fmt.Sprintf("no %s %s document found", dp.Status, dp.Type),
		Source:    "registry",
	}
}
