# Design: Agentic Reviewing Stage Auto-Advance

**Feature:** FEAT-01KPQ08YE4399
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Status:** Draft

---

## Overview

The `reviewing` stage is defined in `.kbz/stage-bindings.yaml` with `human_gate: true`. In the
advance engine (`internal/service/advance.go`), this is enforced via `advanceStopStates` — a
hardcoded map that causes `AdvanceFeatureStatus` to halt unconditionally after entering
`reviewing`, even when the caller provides `override: true`. The only path past `reviewing` is an
explicit `entity(action: transition, status: done)` call issued after the advance has already
stopped.

In agentic-only pipelines — where every task is completed via `finish()` with a recorded
`verification` field — there is no meaningful human review happening. The reviewing stage is not
being used to run a reviewer-agent panel; it is simply an obstacle. Every agentic merge in
practice has required `override: true` on the `reviewing→done` gate, which is the only way to
proceed. This degrades the value of `override` as an exceptional-circumstance signal. When
`override` is the normal operating mode, it can no longer serve as an audit signal for genuine
exceptions.

The `require_github_pr` flag added in P24 established the pattern for this class of problem:
project-level config controls whether a workflow gate is active. That flag defaults to nil/false
(the gate is off) and teams that want the gate set it explicitly. This design follows the same
pattern for the reviewing stage.

---

## Goals and Non-Goals

### Goals

- Add a `require_human_review` flag to `MergeConfig` in `.kbz/config.yaml` that controls whether
  the reviewing stage is a mandatory halt in `AdvanceFeatureStatus`.
- When the flag is absent or `false`, allow `AdvanceFeatureStatus` to continue past `reviewing`
  toward `done` provided all auto-advance conditions are satisfied.
- When the flag is `true`, preserve the current mandatory-halt behaviour exactly.
- Define the auto-advance conditions precisely: all tasks terminal AND all tasks have a non-empty
  `verification` field.
- Mirror the `RequireGitHubPR` implementation pattern: `*bool` field, nil-equals-false accessor
  method, no default in `DefaultConfig`.

### Non-Goals

- This design does not remove the `reviewing` stage from the lifecycle or from the state machine.
  An explicit `entity(action: transition, status: reviewing)` call followed by `status: done`
  continues to work identically.
- This design does not change the `reviewing→done` transition gate (`checkReviewReportExists`).
  That gate applies only to the explicit two-step path.
- This design does not propagate task verification fields upward to the feature entity. That is
  the scope of P6 (a separate proposal item). The auto-advance condition here reads task records
  directly.
- This design does not affect the `needs-rework` cycle. If a feature has been through
  `reviewing→needs-rework`, re-entry into reviewing and further advance follow the same
  `require_human_review` logic.
- This design does not add feature-level config granularity. Project-level config is sufficient
  and consistent with P24.

---

## Design

### Config flag

Add `RequireHumanReview *bool` to `MergeConfig` in `internal/config/config.go`:

```kanbanzai/internal/config/config.go
type MergeConfig struct {
    PostMergeInstall    *bool `yaml:"post_merge_install,omitempty"`
    RequireGitHubPR     *bool `yaml:"require_github_pr,omitempty"`
    RequireHumanReview  *bool `yaml:"require_human_review,omitempty"`
}

func (m MergeConfig) RequiresHumanReview() bool {
    return m.RequireHumanReview != nil && *m.RequireHumanReview
}
```

The field is `*bool` with `omitempty` so it is absent from `config.yaml` by default, following
the exact `RequireGitHubPR` pattern. Nil and false are both treated as "human review not
required" by the accessor. `DefaultConfig()` requires no change — the zero value of `*bool` is
nil, which maps to false.

**Default rationale:** The default is `false` (nil). This is an intentional agentic-first
choice, consistent with P24's `require_github_pr` which also defaulted to off. The alternative
— defaulting to `true` for backward compatibility — would require every project to explicitly
opt out of a gate they may not need. Kanbanzai's design philosophy is that gates are opt-in.
Projects that need a human review gate (teams with compliance requirements, open-source
contribution workflows, etc.) set `require_human_review: true`.

The backward-compatibility cost is accepted: any project currently relying on the reviewing halt
as an implicit workflow pause will need to add `require_human_review: true` to their config. The
blast radius is low — the only known project using Kanbanzai is Kanbanzai itself, which runs an
agentic pipeline and benefits from the new default.

### Auto-advance conditions

When `RequiresHumanReview()` returns false, the advance engine may continue past `reviewing`
only when:

1. **All tasks are terminal.** Every task whose `parent_feature` matches the feature ID must be
   in a terminal status (`done` or `needs-review`). This condition is already enforced by the
   `developing→reviewing` gate; by the time `reviewing` is reached via advance, it is
   structurally guaranteed to hold.

