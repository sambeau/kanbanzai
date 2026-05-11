| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-10T13:23:31Z          |
| Status | approved |
| Author | Architect (via write-research) |

# Research Report: Skills/Roles Pipeline Routing Integrity

## Research Question

The Kanbanzai skills/roles system — which maps workflow stages to agent identities
and procedures via `stage-bindings.yaml` — has accumulated failures that suggest
systemic fragility. Two concrete failures surfaced simultaneously:

1. `retro_fix` features (created via `retro(action: "create_fix")`) silently fall
   through to the generic `developing` pipeline — the `implement-retro-fix` skill
   on disk is **completely unreachable**.
2. `bug_fix` bugs have **no pipeline at all** — zero stage bindings exist for any
   bug lifecycle status.

Both failures reproduce identically in consumer projects because the relevant
files are embedded in the binary and installed verbatim by `kbz init`.

**The core question:** Is this system salvageable through hardening, or have we
reached the practical complexity ceiling for instruction-based orchestration —
where the gap between "instructions are advice" and "the system must behave
deterministically" becomes unbridgeable?

## Scope and Methodology

### In scope

1. Root-cause analysis of the `retro_fix` and `bug_fix` routing failures —
   tracing every layer from entity creation through context assembly to skill
   loading.
2. Git archaeology on `stage-bindings.yaml`, `internal/context/pipeline.go`,
   `internal/binding/model.go`, and `internal/binding/validate.go`.
3. Brittleness diagnosis: enumeration of every synchronization dependency and
   failure mode.
4. Completeness audit: reachability matrix for every binding × skill × pipeline
   path.
5. Consumer-project impact analysis via `internal/kbzinit/`.
6. Industry comparison across AI-coding orchestration systems and agent framework
   best practices.
7. Concrete regression test designs.
8. Long-term architectural recommendation with confidence levels.

### Out of scope

- Fixing the actual bugs (this is investigation, not implementation).
- Redesigning skill content itself.
- Evaluating whether the orchestration model is correct per stage.
- Broader Kanbanzai architecture beyond the skills/roles pipeline.

### Methodology

1. **Code archaeology**: Traced `internal/context/pipeline.go` (all 15 pipeline
   steps), `internal/binding/model.go` (StageBinding, validStages,
   ValidateBinding), `internal/binding/validate.go` (ValidateBindingFile),
   `internal/binding/registry.go` (BindingRegistry), `internal/binding/loader.go`
   (LoadBindingFile), `internal/mcp/handoff_tool.go` (PipelineInput construction),
   `internal/mcp/server.go` (pipeline initialization), `internal/model/entities.go`
   (Feature struct with Tier), `kbzschema/types.go` (public Feature/Bug schemas),
   `internal/kbzinit/stage_bindings.go` (embedded bindings installation).
2. **Git archaeology**: Traced commit history for `stage-bindings.yaml`,
   `model.go`, `pipeline.go`, and `validate.go`.
3. **Reachability analysis**: Audited every binding in `stage-bindings.yaml` and
   every skill in `.kbz/skills/` against pipeline routing.
4. **Retrospective synthesis**: Called `retro(action: "synthesise")` to surface
   signals from all prior sessions.
5. **Synthesis**: Findings organized by theme with evidence grades.

---

## Findings

### Finding 1: The pipeline resolves `status → binding` with zero tier awareness (root cause)

The pipeline's `stepResolveStage` extracts the feature's `status` field and
uses it directly as the lookup key into `stage-bindings.yaml`:

```go
// internal/context/pipeline.go lines 291-300
func (p *Pipeline) stepResolveStage(state *PipelineState) error {
    status, _ := state.Input.FeatureState["status"].(string)
    state.Stage = status  // ← "developing", "reviewing", etc.
    return nil
}

// internal/context/pipeline.go lines 303-318
func (p *Pipeline) stepLookupBinding(state *PipelineState) error {
    b, err := p.Bindings.Lookup(state.Stage)  // ← Lookup("developing")
    state.Binding = b
    return nil
}
```

A feature with `tier: retro_fix` and `status: developing` resolves to the
standard `developing` binding because `Bindings.Lookup("developing")` returns
the standard developing binding. The `retro-fixing` binding key in
`stage-bindings.yaml` is never consulted because no feature ever has
`retro-fixing` as a lifecycle status.

The `Feature` struct in `internal/model/entities.go` (line 400) does carry a
`Tier` field:

```go
// internal/model/entities.go line 400
Tier string `yaml:"tier,omitempty"`
```

And the tier is stored in the feature's state map accessible via
`PipelineInput.FeatureState`. But zero pipeline steps read it. The
`StageBinding` struct has the `Tier` field decoded from YAML, but zero
pipeline steps consume it.

**Source:** `internal/context/pipeline.go` (primary, current)
**Confidence:** High

### Finding 2: The `StageBinding` struct has tier-awareness fields that are decoded but never consumed

```go
// internal/binding/model.go lines 30-33
// Profile, Tier, Modes, and Verifying support stages that opt into the
// gated-mode profile schema (e.g. retro-fixing). They are decoded but not
// yet consumed by the pipeline; full schema work is tracked separately.
Profile   *bool                 `yaml:"profile,omitempty"`
Tier      string                `yaml:"tier,omitempty"`
Modes     map[string]*StageMode `yaml:"modes,omitempty"`
Verifying *VerifyingBlock       `yaml:"verifying,omitempty"`
```

The comment itself acknowledges these are "not yet consumed." No pipeline step
reads `Tier`, `Profile`, `Modes`, or `Verifying`. These were added as a forward
declaration but the routing layer was never completed.

**Source:** `internal/binding/model.go` lines 30-33 (primary, current)
**Confidence:** High

### Finding 3: `validStages` does not include `retro-fixing` or any bug-lifecycle stages

The hardcoded `validStages` map in `internal/binding/model.go`:

```go
// internal/binding/model.go lines 116-124
var validStages = map[string]bool{
    "designing":      true,
    "specifying":     true,
    "dev-planning":   true,
    "developing":     true,
    "reviewing":      true,
    "researching":    true,
    "documenting":    true,
    "plan-reviewing": true,
}
```

`retro-fixing` is absent. `merging` and `verifying` — both present in the live
`.kbz/stage-bindings.yaml` — are also absent. No bug-lifecycle stages
(`in-progress`, `needs-review`, `needs-rework`, `verifying`, etc.) appear.

This has a compounding consequence: even if `ValidateBindingFile` were called
against the production file, it would reject `retro-fixing`, `merging`, and
`verifying` as invalid stage names.

**Source:** `internal/binding/model.go` lines 116-124 (primary, current)
**Confidence:** High

### Finding 4: The `retro-fixing` binding is structurally invalid — but validation is never run

The `retro-fixing` binding in `.kbz/stage-bindings.yaml` declares `profile`,
`tier`, `modes`, and `verifying` blocks — but has no `orchestration`, `roles`,
or `skills` fields at the top level. `ValidateBinding` (called by
`ValidateBindingFile`) would reject it with three errors:

1. `invalid orchestration ""` (empty string not in `validOrchestrations`)
2. `roles must not be empty`
3. `skills must not be empty`

Additionally, `ValidateBindingFile` would reject `retro-fixing` as an invalid
stage name (not in `validStages`).

**However, validation is never run against the production file.** The server
initialization path (`internal/mcp/server.go` lines 140-166) calls
`binding.LoadBindingFile` — which only parses and decodes the YAML — then
passes the raw `BindingFile` directly to `BindingFileAdapter`. This adapter is
a simple map wrapper with no validation:

```go
// internal/context/pipeline_adapters.go lines 50-59
func (a *BindingFileAdapter) Lookup(stage string) (*binding.StageBinding, error) {
    sb, ok := a.File.StageBindings[stage]
    if !ok {
        return nil, fmt.Errorf("no binding for stage %q", stage)
    }
    return sb, nil
}
```

The `BindingRegistry.Load()` path — which does call `ValidateBindingFile` — is
an alternative code path used elsewhere (possibly CLI tools), **not** by the
MCP server pipeline initialization.

**Source:** `internal/mcp/server.go` lines 140-166, `internal/context/pipeline_adapters.go` lines 50-59, `internal/binding/model.go` lines 116-180 (primary, current)
**Confidence:** High

### Finding 5: `implement-retro-fix` skill is complete but unreachable

The skill at `.kbz/skills/implement-retro-fix/SKILL.md` is a complete,
well-structured 300+ line document with:

- YAML frontmatter declaring `stage: developing` and roles
- Vocabulary section defining retro fix terms
- Anti-patterns specific to retro fix scenarios
- A 10-step procedure with checklist items

No binding lists `implement-retro-fix` in its `skills` field. No pipeline path
can reach it because:

1. The `retro-fixing` binding has no `skills` field (it's structurally invalid)
2. Even if it had skills, the pipeline never consults the `retro-fixing` binding
   because the feature's status is `developing`, not `retro-fixing`
3. The standard `developing` binding's skills are `[orchestrate-development]`

**Source:** `.kbz/skills/implement-retro-fix/SKILL.md`, `.kbz/stage-bindings.yaml` (primary, current)
**Confidence:** High

### Finding 6: Bugs have zero pipeline routing

Bug lifecycle statuses (`in-progress`, `needs-review`, `needs-rework`,
`verifying`, etc.) have no corresponding entries in `stage-bindings.yaml`.
There are no bug-lifecycle entries in `validStages`. The `handoff_tool.go`
handler only resolves parent features (not parent bugs) — there is no code
path for generating context for a bug worktree task.

The `WorktreeTransitionHook` (`internal/service/status_transition_hook.go`
line 132) does create worktrees for bugs transitioning to `in-progress`, but
no skills/roles pipeline exists to guide the agent once inside the worktree.

**Source:** `kbzschema/types.go` lines 55-65 (bug statuses), `stage-bindings.yaml` (no bug bindings), `internal/binding/model.go` lines 116-124 (no bug stages in validStages), `internal/mcp/handoff_tool.go` (no bug parent resolution) (primary, current)
**Confidence:** High

### Finding 7: Regression timeline — two separate creation events with an unreconciled gap

**Event 1: Foundation (2026-04-01)**
- Commit `53bef6a0`: Foundation model types and validation for roles, skills,
  and bindings created. `validStages` defined with 8 standard lifecycle stages.
- Commit `3c691d7a`: 10-step pipeline orchestrator created. The pipeline uses
  `status → binding` lookup with no tier awareness.

**Event 2: Retro-fix binding added (2026-05-07)**
- Commit `b4b2de39`: `retro-fixing` profile added to `stage-bindings.yaml`
  with `profile: true`, `tier: retro_fix`, `modes`, and `verifying` blocks.
  The `Tier` field was already present in the `StageBinding` struct (added
  at foundation time), but the routing layer was never modified to use it.

**Gap:** Between foundation (April 1) and the retro-fix binding addition
(May 7), no work was done to implement tier-aware routing. The binding was
added knowing the routing didn't exist — the comment "full schema work is
tracked separately" in `model.go` line 30 was treated as an accepted
deferral, not a blocking prerequisite.

The `FixPlan` field was also added to the `Bug` entity in the same commit
(`b4b2de39`), suggesting a parallel intent to add bug pipeline support.
But no further commits addressed either gap.

**Source:** Git log for `b4b2de39`, `3c691d7a`, `53bef6a0` (primary, 2026-04-01 through 2026-05-07)
**Confidence:** High

### Finding 8: The embedded bindings file is missing three bindings present in the live file

The file embedded at `internal/kbzinit/stage-bindings.yaml` (installed to
consumer projects by `kbz init`) contains **9 bindings**. The live file at
`.kbz/stage-bindings.yaml` contains **12 bindings**. The three missing
bindings are: `merging`, `verifying`, and `retro-fixing`.

This means:

1. Consumer projects do not have `retro-fixing` at all — even the unreachable
   dead binding is stripped.
2. Consumer projects are also missing `merging` and `verifying` —
   though these share skills with the `reviewing` orchestration and may work
   via the `reviewing` pipeline.

There is no automated synchronization mechanism between the live
`stage-bindings.yaml` and the embedded copy. The embedded file must be
manually updated — and was not updated when `b4b2de39` added `retro-fixing`
to the live file.

**Source:** `internal/kbzinit/stage-bindings.yaml` vs `.kbz/stage-bindings.yaml`, diff analysis (primary, current)
**Confidence:** High

### Finding 9: Synchronization dependency enumeration — six independent components

Adding a new lifecycle stage or tier requires updates to **six** independent
components, none of which are mechanically linked:

| # | Component | Location | Failure if missed |
|---|-----------|----------|-------------------|
| 1 | Binding entry in YAML | `stage-bindings.yaml` | `stepLookupBinding` returns error (loud) |
| 2 | `validStages` Go map | `internal/binding/model.go` | `ValidateBindingFile` rejects stage (loud if called; silent in server path) |
| 3 | `workableStatuses` Go slice | `internal/context/pipeline.go` | `stepValidateLifecycle` rejects feature status (loud) |
| 4 | Feature lifecycle constants | `kbzschema/types.go` + `internal/model/entities.go` | Entity creation/transition may fail (loud or silent depending on path) |
| 5 | Embedded bindings file | `internal/kbzinit/stage-bindings.yaml` | Consumer projects get stale bindings (silent) |
| 6 | Tier-aware routing (not yet built) | `internal/context/pipeline.go` | Tier-based lookup never fires (silent) |

There is no checklist, no mechanical validation, and no automated test that
verifies these components stay synchronized. The knowledge of what must be
updated is tribal.

**Source:** Cross-referencing the six locations above (primary, current)
**Confidence:** High

### Finding 10: The system fails silently more often than loudly

Of the four failure modes identified in the current bugs:

| Failure | Loud or Silent? | Detection |
|---------|-----------------|-----------|
| `retro_fix` feature routes to wrong pipeline | **Silent** | Agent follows wrong skill; user may notice incorrect behavior |
| `bug_fix` has no pipeline | **Silent** (no handoff possible unless a task exists under a feature) | Only detectable if someone tries to hand off a bug task |
| Missing embedded bindings | **Silent** | Consumer project silently lacks retro-fixing, merging, verifying |
| `ValidateBindingFile` not called in server path | **Silent** | Invalid bindings loaded without error |

The pipeline *does* fail loudly when a feature status has no binding at all
(`stepLookupBinding` returns error). But tier-based misrouting is silent
because the feature's `status` always matches a standard binding.

**Source:** Tracing all four failure paths through the code (primary, current)
**Confidence:** High

---

## Reachability Matrix

### Bindings → Pipeline Reachability

| Binding | In `validStages`? | In `workableStatuses`? | In embedded? | Pipeline can route? | Skills exist on disk? | Notes |
|---------|-------------------|----------------------|-------------|--------------------|-----------------------|-------|
| `designing` | Yes | Yes | Yes | Yes | `write-design` ✓ | |
| `specifying` | Yes | Yes | Yes | Yes | `write-spec` ✓ | |
| `dev-planning` | Yes | Yes | Yes | Yes | `write-dev-plan`, `decompose-feature` ✓ | |
| `developing` | Yes | Yes | Yes | Yes | `orchestrate-development` ✓ | |
| `reviewing` | Yes | Yes | Yes | Yes | `orchestrate-review` ✓ | |
| `merging` | No | No | No | **No** | `orchestrate-review` (shared) ✓ | `validStages` rejects; not in embedded |
| `verifying` | No | No | No | **No** | `orchestrate-review` (shared) ✓ | `validStages` rejects; not in embedded |
| `batch-reviewing` | As `plan-reviewing` | As `plan-reviewing` | Yes | Yes (via `plan-reviewing`) | `review-plan` ✓ | Key is `plan-reviewing` in YAML, `batch-reviewing` in embedded |
| `researching` | Yes | Yes | Yes | Yes | `write-research` ✓ | |
| `documenting` | Yes | Yes | Yes | Yes | `update-docs` ✓ | |
| `doc-publishing` | No | No | Yes | **No (via pipeline)** | `orchestrate-doc-pipeline` ✓ | Not in `validStages` or `workableStatuses`; may work via direct trigger |
| `retro-fixing` | No | No | No | **No** | N/A (binding has no skills) | Structurally invalid binding; tier-based routing never built |

### Skills on Disk → Reachability

| Skill directory | Referenced by which binding? | Reachable? |
|----------------|------------------------------|------------|
| `write-design` | `designing` | Yes |
| `write-spec` | `specifying` | Yes |
| `write-dev-plan` | `dev-planning` | Yes |
| `decompose-feature` | `dev-planning` | Yes |
| `orchestrate-development` | `developing` | Yes |
| `orchestrate-review` | `reviewing`, `merging`, `verifying` | Via `reviewing` only |
| `review-code` | `reviewing` (sub_agents) | Yes (sub-agent) |
| `implement-task` | `developing` (sub_agents) | Yes (sub-agent) |
| `review-plan` | `batch-reviewing` | Yes (via `plan-reviewing`) |
| `write-research` | `researching` | Yes |
| `update-docs` | `documenting` | Yes |
| `orchestrate-doc-pipeline` | `doc-publishing` | **No (via pipeline)** |
| `write-docs` | `doc-publishing` (sub_agents) | Via sub_agents only |
| `edit-docs` | `doc-publishing` (sub_agents) | Via sub_agents only |
| `check-docs` | `doc-publishing` (sub_agents) | Via sub_agents only |
| `style-docs` | `doc-publishing` (sub_agents) | Via sub_agents only |
| `copyedit-docs` | `doc-publishing` (sub_agents) | Via sub_agents only |
| `verify-closeout` | `verifying` (sub_agents), `retro-fixing` (verifying block) | **No** (neither `verifying` nor `retro-fixing` reachable) |
| `implement-retro-fix` | None | **No** |
| `validate-spec` | None | **No** (standalone, triggered by name) |
| `validate-plan` | None | **No** (standalone, triggered by name) |
| `validate-review` | None | **No** (standalone, triggered by name) |
| `audit-codebase` | None | **No** (standalone, triggered by name) |
| `write-skill` | None | **No** (standalone, triggered by name) |
| `prompt-engineering` | None | **No** (standalone, triggered by name) |

**Summary:** Of 26 skill directories, 14 are reachable via the standard
pipeline. 12 are either standalone (triggered by name, not by binding),
unreachable dead code (`implement-retro-fix`), or reachable only via
unreachable bindings (`verify-closeout`).

---

## Trade-Off Analysis

| Dimension | Option A: Harden Instructions | Option B: Code-Managed Pipelines | Option C: Hybrid |
|-----------|------------------------------|----------------------------------|------------------|
| **Correctness guarantee** | Low — instructions remain advisory; agents may skip or misinterpret | High — Go code routing is deterministic | Medium-high — code routing for execution, YAML for declaration |
| **Maintainability** | Low — 6+ synchronization points, manual coordination | High — single source of truth in code; compiler catches drift | Medium — YAML declares; code validates; tests enforce consistency |
| **Configurability** | High — YAML bindings can be edited without recompilation | Low — pipeline changes require Go code changes and rebuild | Medium — YAML configurable for standard stages; code for tier-aware routing |
| **Migration cost** | Low — fix existing bugs, add tests, add validation | High — rewrite pipeline routing, migrate bindings to code | Medium — add tier-aware routing to existing pipeline; keep YAML |
| **Consumer impact** | Medium — consumer bindings must be resynced; validation may reject customizations | High — consumers lose ability to customize bindings; migration required | Low-medium — consumers keep YAML customization; validation improves |
| **Complexity ceiling** | Medium — each new tier/stage adds O(n) manual sync points (currently 6) | High — Go code scales with constant overhead per new pipeline | Medium — new stages add 1 YAML entry + 1 code path |
| **Testability** | Low — instruction-based systems are hard to test mechanically | High — Go code paths are unit-testable; binding validation is testable | High — code paths testable; YAML validation testable |
| **Failure mode** | Mostly silent (current situation) | Mostly loud (compiler errors, test failures) | Mostly loud (validation at startup) |

---

## Recommendations

### Recommendation 1: Adopt the hybrid approach (Option C)

**What:** Keep `stage-bindings.yaml` for human-authored stages (designing,
specifying, developing, reviewing, researching, documenting), but implement
tier-aware routing in Go code. The pipeline should resolve `(status, tier) →
binding` rather than `status → binding`. The YAML declares *what*; the code
enforces *how to route there*.

**Confidence:** Medium
**Based on:** Findings 1, 2, 3, 9, 10.
**Conditions:** This recommendation assumes the project will continue adding
tiers (e.g., `critical_fix`) beyond `retro_fix` and `bug_fix`. If `retro_fix`
and `bug_fix` are the only anticipated tiers, a simpler approach may suffice.

### Recommendation 2: Run `ValidateBindingFile` at server startup (immediate fix)

**What:** Replace the current `LoadBindingFile` → `BindingFileAdapter` path
in `server.go` with `BindingRegistry.Load()` → adapter, which calls
`ValidateBindingFile` and rejects invalid bindings at startup. This converts
several silent failures into loud ones.

**Confidence:** High
**Based on:** Findings 4, 10. The validation code already exists; it's simply
not wired into the server path. This is a low-risk, high-impact change.
**Implementation:** ~5 lines changed in `internal/mcp/server.go`.

### Recommendation 3: Synchronize `validStages` and `workableStatuses` with stage-bindings.yaml

**What:** Either (a) derive `validStages` and `workableStatuses` from the
YAML file at startup rather than hardcoding them, or (b) add a Go test that
loads the YAML and asserts every stage name in the YAML appears in both Go
constants, and vice versa.

**Confidence:** High
**Based on:** Finding 9 (six-component synchronization). This is a
well-understood problem — derive from source of truth or test for drift.
**Implementation:** ~30 lines in a new test file.

### Recommendation 4: Add a pipeline routing test for every tier

**What:** A Go test that creates a mock `PipelineInput` with
`FeatureState["status"] = "developing"` and `FeatureState["tier"] = "retro_fix"`,
runs the pipeline, and asserts `state.Binding` is the `retro-fixing` binding
(or the correct merged behavior). Do the same for `bug_fix`, `feature`
(default), and `critical`.

**Confidence:** High
**Based on:** Findings 1, 2, 5. These tests would have caught the current
failure at implementation time.
**Implementation:** ~60 lines per tier, in `pipeline_test.go`.

### Recommendation 5: Test embedded bindings parity

**What:** A Go test that loads both `.kbz/stage-bindings.yaml` (source of
truth) and `internal/kbzinit/stage-bindings.yaml` (embedded), and asserts
they are byte-identical (after stripping managed-marker comments).

**Confidence:** High
**Based on:** Finding 8. The embedded file is 3 bindings behind the live
file. A test would catch this at build time.
**Implementation:** ~25 lines in `internal/kbzinit/`.

### Recommendation 6: Add a `kbz health` check for binding/skill integrity

**What:** A health check that validates: every binding's roles exist on disk;
every binding's skills exist on disk; every feature lifecycle status has a
binding; embedded bindings match live bindings. This surfaces binding/skill
mismatches in consumer projects at runtime rather than at implementation time.

**Confidence:** Medium
**Based on:** Finding 8 (consumer impact), Finding 10 (silent failures).
**Implementation:** ~80 lines in `internal/health/`.

### Recommendation 7: Implement tier-aware routing for `retro_fix` and `bug_fix`

**What:** Add a pipeline step between `stepResolveStage` and
`stepLookupBinding` that checks `FeatureState["tier"]` and, if set to
`retro_fix` or `bug_fix`, overrides the stage lookup key to use the
tier-specific binding. For `retro_fix`, the binding would need to be made
structurally valid (add `orchestration`, `roles`, `skills` fields). For
`bug_fix`, create a new binding mapping bug lifecycle statuses to
appropriate roles/skills.

**Confidence:** Medium
**Based on:** Findings 1, 2, 5, 6. This is the actual fix for the reported
bugs. The medium confidence reflects uncertainty about whether `bug_fix`
should have its own pipeline or reuse the feature pipeline with different
skills.
**Implementation:** ~100 lines in `pipeline.go` + binding file updates.

---

## Limitations

1. **Bug pipeline design**: This report identifies that bugs have no pipeline
   but does not investigate what the correct bug pipeline design would be.
   Bug lifecycle stages (`in-progress`, `needs-review`, `needs-rework`,
   `verifying`) are different from feature stages (`developing`, `reviewing`),
   and mapping between them may require new bindings or a separate pipeline
   path.

2. **`doc-publishing` stage**: The `doc-publishing` binding uses
   `orchestration: pipeline-coordinator`, which is not in `validOrchestrations`.
   This binding may work via a separate code path (direct trigger by skill
   name) rather than through the standard pipeline. This report did not
   investigate the doc-publishing dispatch path.

3. **Sub-agent skill loading**: This report focuses on the top-level
   `stage → binding → skill` routing. The sub-agent skill routing in
   `stepLoadSkill` (which determines which skill a worker agent receives)
   was not exhaustively audited.

4. **`merging` and `verifying` stages**: These bindings exist in the live YAML
   but are missing from `validStages` and the embedded file. They share the
   `orchestrate-review` skill and may work correctly through the `reviewing`
   pipeline. This report did not verify their actual behavior.

5. **Consumer customization**: This report identifies that embedded bindings
   can diverge from the live file but does not investigate whether consumer
   projects *can* customize their installed `stage-bindings.yaml` and whether
   `kbz init` or `kbz upgrade` handles customizations correctly.

6. **Anthropic-specific research**: The investigation scope included surveying
   published research on instruction-following reliability. This was not
   completed in this draft. The trade-off analysis relies on internal
   structural analysis rather than external research validation.

7. **Complexity ceiling modeling**: The claim that the current architecture
   has a complexity ceiling of O(n) manual sync points is based on static
   analysis of the code structure, not on empirical measurement of failure
   rates as n increases.

8. **Assumptions**: This report assumes that tier-aware routing is the correct
   architectural direction (i.e., that `retro_fix` and `bug_fix` should be
   tiers that modify pipeline behavior, not separate lifecycle statuses). If
   the project decides instead to add `retro-fixing` as a separate lifecycle
   status, the routing problem changes significantly.
