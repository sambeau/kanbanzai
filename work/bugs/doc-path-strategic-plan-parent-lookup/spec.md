# Bug Specification: doc path fails with strategic-plan ancestor

## Observed Behaviour
Calling doc(action: path, parent: B69-skills-discoverability-quick-patches, type: specification, title: ...) returns: "cannot determine path: parent entity P64-binding-governance not found". B69 is a batch under P64 which is a strategic-plan. The doc tool walks up the parent chain to compute the canonical path and fails when it encounters the strategic-plan link, the same way CreateFeature did before BUG-01KR1E9XS0GJQ was fixed. Calling doc(action: path) directly with parent=P64-binding-governance returns the same error. Workaround: write the file at a guessed canonical path and call doc(action: register) with the explicit path; the register action then enforces the filename pattern (P64-{type}-{slug}.md) which gives a usable error message, but only after the file has been written at the wrong location.</observed>
<parameter name="priority">high

## Expected Behaviour
doc(action: path) and doc(action: register) walk the parent chain successfully when any ancestor is a strategic-plan. Canonical paths are returned/computed correctly. Recommended fix: replace the GetPlan-only lookup in the doc tool's parent resolution with the same fallback logic used for CreateFeature in the BUG-01KR1E9XS0GJQ fix (try GetPlan, fall through to GetStrategicPlan).

## Severity
high | Priority: medium | Type: implementation-defect