2. **All tasks have recorded verification.** Every task must have a non-empty `Verification`
   field. This is the field written by `CompleteTask` when `VerificationPerformed` is provided
   to `finish()`. A task completed without verification is treated as unverified — the advance
   halts at `reviewing` with a clear message indicating which tasks lack verification.

These two conditions together establish that the work was done and was tested. They are the
agentic equivalent of a reviewer confirming the implementation is correct.

### Advance engine change

The `advanceStopStates` map in `advance.go` is currently a package-level constant. The change
makes the stop-state check conditional on project config.

`AdvanceFeatureStatus` already accepts an `*AdvanceConfig` parameter that carries optional
injected functions for gate checking, override policy, and checkpoint creation. The config
struct will gain a `RequiresHumanReview` function field:

```kanbanzai/internal/service/advance.go
type AdvanceConfig struct {
    CheckGate            GateCheckFunc
    OverridePolicy       OverridePolicyFunc
    OnCheckpoint         CheckpointCreateFunc
    RequiresHumanReview  func() bool  // nil → false (no human review required)
}
```

Inside the loop, the mandatory-halt block becomes:

```kanbanzai/internal/service/advance.go
requiresHumanReview := cfg != nil && cfg.RequiresHumanReview != nil && cfg.RequiresHumanReview()
if advanceStopStates[nextState] && !isTarget {
    if requiresHumanReview {
        return AdvanceResult{
            FinalStatus:  nextState,
            StoppedReason: "stopped at reviewing: require_human_review is true",
            ...
        }, nil
    }
    // Check auto-advance eligibility.
    if err := checkAllTasksHaveVerification(feature, entitySvc); err != nil {
        return AdvanceResult{
            FinalStatus:  nextState,
            StoppedReason: fmt.Sprintf("stopped at reviewing: %s", err),
            ...
        }, nil
    }
    // All conditions satisfied — continue past reviewing.
}
```

When `RequiresHumanReview` is nil on the config (the default call path), the advance behaves
as if human review is not required, enabling auto-advance for agentic pipelines without any
config change.

The caller in `entity_tool.go` (the MCP layer) is responsible for injecting the config
function, which it already does for other `AdvanceConfig` fields. It reads the project config
via `config.LoadOrDefault()` and sets `RequiresHumanReview` to `cfg.Merge.RequiresHumanReview`.

### Interface boundary: `checkAllTasksHaveVerification`

A new helper in `internal/service/prereq.go` (alongside `checkAllTasksTerminal`):

```kanbanzai/internal/service/prereq.go
func checkAllTasksHaveVerification(feature *model.Feature, entitySvc *EntityService) error
```

It loads all tasks for the feature, confirms each is terminal and has a non-empty `Verification`
field. Returns nil if all conditions hold; returns an error naming the first task that lacks
verification.

This function is the single gate: terminal status was already enforced at `developing→reviewing`,
so by this point the only failure mode is missing verification. The function returns a
descriptive error rather than a `GateResult` because it is called from within the halt-state
branch, not from the main gate-check path.

### Backward compatibility

| Scenario | Before | After |
|---|---|---|
| `require_human_review` absent | halts at reviewing | auto-advances if all tasks verified |
| `require_human_review: false` | halts at reviewing | auto-advances if all tasks verified |
| `require_human_review: true` | halts at reviewing | halts at reviewing (identical) |
| explicit `status: reviewing` transition | works | works (unchanged) |
| explicit `status: done` from reviewing | works | works (unchanged) |

The `reviewing→done` gate (`checkReviewReportExists`, FR-008) is on the explicit transition
path, not the advance stop-state path. It is unaffected.

---

## Alternatives Considered

### Alternative 1 (recommended): Project-level config flag in `MergeConfig`

Described in full in the Design section above. A `*bool` field in `MergeConfig`, defaulting to
nil (false), with a `RequiresHumanReview()` accessor. The advance engine reads it through the
`AdvanceConfig` injection point.

**Easier:** mirrors P24's `RequireGitHubPR` exactly — no new patterns, no new config structs.
One field, one accessor, one injection site. Consistent vocabulary across gates.

**Harder:** changing the default from "always halt" to "auto-advance when verified" is a
semantic breaking change for existing projects. Teams relying on the implicit halt must
explicitly set `require_human_review: true`. The blast radius is low for this project.

**Why chosen:** Simplest correct implementation. Consistent with the established pattern.
Agentic-first default matches the project's philosophy and the only known use case.

---

### Alternative 2: Remove the reviewing stop state entirely

Remove `"reviewing"` from `advanceStopStates`. The advance engine would proceed through
reviewing to done whenever the gate passes — but the `reviewing→done` gate currently requires
a review report document (`checkReviewReportExists`). This would need to be relaxed or replaced
for agentic pipelines.

**Easier:** no new config flag. Simpler advance logic.

