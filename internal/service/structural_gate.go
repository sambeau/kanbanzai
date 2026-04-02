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

	// At dev-planning→developing: also check the approved specification for
	// acceptance criteria and cross-reference (FR-001). B-17 fix.
	// Note: these spec-specific checks do NOT run at specifying→dev-planning. B-16 fix.
	if gate == "dev-planning→developing" && feature.Spec != "" {
		specIdx, err := indexStore.LoadDocumentIndex(feature.Spec)
		if err == nil {
			const specDocType = "specification"

			appendSpecCheck := func(r structural.CheckResult) {
				key := structural.CheckKey{CheckType: r.CheckType, DocumentType: specDocType}
				r.Mode = ps.GetMode(key)
				results = append(results, r)
				if r.Passed {
					ps.RecordPass(key)
				}
				if !r.Passed && r.Mode == "hard_gate" {
					anyHardFail = true
				}
			}

			appendSpecCheck(structural.CheckAcceptanceCriteria(specIdx.Sections, specIdx.ConventionalRoles, feature.Spec, gate))

			// Build design paths/IDs from feature.Design and from the parent
			// plan's design documents (FR-005). B-18 fix.
			var designPaths, designIDs []string
			if feature.Design != "" {
				designIDs = append(designIDs, feature.Design)
				if doc, err := docSvc.GetDocument(feature.Design, false); err == nil {
					designPaths = append(designPaths, doc.Path)
				}
			}
			if feature.Parent != "" {
				if planDocs, err := docSvc.ListDocuments(DocumentFilters{Owner: feature.Parent, Type: "design"}); err == nil {
					for _, d := range planDocs {
						if d.ID != feature.Design {
							designIDs = append(designIDs, d.ID)
							designPaths = append(designPaths, d.Path)
						}
					}
				}
			}

			er := docint.ExtractResult{CrossDocLinks: specIdx.CrossDocLinks, EntityRefs: specIdx.EntityRefs}
			appendSpecCheck(structural.CheckCrossReference(er, designPaths, designIDs, feature.Spec, gate))
		}
	}

	_ = ps.Save() // best-effort persist
	return results, anyHardFail
}
