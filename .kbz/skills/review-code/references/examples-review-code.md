# Review-Code Examples

Worked examples of correct and incorrect code review output.
Linked from `.kbz/skills/review-code/SKILL.md`.

---

## BAD: Rubber-stamp with prose

```
Review Unit: service-layer
Overall: approved
Notes: Code is well-structured and follows Go conventions. Good use
of error handling. Tests look comprehensive.
```

**WHY BAD:** No findings. No evidence citations. No per-dimension verdicts. Qualitative
prose ("well-structured," "comprehensive") with no structured data. A human or machine
cannot determine what was actually checked. This is FM-3.1 — indistinguishable from not
reviewing at all.

---

## GOOD: Evidence-backed structured review

```
Review Unit: service-layer
Files: internal/service/feature.go, internal/service/feature_test.go
Spec: work/spec/feature-lifecycle.md §3 (AC-1 through AC-5)
Reviewer Role: reviewer-conformance

Overall: approved_with_followups

Dimensions:
  spec_conformance: pass
    Evidence:
      - AC-1: entity creation verified (feature.go L34-52, NewFeature constructor)
      - AC-2: input validation verified (feature.go L55-71, Validate method)
      - AC-3: error response format verified (feature.go L73-89, error wrapping)
  implementation_quality: pass_with_notes
    Evidence:
      - Error handling present on all exported functions
      - Interface accepted at consumer (feature.go L8), struct returned
    Findings:
      - [non-blocking] Error wrapping in CreateFeature (feature.go L48) uses
        fmt.Errorf without %w — loses error chain for callers using errors.Is
        Recommendation: Use fmt.Errorf("create feature: %w", err) to preserve
        the error chain
  test_adequacy: pass
    Evidence:
      - 14 test cases covering happy path, validation failures, and duplicate
        detection (feature_test.go L12-189)
      - Table-driven pattern used throughout

Finding Summary:
  Blocking: 0
  Non-blocking: 1
  Total: 1
```

**WHY GOOD:** Per-dimension verdicts with specific evidence. The single finding has a
location, explanation, and remediation recommendation. Spec requirements cited by number.
A machine can parse this; a human can verify each claim.

---

## GOOD: Evidence-backed clearance with zero findings

```
Review Unit: storage-layer
Files: internal/store/yaml.go, internal/store/yaml_test.go
Spec: work/spec/entity-storage.md §2 (AC-4 through AC-6)
Reviewer Role: reviewer-conformance

Overall: approved

Dimensions:
  spec_conformance: pass
    Evidence:
      - AC-4: YAML serialisation verified (yaml.go L12-34, Marshal method)
      - AC-5: canonical field order verified (yaml.go L36-58, fieldOrder slice)
      - AC-6: round-trip determinism verified (yaml_test.go L102,
        TestStore_RoundTrip confirms identical output)
  implementation_quality: pass
    Evidence:
      - Error wrapping with %w throughout (yaml.go L22, L41, L67)
      - No exported functions without doc comments
      - Interface accepted at consumer (yaml.go L8), struct returned
  test_adequacy: pass
    Evidence:
      - 22 test cases including round-trip serialisation (yaml_test.go L15-198)
      - Error path coverage via TestStore_CreateConflict (yaml_test.go L145)
      - Table-driven pattern for serialisation variants

Finding Summary:
  Blocking: 0
  Non-blocking: 0
  Total: 0
```

**WHY GOOD:** Zero findings but substantive evidence for every dimension. The reviewer
demonstrably examined the code — each pass verdict is backed by specific locations and
spec anchors. This is a legitimate clearance, not a rubber stamp. Best example placed
last because it demonstrates the hardest case: saying "approved" credibly.