**Harder:** removes the reviewing stage as a meaningful pause point for all pipelines, not just
agentic ones. Projects that run a formal reviewer-agent panel (non-agentic workflows) lose their
halt. The review report gate would need its own bypass mechanism, reintroducing the same
complexity elsewhere. This is architectural deletion, not surgical configuration.

**Why rejected:** Too broad. Eliminates reviewer-panel workflows entirely. The reviewing stage
exists for a reason; the problem is not its existence but its unconditional mandatory status.

---

### Alternative 3: Make `advance` skip reviewing when tasks have verification (no config)

Always bypass the reviewing halt when all tasks have verification. No config flag.

**Easier:** no config plumbing. Simpler call sites.

**Harder:** removes human override entirely from the agentic path — projects cannot opt back in
to mandatory human review without forking the binary. Makes the behaviour non-configurable,
which is inconsistent with the project's philosophy of explicit gates. Treating all projects as
agentic regardless of their workflow is premature.

**Why rejected:** Non-configurable behavior is an anti-pattern for a workflow engine. Projects
must be able to express that they require human review. The absence of a flag makes that
impossible.

---

### Alternative 4: Feature-level `require_human_review` config

Add a `Config` sub-struct to the `Feature` entity model and write `require_human_review` there
instead of in project-level `MergeConfig`. Individual features could then opt in or out of
human review independently.

**Easier:** per-feature granularity is more expressive. A project can have one high-risk feature
that always requires human review alongside many routine features that auto-advance.

**Harder:** requires changes to the `Feature` model (new `Config` field), the YAML schema, the
entity tool's create/update handlers, the advance engine's feature-loading path, and any schema
validation. Significantly more implementation surface for a use case that does not yet exist.
The only current use case is project-wide agentic pipelines.

**Why rejected:** YAGNI. The per-feature use case is hypothetical. Project-level config
satisfies all known requirements. If per-feature granularity is needed in the future, it can be
added as an override layer on top of the project default.

---

### Alternative 5: Status quo (no change)

Continue requiring `override: true` on every agentic merge. Document this as the expected
workflow for agentic pipelines.

**Easier:** no implementation.

**Harder:** every agentic merge is recorded as an override, diluting the override audit trail.
Agents must be reminded to include `override: true` and `override_reason` on every advance
call. The `override` mechanism loses its signal value over time as the ratio of exceptional
to routine overrides approaches zero.

**Why rejected:** The problem statement is valid. Override-as-normal-path is a design smell.
The cost of fixing it is low and the P24 pattern makes the implementation path clear.

---

## Dependencies

- **P24 — `require_github_pr`:** This design directly mirrors the pattern established in P24.
  `MergeConfig`, `RequireGitHubPR`, and the `RequiresGitHubPR()` accessor are the structural
  templates. P24 must be merged before or alongside this feature (it is already merged as of
  P25 planning).

- **P6 — Propagate task verification to feature entity:** P6 and P7 address the same root
  problem (verification data not surfacing to the merge path) from different angles. P6 writes
  a feature-level `verification` field from task `finish()` aggregation; P7 uses per-task
  `verification` fields to bypass the reviewing halt. The two are independent and can be
  implemented in either order. P7 does not depend on P6.

- **`internal/service/advance.go`:** The advance engine is the primary change site. The
  `AdvanceConfig` struct and the halt-state branch must be modified. Existing tests cover the
  mandatory-halt behaviour (`TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing`) and will
  need updating to reflect the new conditional logic.

- **`internal/service/prereq.go`:** New `checkAllTasksHaveVerification` helper. The existing
  `checkAllTasksTerminal` helper is the structural model.

- **`internal/mcp/entity_tool.go`:** The MCP layer must inject `RequiresHumanReview` into the
  `AdvanceConfig` it constructs before calling `AdvanceFeatureStatus`. This is the only MCP
  change required.

- **`internal/config/config.go`:** New field and accessor on `MergeConfig`. No schema version
  bump required — the field is additive and `omitempty` ensures existing config files remain
  valid.

---

## Open Questions

None blocking this design. The following are noted for the specification stage:

1. **Partial verification:** If some tasks have verification and some do not, the advance halts
   at reviewing with a message naming the unverified tasks. Should the message be a warning
   that allows override, or a hard block? Proposal: hard block with a clear actionable message,
   consistent with how other gate failures behave.

2. **Zero tasks:** If a feature has no tasks (tasks were never created), should reviewing be
   treated as auto-advanceable? Proposal: yes — no tasks means no missing verification. The
   condition "all tasks have verification" is vacuously true. This matches the behaviour of
   `checkAllTasksTerminal` for empty task lists.

3. **`needs-review` task status:** A task completed to `needs-review` rather than `done` is
   terminal but may or may not have a verification field. The auto-advance condition requires
   a non-empty verification field regardless of terminal status variant. This is intentional:
   `needs-review` implies human attention is wanted; if that task is part of a feature, the
   feature should not auto-advance past reviewing.
```

Now I'll register the document: