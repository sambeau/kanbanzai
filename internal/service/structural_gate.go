package service

import (
	"path/filepath"

	"github.com/sambeau/kanbanzai/internal/docint"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/structural"
)

// gateDocInfo describes which document to structurally check for a (from→to) gate.
type gateDocInfo struct {
	docRef  func(*model.Feature) string
	docType string
}

// gateDocChecks maps transitions to the document field to structurally check.
var gateDocChecks = map[string]gateDocInfo{
	"designing→specifying": {
		docRef:  func(f *model.Feature) string { return f.Design },
		docType: "design",
	},
	"specifying→dev-planning": {
		docRef:  func(f *model.Feature) string { return f.Spec },
		docType: "specification",
	},
	"dev-planning→developing": {
		docRef:  func(f *model.Feature) string { return f.DevPlan },
		docType: "dev-plan",
	},
}

// runStructuralChecksForGate runs structural checks on the document associated
// with the given (from→to) gate. Returns nil results if no checks apply or the
// document is not indexed. Returns anyHardFail=true if a hard_gate check fails.
func runStructuralChecksForGate(
	from, to string,
	feature *model.Feature,
	docSvc *DocumentService,
) (results []structural.CheckResult, anyHardFail bool) {
	gate := from + "→" + to
	info, ok := gateDocChecks[gate]
	if !ok {
		return nil, false
	}

	docID := info.docRef(feature)
	if docID == "" {
		return nil, false
	}

	// Load docint index — best effort; skip if not yet indexed.
	indexStore := docint.NewIndexStore(filepath.Join(docSvc.stateRoot, "index"))
	idx, err := indexStore.LoadDocumentIndex(docID)
	if err != nil {
		return nil, false
	}

	// Load promotion state — skip structural checks if unavailable.
	ps, err := structural.LoadPromotionState(docSvc.stateRoot)
	if err != nil {
		return nil, false
	}

	appendCheck := func(r structural.CheckResult) {
		key := structural.CheckKey{CheckType: r.CheckType, DocumentType: info.docType}
		r.Mode = ps.GetMode(key)
		results = append(results, r)
		if r.Passed {
			ps.RecordPass(key)
		}
		if !r.Passed && r.Mode == "hard_gate" {
			anyHardFail = true
		}
	}

	// Required sections (all document types).
	appendCheck(structural.CheckRequiredSections(idx.Sections, info.docType, docID, gate))

	// Spec-specific: acceptance criteria + cross-reference.
	if info.docType == "specification" {
		appendCheck(structural.CheckAcceptanceCriteria(idx.Sections, idx.ConventionalRoles, docID, gate))

		var designPaths, designIDs []string
		if feature.Design != "" {
			designIDs = append(designIDs, feature.Design)
			if doc, err := docSvc.GetDocument(feature.Design, false); err == nil {
				designPaths = append(designPaths, doc.Path)
			}
		}
		er := docint.ExtractResult{CrossDocLinks: idx.CrossDocLinks, EntityRefs: idx.EntityRefs}
		appendCheck(structural.CheckCrossReference(er, designPaths, designIDs, docID, gate))
	}

	_ = ps.Save() // best-effort persist
	return results, anyHardFail
}
