# Audit-Codebase Examples

Worked examples of correct and incorrect codebase audit output.
Linked from `.kbz/skills/audit-codebase/SKILL.md`.

---

## BAD: Dead code reported without verification

> structural_quality: has_findings
>   Findings:
>     - [major] dead_code: formatEntity — zero callers (location: internal/format.go:42)
>       Recommendation: delete

**WHY BAD:** `formatEntity` was identified via a graph degree query but was
never verified with `trace_call_path` or a USAGE edge check. It may be called
via an interface, a test helper, or registered through reflection. Deleting it
without verification risks a regression.

---

## BAD: Linter output pasted without triage

> static_analysis: has_findings
>   Findings:
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/root.go:12)
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/root.go:12)
>     - [minor] go vet: unreachable code (location: internal/store/store.go:88)
>     - [major] staticcheck: SA4016: certain bitwise ops have no effect (location: internal/id.go:31)
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/util.go:7)
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/util.go:7)

**WHY BAD:** The same finding at `cmd/root.go:12` appears twice (not
deduplicated). All findings are classified major regardless of actual severity.
The reader cannot tell what to fix first.

---

## GOOD: Triaged audit with calibrated severity

> Quality Audit: pre-v1-release
> Scope: ./...
> Tools run: go vet, staticcheck, go test -race, knowledge graph
>
> Overall: has_major_findings
>
> Dimensions:
>
>   test_health: clean
>     Summary: 142 tests, 142 passed, 0 failed; race: none detected
>
>   static_analysis: clean_with_notes
>     Summary: 0 go vet findings, 2 staticcheck findings
>     Findings:
>       - [minor] staticcheck SA1006: printf with dynamic first arg
>           (location: internal/cli/root.go:14)
>           Recommendation: use fmt.Println or wrap in a format string
>       - [minor] staticcheck SA4016: bitwise AND always returns zero
>           (location: internal/id/id.go:31)
>           Recommendation: review flag mask; likely off-by-one in constant
>
>   structural_quality: has_findings
>     Summary: 1 dead code confirmed, 1 high fan-out, 0 coupled pairs
>     Findings:
>       - [major] dead_code: legacyMigrateV1 — zero callers confirmed,
>           no USAGE edges, not an entry point
>           (location: internal/store/migrate.go:88)
>           Recommendation: delete; migration path was superseded
>       - [minor] high_fan_out: applyTransition calls 11 functions
>           (location: internal/lifecycle/transition.go:55)
>           Recommendation: incidental — large switch dispatch; acceptable
>
>   style_conformance: clean_with_notes
>     Summary: 1 violation found
>     Findings:
>       - [minor] naming: exported type EntityId uses lowercase acronym
>           (location: internal/entity/types.go:12)
>           Recommendation: rename to EntityID per refs/go-style.md acronym rule
>
> Finding Summary:
>   Critical: 0
>   Major:    1
>   Minor:    3
>   Total:    4
>
> Next Actions:
>   1. Delete legacyMigrateV1 (internal/store/migrate.go:88) — confirmed dead code
>   2. Fix printf dynamic arg (internal/cli/root.go:14)
>   3. Rename EntityId -> EntityID (internal/entity/types.go:12)
>   4. Verify bitwise mask in id.go:31

**WHY GOOD:** Dead code was confirmed with `trace_call_path` before
classification. Findings are deduplicated and severity-ranked. The high-fan-out
function was examined and correctly classified as incidental rather than an SRP
violation. The Next Actions list gives the reader an unambiguous work queue.
