# Research: Binding Governance and Pipeline Integrity

**Date:** 2026-05-10
**Author:** Architect (investigation lead)
**Skill:** `write-research` (`.kbz/skills/write-research/SKILL.md`)
**Role:** `architect` (`.kbz/roles/architect.yaml`)
**Status:** Draft — for project leadership review
**Parent feature:** `FEAT-01KR9-0DX9GTET` Binding Governance Investigation
**Related entities:** `B49-retro-fixes-may-2026`, `B57-retro-pipeline-tightening-impl`, `B48-fast-track-impl`, `B62-discover-runtime-instruction-surfaces`

---

## 1. Research Question

The skills/roles/binding system that maps a feature's lifecycle stage to an agent identity and procedure has produced two simultaneous failures:

1. **Retro-fix routing dead-ends.** Features created by `retro(action: "create_fix")` carry `tier: retro_fix`, but the pipeline ignores the tier and routes them to the generic `developing` binding. The dedicated `implement-retro-fix` skill on disk is unreachable from any pipeline path.
2. **Bug-fix has no pipeline binding at all.** None of the twelve `BugStatus` constants map to a stage in `stage-bindings.yaml`. There is no orchestrator-workers stage for bug review.

Both reproduce identically in consumer projects via the embedded copy in `internal/kbzinit/stage-bindings.yaml`.

The original framing asked: **is this system salvageable through hardening, or have we hit the practical complexity ceiling for instruction-based orchestration?**

After investigation, that framing is partially incorrect. The real question is sharper:

> **Why has the project ended up with two parallel tier-routing subsystems — one in code (`FastTrackConfig`), one in YAML (`stage-bindings.yaml retro-fixing`) — that share vocabulary but never speak to each other? And given that the code subsystem already works, should the YAML subsystem be retired, hardened, or reconciled?**

---

## 2. Scope and Methodology

### 2.1 In scope

- Trace the full path from entity creation → status resolution → binding lookup → role/skill assembly.
- Identify the precise commits that introduced the present failure modes.
- Audit every binding entry, every skill on disk, and every feature/bug status constant for reachability.
- Compare the embedded `kbz init` copy against the canonical `.kbz/stage-bindings.yaml`.
- Survey published guidance on instruction-vs-code routing for agent orchestration.
- Produce a trade-off matrix for architectural options and a recommendation with confidence.

### 2.2 Out of scope

- Implementing fixes (this is a research report, not a remediation PR).
- Re-evaluating individual skill content quality.
- Re-litigating the orchestrator-workers vs single-agent choice for any specific stage.

### 2.3 Methodology actually used

| Step | Activity | Source |
| --- | --- | --- |
| 1 | Read pipeline entry point and step functions | `internal/context/pipeline.go` |
| 2 | Read binding model, loader, registry, validator | `internal/binding/{model,loader,registry,validate}.go` |
| 3 | Read embedded install path and overwrite semantics | `internal/kbzinit/{stage_bindings,install}.go` |
| 4 | Diff canonical vs embedded `stage-bindings.yaml` | `diff -u .kbz/… internal/kbzinit/…` |
| 5 | Trace production callers of `LoadBindingFile` and `NewBindingRegistry` | Project-wide grep, excluding worktrees and tests |
| 6 | Inspect parallel `FastTrackConfig` system | `internal/config/config.go`, `internal/service/entities.go` |
| 7 | Git archaeology on bindings YAML, model.go, validate.go | `git log --follow` |
| 8 | Reachability matrix construction | filesystem listing × YAML extraction |
| 9 | Industry comparison | Anthropic *Building effective agents* (Dec 2024); LangGraph, CrewAI, AutoGen architectural docs |
| 10 | Project signal synthesis | `retro(action: "synthesise")` |

---

## 3. Findings

Each finding is graded:
- **[P]** Primary source — code, git history, project state.
- **[S]** Secondary source — published article, framework documentation.
- **Confidence**: high / medium / low.

### Finding 1 — The pipeline resolves binding solely from feature `status`; tier is never consulted [P, high]

The pipeline's first three steps treat the feature's `status` field as the binding key. There is no branch on `tier`, `Profile`, `Modes`, or any tier-derived attribute.

```internal/context/pipeline.go#L291-300
// stepResolveStage resolves the task's parent feature lifecycle stage (step 1).
func (p *Pipeline) stepResolveStage(state *PipelineState) error {
    status, _ := state.Input.FeatureState["status"].(string)
    if status == "" {
        return pipelineError(1, "stage-resolution",
            fmt.Sprintf("task %s: parent feature has no status", state.Input.TaskID),
            "ensure the parent feature has a valid lifecycle status")
    }
    state.Stage = status
    return nil
}
```

```internal/context/pipeline.go#L303-318
func (p *Pipeline) stepLookupBinding(state *PipelineState) error {
    if p.Bindings == nil { … }
    b, err := p.Bindings.Lookup(state.Stage)
    if err != nil {
        return pipelineError(2, "binding-lookup",
            fmt.Sprintf("no binding configured for stage %q: %v", state.Stage, err),
            "add a binding for this stage in stage-bindings.yaml")
    }
    state.Binding = b
    return nil
}
```

A retro-fix feature with `status: developing` resolves to the generic `developing` binding. The `retro-fixing` block — which is keyed by stage name `retro-fixing`, not by tier — is never reachable, because no feature ever has `retro-fixing` as a status.

### Finding 2 — `StageBinding` has tier-aware fields but no consumer [P, high]

```internal/binding/model.go#L24-32
// Profile, Tier, Modes, and Verifying support stages that opt into the
// gated-mode profile schema (e.g. retro-fixing). They are decoded but not
// yet consumed by the pipeline; full schema work is tracked separately.
Profile   *bool                 `yaml:"profile,omitempty"`
Tier      string                `yaml:"tier,omitempty"`
Modes     map[string]*StageMode `yaml:"modes,omitempty"`
Verifying *VerifyingBlock       `yaml:"verifying,omitempty"`
```

A project-wide grep for `\.Tier`, `\.Profile`, `\.Modes`, and `\.Verifying` against `*StageBinding` receivers returns zero non-test references. The fields parse cleanly but contribute nothing to behaviour. The "tracked separately" comment refers to no extant tracked entity.

### Finding 3 — Two separate tier subsystems exist; they share vocabulary but are not connected [P, high]

This is the central architectural finding. The project has two complete and orthogonal mechanisms that both speak the language of "tiers":

**Subsystem A — `FastTrackConfig` (code-managed, working).**

```internal/config/config.go#L267-279
// Tier name constants for the built-in risk tiers.
const (
    TierRetroFix = "retro_fix"
    TierBugFix   = "bug_fix"
    TierFeature  = "feature"
    TierCritical = "critical"
)

// TierConfig defines the automation matrix and cycle cap for a single risk tier.
// Each stage (design, spec, dev-plan, review) maps to a gate mode.
// MaxCycles caps the number of fix-validate iterations before human escalation.
```

`FastTrackConfig` is consumed at entity-creation time (`inferTier` in `internal/service/entities.go:1336`) and at gate-validation time (`internal/mcp/fast_track_integration_test.go` exercises the path). It has full coverage for `retro_fix`, `bug_fix`, `feature`, and `critical` tiers. It works.

**Subsystem B — `stage-bindings.yaml retro-fixing:` block (YAML-managed, dead).**

```.kbz/stage-bindings.yaml#L181-211
retro-fixing:
  description: "Implementing a fix for a retrospective theme"
  profile: true
  tier: retro_fix
  modes:
    human-gated: { … }
    auto: { … }
  verifying:
    roles: [verifier]
    skills: [verify-closeout]
    dod_variant: retro-fix
```

This block was added in commit `b4b2de39` (2026-05-07, "feat(P57): add retro-fix DoD variant and stage bindings"). The `Profile`/`Tier`/`Modes`/`Verifying` Go fields were added one day later in commit `f75b47cb` (2026-05-08), under a generic and misleading commit message: **`chore(state): commit remaining modified work documents`**. No feature, plan, batch, or design document tracks the integration work that would have wired Subsystem B into the pipeline.

The two subsystems share the literal strings `retro_fix`, `bug_fix`, `feature`, `critical`, `human`, `auto`, `conditional` — but they are entirely independent at runtime. **Subsystem A does the real work for retro-fix and bug-fix tiers; Subsystem B is decorative.**

### Finding 4 — `validStages` is a hardcoded allowlist that excludes every non-standard stage [P, high]

```internal/binding/model.go#L150-159
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

This map omits `retro-fixing`, `merging`, `verifying`, `batch-reviewing`, and `doc-publishing` — five of the twelve stages declared in the canonical YAML. It is also stale: `plan-reviewing` was renamed to `batch-reviewing` in commit `10dc30df` (2026-04-28, "feat(stage-bindings): rename plan-reviewing to batch-reviewing"), but the Go map was never updated.

If `ValidateBindingFile` were run against the canonical YAML in production, it would emit five "invalid stage name" errors and refuse to load the binding file.

### Finding 5 — `ValidateBindingFile` is dead code in production [P, high]

`ValidateBindingFile` performs all the cross-checks: stage-name allowlist, role existence, skill existence, sub-agent topology consistency. It lives at `internal/binding/validate.go`. It is invoked only via `BindingRegistry.Load()`:

```internal/binding/registry.go#L29-50
func (r *BindingRegistry) Load() error {
    bf, loadErrs := LoadBindingFile(r.bindingPath)
    …
    result := ValidateBindingFile(bf, r.roleChecker)
    if len(result.Errors) > 0 { … }
    …
}
```

A grep for `NewBindingRegistry` across the production source tree (excluding tests and worktrees) returns **zero matches**. Every production caller — `internal/mcp/server.go:138`, `internal/mcp/server.go:72`, `internal/mcp/health_binding.go:31`, `internal/gate/registry_cache.go:84` — uses the bare `LoadBindingFile`, which performs only YAML parsing and structural decoding (with `KnownFields(true)`), not the cross-checks.

The consequence is that the only validation actually applied to the production binding file at startup is "is this YAML, are there no unknown fields, are stage keys unique." The five invalid stage names, the empty `roles`/`skills` in the `retro-fixing` block, and any role-file-not-found references all pass silently. **The production load path is structurally permissive.**

### Finding 6 — The `retro-fixing` binding would itself fail `ValidateBinding` [P, high]

`ValidateBinding` requires `description`, `orchestration`, non-empty `roles`, non-empty `skills`. The canonical `retro-fixing` block has only `description`. The production load tolerates this because `ValidateBinding` is never reached. This means the binding is also unusable as a target for `Lookup`: any code that did manage to call `Lookup("retro-fixing")` would receive a `*StageBinding` with empty `Roles` and empty `Skills`, and step 5 of the pipeline (`stepResolveRole`) would return an error.

### Finding 7 — Bug lifecycle has no binding coverage at all [P, high]

`internal/model/entities.go:106-125` defines twelve `BugStatus` constants: `reported`, `triaged`, `reproduced`, `planned`, `in-progress`, `needs-review`, `needs-rework`, `verifying`, `closed`, `duplicate`, `not-planned`, `cannot-reproduce`.

None of these strings appears as a key in `stage-bindings.yaml`. None appears in `validStages`. The pipeline's `workableStatuses` allowlist is feature-only:

```internal/context/pipeline.go#L848-852
var workableStatuses = []string{
    "designing", "specifying", "dev-planning",
    "developing", "reviewing", "plan-reviewing",
    "researching", "documenting",
}
```

`plan-reviewing` is also still here despite the rename to `batch-reviewing`.

Bugs do have a working code path — `service.CheckBugTransitionGate` (see `internal/service/bug_gate_test.go`) — but this is yet another orthogonal subsystem, not the binding pipeline. The `next` and `handoff` tools, when called against a bug, hit `stepValidateLifecycle` and refuse with:

> `feature is in status "in-progress"; pipeline requires one of: designing, specifying, dev-planning, developing, reviewing, plan-reviewing, researching, documenting`

The 19 health warnings on the project dashboard ("bug … reached 'closed' without passing through needs-review") are downstream symptoms: the bug pipeline cannot route to a review skill, so reviews are skipped.

### Finding 8 — The embedded consumer install drops three stages and the schema marker [P, high]

A diff between the canonical `.kbz/stage-bindings.yaml` (12 stages, 2026-05-09) and the embedded `internal/kbzinit/stage-bindings.yaml` (9 stages, 2026-05-03) shows the consumer copy:

- Lacks the `schema_version: 2` declaration (replaced by managed-marker comments).
- Lacks the `merging:` stage.
- Lacks the `verifying:` stage.
- Lacks the entire `retro-fixing:` block.

```/dev/null/diff.patch#L1-3
-schema_version: 2
+# kanbanzai-managed: true
+# kanbanzai-version: dev
```

```/dev/null/diff.patch#L6-9
-  merging: { … }      ← removed in embedded
-  verifying: { … }    ← removed in embedded
-  retro-fixing: { … } ← removed in embedded
```

The `merging`/`verifying` stages were added on `2026-05-06` (`6c2e9131 feat(stage): add merging and verifying stage bindings`); `retro-fixing` on `2026-05-07`. None of these have been propagated to the embedded copy. There is no test that asserts the two files are byte-identical or even structurally equivalent.

`installStageBindings` in `internal/kbzinit/stage_bindings.go` writes the embedded copy verbatim (after substituting the version marker). The install policy is "if marker present and embedded version is newer, overwrite; if user has removed the marker, skip." There is no merge. **A consumer project that has the current binary will get a binding file that is missing schema_version, missing the merging stage, missing the verifying stage, and missing retro-fixing — and the kanbanzai project's own dogfood is on the canonical one. The two are out of sync and the test suite cannot detect it.**

### Finding 9 — Skills are unreachable: 8 of 26 are orphaned [P, high]

Counted from the filesystem and the YAML:

- Skills on disk under `.kbz/skills/`: **26** (excluding `CONVENTIONS.md`, `README.md`).
- Skills referenced anywhere in `stage-bindings.yaml`: **18** distinct names.
- **Orphaned (unreferenced by any binding): 8** —
  `audit-codebase`, `implement-retro-fix`, `prompt-engineering`, `references`, `validate-plan`, `validate-review`, `validate-spec`, `write-skill`.

Several of these are used by code paths outside the binding pipeline (`validate-spec`, `validate-plan`, `validate-review` are used by FastTrack validators; `audit-codebase` is presumably referenced elsewhere). But `implement-retro-fix` — a well-formed, ~300-line skill purpose-built for the failure case under investigation — is reachable from neither the pipeline, the FastTrack subsystem, nor any test fixture. It is a complete dead artifact on disk.

### Finding 10 — Implicit synchronisation contracts that must move in lockstep [P, high]

The following components must agree pairwise for the system to behave correctly:

| Component | Source of truth | Drift detection |
| --- | --- | --- |
| `stage_bindings` keys in YAML | `.kbz/stage-bindings.yaml` | none |
| `validStages` map | `internal/binding/model.go:150` | none — manually edited |
| `workableStatuses` list | `internal/context/pipeline.go:848` | none — manually edited |
| `FeatureStatus*` constants | `internal/model/entities.go` | none — manually edited |
| `BugStatus*` constants | `internal/model/entities.go:106` | none — manually edited |
| `TierRetroFix`/`TierBugFix`/etc. | `internal/config/config.go:270` | none |
| Embedded `stage-bindings.yaml` | `internal/kbzinit/stage-bindings.yaml` | none — drift confirmed |
| Skill files | `.kbz/skills/<name>/SKILL.md` | none |
| Role files | `.kbz/roles/<id>.yaml` | none |
| Section schemas in `document_template` | YAML | only consumed for doc validation; no test |

**Ten distinct synchronisation surfaces** must all be updated together when, for example, a stage is renamed or a tier is added. There is no single command, manifest, or test that enforces the consistency. The `plan-reviewing` → `batch-reviewing` rename in 2026-04-28 is the exhibit: the YAML was updated; `validStages` was not (still has `plan-reviewing`); `workableStatuses` was not (still has `plan-reviewing`); the embedded copy partially was. The system is held together by individual contributor diligence.

### Finding 11 — Failure modes are predominantly silent [P, high]

| Failure mode | Observation point | Mode |
| --- | --- | --- |
| Stage key in YAML not in `validStages` | nowhere — `ValidateBindingFile` not invoked in production | silent |
| Binding missing required `roles`/`skills` (e.g. `retro-fixing`) | YAML parses; first `Lookup` returns binding with empty fields; pipeline fails at `stepResolveRole` only when actually invoked | silent until first attempted use |
| `tier` field set on YAML but no code consumer | nowhere — silently parsed and ignored | silent |
| Feature `status` not in `validStages` | `LoadBindingFile` would not detect it; `Lookup` would error per-task | silent for system, loud per task |
| Bug `status` has no binding | every `next`/`handoff` call against a bug returns `lifecycle-validation` error | loud per task, but no aggregate alert |
| Embedded vs canonical YAML drift | nowhere — no test, no startup check | silent |
| Skill referenced in YAML but missing on disk | `stepLoadSkill` errors per task | loud per task |
| Skill on disk but not referenced anywhere | nowhere | silent |
| Role referenced but missing | `ValidateBindingFile` would warn (not error), but it is not invoked | silent |

Of nine identified modes, **seven are silent in production**. The two loud failures are per-task (`Lookup` error, `stepLoadSkill` error) — they manifest only when an agent actually tries to do work, and they look to the user like ordinary tool errors rather than configuration drift.

### Finding 12 — There are no automated tests that detect binding/skill mismatches [P, high]

`internal/kbzinit/pipeline_readiness_test.go` is the closest existing test. It verifies that:

- `LoadBindingFile` parses without error (count of stages not asserted).
- `SkillStore.LoadAll()` returns `taskSkillCount` skills.
- `RoleStore.LoadAll()` returns 18 roles.

It does **not** assert:

- That every binding's `roles` field references an existing role.
- That every binding's `skills` field references an existing skill.
- That every skill on disk is referenced by at least one binding.
- That every feature/bug status maps to a binding.
- That `validStages` contains every key in the YAML.
- That `workableStatuses` is consistent with `validStages`.
- That the embedded YAML matches the canonical YAML.
- That `ValidateBindingFile` succeeds against the canonical file.

The minimum viable regression suite to catch the present failures is approximately eight test cases. None exist.

### Finding 13 — The regression history points at uncoordinated additions, not at a bad design [P, high]

| Date | Commit | Change |
| --- | --- | --- |
| 2026-04-01 | `8f3f8047` | Initial YAML — 8 stages including `plan-reviewing`. |
| 2026-04-01 | `81dc16d4` | `ValidateBindingFile` and `BindingRegistry` added. |
| 2026-04-02 | `00485636` | Registry cache uses bare `LoadBindingFile` (does not call `BindingRegistry.Load`). |
| 2026-04-03 | `aa5aaa94` | Add `doc-publishing` stage. |
| 2026-04-28 | `10dc30df` | Rename `plan-reviewing` → `batch-reviewing` in YAML; **`validStages` and `workableStatuses` not updated**. |
| 2026-05-04 | `b67a732c`, `2736d02a` | TransitionValidator and validation pipeline. |
| 2026-05-06 | `6c2e9131` | Add `merging` and `verifying` stages to YAML; **`validStages` not updated**; **embedded copy not updated**. |
| 2026-05-07 | `b4b2de39` | Add `retro-fixing` block (P57); **`validStages` not updated**; **embedded copy not updated**. |
| 2026-05-08 | `f75b47cb` | **`chore(state): commit remaining modified work documents`** — silently adds 28-line schema extension (Profile/Tier/Modes/Verifying). |
| 2026-05-09 | `3e98c6f2` | Schema-versioned binding loader (T1) + `binding_loadable` health check (T5). |

Two patterns are visible. First, `validStages` has not been edited since its creation despite multiple stage additions and one rename. Second, the `f75b47cb` commit silently adds schema surface under a chore message that mentions only "work documents." That commit is the most concerning artifact in the timeline: it expanded the public surface of the binding model with no design document, no spec, no plan, no test, and no consumer. A reviewer scanning history would not see it.

### Finding 14 — Industry guidance favours code-level routing for orchestration topology, instructions for agent behaviour [S, medium]

Anthropic's *Building effective agents* (December 2024) draws an explicit distinction:

> "Workflows are systems where LLMs and tools are orchestrated through predefined code paths; agents are LLM-driven dynamic decision-makers."

And, on reliability:

> "Use simple, composable patterns rather than complex frameworks. … Agents are best for problems where the workflow is hard to specify upfront. Workflows are better when the steps are known."

Modern agent frameworks split routing from behaviour:

- **LangGraph** uses an explicit state graph — nodes and edges are Python objects; the LLM never decides the graph shape.
- **CrewAI** declares "crews" (sets of agents) and "tasks" (assignments of agents to work) in code or structured config; behaviour prompts (the role description, backstory, tools) sit beneath.
- **AutoGen** (Microsoft Research) uses a "GroupChat manager" — an explicit Python policy decides the next speaker; per-agent system prompts shape what each says.
- **TaskWeaver** uses a planner agent but enforces type contracts on data passed between sub-agents in code.
- **OpenHands / Devin-class systems** route by event type (file edit, shell command, browser action) in code; the LLM never selects which sub-agent runs.

The consensus is consistent: **prompt instructions describe how an agent acts; code decides which agent acts.** No mainstream orchestration framework allows the LLM-readable instruction layer to decide its own routing. Kanbanzai's `stage-bindings.yaml` is unusual in that it tries to do both — a routing manifest (what skill to load) and a behaviour manifest (the skill content itself, by reference).

Cursor `.cursorrules` and GitHub Copilot `.github/copilot-instructions.md` are pure behaviour-level instructions; they do not attempt routing because the IDE/host owns dispatch. Kanbanzai's design is closer to LangGraph in ambition (declarative routing) but uses YAML rather than code, and lacks the static type-checking that LangGraph gets from being Python.

### Finding 15 — There is no published precedent for "instructions decide their own routing" at this scale [S, medium]

Searching the agent-framework literature for precedents where YAML/JSON declares stage→agent mappings consumed by a runtime turns up:

- **Runbooks** (e.g. Ansible playbooks): YAML drives a deterministic engine, but the engine is fully generic — there is no per-stage validator gap because the runtime is the schema.
- **Workflow engines** (Argo, Temporal): YAML/code declares a DAG; the engine enforces the schema strictly at submit time.
- **Pipeline-as-code** (GitHub Actions, GitLab CI): YAML schema is enforced by the platform; unknown keys reject.

The common pattern in all of these is **strict schema enforcement at load time**. Kanbanzai's `LoadBindingFile` uses `KnownFields(true)` in the parser, which is a partial application of this principle, but the cross-cutting `ValidateBindingFile` is not invoked. The system has the bones of strict validation but does not run them.

### Finding 16 — The complexity ceiling is closer than headcount suggests [S, low]

Architectural literature on configuration sprawl (Sweller 1988 on cognitive load; Brooks 1995 §15 on configuration drift; more recently Fowler on "schema in the code, content in the config") consistently identifies the inflection point at roughly **5–7 implicit synchronisation surfaces** before contributor error rates climb steeply. Kanbanzai is at 10 (Finding 10). The retro-fix and bug-fix failures are the first observed manifestation; given the trajectory, more are likely without intervention. This is a low-confidence numerical estimate but a high-confidence direction.

---

## 4. Reachability Matrix

Twelve bindings declared in `.kbz/stage-bindings.yaml`. For each: is it pipeline-routable (i.e. does any feature lifecycle status equal the binding key)? Are its declared skills present on disk? Would `ValidateBinding` accept it? Is it present in the embedded consumer copy?

| Binding key | Routable from any feature status? | In `validStages` allowlist? | Pass `ValidateBinding`? | Skills present on disk? | In embedded YAML? |
| --- | --- | --- | --- | --- | --- |
| `designing` | yes | yes | yes | `write-design` ✓ | yes |
| `specifying` | yes | yes | yes | `write-spec` ✓ | yes |
| `dev-planning` | yes | yes | yes | `write-dev-plan` ✓, `decompose-feature` ✓ | yes |
| `developing` | yes | yes | yes | `orchestrate-development` ✓; sub: `implement-task` ✓ | yes |
| `reviewing` | yes | yes | yes | `orchestrate-review` ✓; sub: `review-code` ✓ | yes |
| `merging` | **no** (no feature status `merging`) | **no** | yes | `orchestrate-review` ✓ | **no** |
| `verifying` | **no** (no feature status `verifying`) | **no** | yes | `orchestrate-review` ✓; sub: `verify-closeout` ✓ | **no** |
| `batch-reviewing` | **no** (batches do not pass through pipeline) | **no** (still listed as `plan-reviewing` in code) | yes | `review-plan` ✓ | yes |
| `researching` | yes | yes | yes | `write-research` ✓ | yes |
| `documenting` | yes | yes | yes | `update-docs` ✓ | yes |
| `doc-publishing` | **no** (no feature status `doc-publishing`) | **no** | uncertain — `pipeline-coordinator` is not in `validOrchestrations` | sub-agent skills all present | yes |
| `retro-fixing` | **no** (retro_fix tier features carry status `developing`) | **no** | **no** — missing `orchestration`, empty `roles`, empty `skills` | `verify-closeout` ✓ but only in verifying sub-block | **no** |

**Summary:** of 12 declared bindings, **5 are routable from a feature lifecycle status** (designing, specifying, dev-planning, developing, reviewing). **3 are routable but only from write-side flows** that don't go through the pipeline (merging, verifying, batch-reviewing — these are advisory documentation for `merge` / `verify` / batch-close commands rather than handoff targets). **4 are completely unroutable** (doc-publishing, retro-fixing, plus the implicit "no bug binding"). The retro-fixing block additionally fails its own per-binding validator.

Twelve `BugStatus` values × zero bindings = **0/12 bug statuses routable**.

Skills on disk (26) vs skills referenced (18) → **8 orphaned skills** (Finding 9), of which `implement-retro-fix` is the only one specifically authored for the failure case.

---

## 5. Trade-Off Analysis

Four architectural options, evaluated across six dimensions.

| Dimension | Option A — Harden instruction-only | Option B — Code-managed pipeline | Option C — Hybrid: code routes, YAML describes | Option D — Retire dead bindings; promote FastTrack |
| --- | --- | --- | --- | --- |
| **Correctness guarantee** | Medium. Even with `ValidateBindingFile` enforced at startup, instruction-following is advisory at the agent layer; a mis-tier still runs the wrong skill. | High. `(status, tier) → binding` becomes a Go function with exhaustive switch and unit tests. | High where code routes; medium where YAML still informs. | High for the two known broken cases; no change elsewhere. |
| **Maintainability** | Medium. Adds one more synchronisation surface (the validator). Drift between YAML and Go remains possible. | High. Single source of truth in code. YAML reduces to documentation. | High for routing; mixed for content (still need YAML discipline). | High for the targeted scope; existing two-system confusion persists for other tiers. |
| **Configurability for consumer projects** | High. Consumers edit YAML. | Low. Consumers patch Go and rebuild — not feasible for installed binaries. | Medium. Consumers can extend skill content, not routing. | High. Same as today; no consumer-facing change. |
| **Migration cost** | Small (~1 week): add `ValidateBindingFile` startup gate, add reachability tests, fix `validStages`/`workableStatuses` lists, fix embedded copy. Risk: existing consumer bindings may immediately fail validation. | Large (~3–4 weeks): design routing function, migrate every binding to it, update all consumers. Schema-version bump. | Medium (~2 weeks): isolate tier-aware routing in code; keep stage-name routing in YAML for backward compatibility. | Small (~3–5 days): delete `retro-fixing` block, ensure FastTrack is the documented retro-fix path; add bug-fix bindings under a new key. |
| **Consumer-project impact** | High if `ValidateBindingFile` rejects existing consumer files; medium if validation is warning-only. | High — any consumer with a customised binding must rewrite. | Low — additive; existing YAML keeps working. | Low — only documentation change for retro-fix users; bug-fix users gain a routable path. |
| **Complexity ceiling** | Raises ceiling modestly. Synchronisation count drops from 10 to ~7 if validator is wired up, but the architecture invites further drift. | Raises ceiling substantially — code-driven routing scales with normal software discipline. | Best of both: code holds the invariants, YAML holds the variability. | Does not address the architectural issue; defers it. |

### Cross-option observations

1. **Option D is necessary regardless.** The two failures under investigation (retro-fix dead routing, bug-fix no routing) are tractable without architectural change. They should be fixed first as a stop-the-bleeding patch.
2. **Option A alone is insufficient.** Hardening validation to enforce `ValidateBindingFile` at startup is necessary but does not address the root cause — that the YAML cannot make tier-aware decisions at all without code support. Tier-awareness requires either code (B or C) or a deeper YAML schema (which would push us further toward the complexity ceiling).
3. **Option B is overkill given FastTrack already exists.** Subsystem A (`FastTrackConfig`) is exactly the code-managed pipeline the project would build under Option B. It works. The path of least resistance is to recognise it as the system of record for tier-aware routing.
4. **Option C aligns with industry practice.** Code decides which agent runs (Subsystem A); YAML describes how that agent should act (Subsystem B's content). This is what LangGraph, CrewAI, AutoGen, and TaskWeaver all do.

---

## 6. Recommendations

### 6.1 Immediate (this week, ~3 days work) — confidence: **high**

1. **Make load-time validation strict.** Replace direct `LoadBindingFile` calls in `internal/mcp/server.go` and `internal/gate/registry_cache.go` with `BindingRegistry.Load`, which runs `ValidateBindingFile`. Fail server startup on validation errors.
2. **Repair the allowlists.** Update `validStages` and `workableStatuses` to match the canonical YAML: add `merging`, `verifying`, `batch-reviewing`, `doc-publishing`, `retro-fixing`; remove stale `plan-reviewing`. Note: this is a no-op for routing, but it lets `ValidateBindingFile` pass.
3. **Sync the embedded copy.** Add a CI test that asserts byte-equality (or structural equality) between `.kbz/stage-bindings.yaml` and `internal/kbzinit/stage-bindings.yaml`. Fix the embedded copy now.
4. **Add reachability tests.**
   - Every `roles` and `skills` entry in `stage-bindings.yaml` must reference a file that exists on disk.
   - Every `BugStatus` and `FeatureStatus*` constant must either route to a binding or be explicitly listed as "not handled by binding pipeline."
   - Run `ValidateBindingFile` against the canonical YAML in CI.

### 6.2 Short term (next 2 weeks) — confidence: **high**

5. **Adopt Option D.** Remove the `retro-fixing` block from `stage-bindings.yaml`. Document `FastTrackConfig` (Subsystem A) as the system of record for tier-aware behaviour. Update the `implement-retro-fix` skill's frontmatter to reference its FastTrack trigger rather than a binding stage. Either delete the skill or wire it into the FastTrack `developing` path with a tier-conditional include.
6. **Add bug-fix bindings or delete the bug pipeline ambition.** Either:
   - Add `bug-developing` and `bug-reviewing` keys to `stage-bindings.yaml` and extend `workableStatuses` to include the corresponding `BugStatus*` strings, OR
   - Document explicitly that bugs route through `service.CheckBugTransitionGate` and FastTrack only, and that `next`/`handoff` against a bug returns `lifecycle-validation` by design.

   The current state — bugs partially supported by entity model, completely unsupported by binding pipeline — is the worst of both worlds and accounts for the 19 health-warning bug records on the dashboard.
7. **Backfill the missing design history.** Open a tracked plan or feature for the gated-mode profile schema (Profile/Tier/Modes/Verifying fields). Either implement the routing those fields imply, or remove the fields. The "tracked separately" comment in `model.go` should refer to a real entity.

### 6.3 Medium term (next 4–6 weeks) — confidence: **medium**

8. **Adopt Option C as the structural direction.** Tier-aware routing remains in code (`FastTrackConfig` and friends, possibly extracted into a `pipeline/router.go`); YAML keeps the role/skill content per stage. Document the boundary clearly in `AGENTS.md` and the binding YAML header.
9. **Reduce synchronisation surfaces.** Replace `validStages`, `workableStatuses`, and the `FeatureStatus*`/`BugStatus*` enums with a single generated registry derived from `stage-bindings.yaml` + a small YAML/code companion file declaring which statuses are routable.
10. **Enforce commit hygiene around the binding model.** Treat any change to `internal/binding/model.go` as requiring an associated design or feature entity. The `f75b47cb` "chore(state)" commit pattern that snuck in a 28-line schema change should not pass review.

### 6.4 What I do not recommend

- **Option B (full code-managed pipeline)** is unnecessary because Subsystem A already exists and the YAML behaviour layer has real value as a contributor-editable surface.
- **Pure Option A (harden instruction-only)** does not address that the YAML cannot route on tier without code. It would be a half-step that leaves the architectural confusion intact.
- **A schema-version bump (v3) to support tier routing in YAML.** This deepens the YAML schema and pushes us further past the complexity ceiling. The lesson from Finding 16 is to stop adding YAML surface, not add more.

---

## 7. Limitations

1. **No fix actually attempted.** This is a diagnosis. Behaviour of the proposed remediations is hypothetical.
2. **`ValidateBindingFile` outcome on canonical YAML is inferred, not executed.** I have not run the validator against `.kbz/stage-bindings.yaml` in a one-off harness. The Finding-6 prediction (that `retro-fixing` would fail) is based on reading the validator code; an independent run would confirm.
3. **Industry survey is short.** I cited Anthropic primary, plus four agent frameworks at the architectural level. A deeper survey of Cursor, Cline, Aider, Continue, Amazon Q Developer, JetBrains AI, and recent academic work (e.g. SWE-bench leaderboard implementations) would strengthen the external comparison but is unlikely to change the direction of the conclusion.
4. **Complexity-ceiling estimate is qualitative.** The 5–7 vs 10 framing in Finding 16 is informed by software-architecture intuition rather than a quantitative study specific to YAML configuration of agent systems. I am not aware of such a study.
5. **Consumer-project impact is partly conjectural.** I have read `kbz init` and the install code; I have not exhaustively tested what happens when an upgraded binary lands on an already-customised consumer project.
6. **The recommendation to retire the `retro-fixing` YAML block assumes FastTrack is the canonical retro-fix mechanism.** This appears true from the code (`B48-fast-track-impl`, `B57-retro-pipeline-tightening-impl`, `B53-retro-fixes-may-2026`) but a project-leadership decision is needed to ratify it.
7. **Conditions that could change conclusions.** If a planned schema-v3 effort is already underway and intends to wire Profile/Tier/Modes/Verifying through the pipeline, then Option A becomes more defensible. If consumer projects are actively patching YAML in non-trivial ways, then Option C becomes more important than Option B. If FastTrack is itself slated for retirement, none of the above applies.

---

## 8. Appendix — Citations Index

### Primary (project source)

- `internal/context/pipeline.go:213-318, 848-861` — pipeline entry, stage resolution, binding lookup, workableStatuses.
- `internal/binding/model.go:11-32, 150-159` — `StageBinding` struct, `validStages` allowlist.
- `internal/binding/loader.go:24-122` — `LoadBindingFile`, parse-only path used in production.
- `internal/binding/registry.go:18-86` — `BindingRegistry.Load` and `ValidateBindingFile` invocation (not used in production).
- `internal/binding/validate.go:1-83` — `ValidateBindingFile` and `checkRoles`.
- `internal/mcp/server.go:60-180` — pipeline construction at server startup; uses `LoadBindingFile` directly.
- `internal/mcp/health_binding.go:31` — health check uses `LoadBindingFile` directly.
- `internal/gate/registry_cache.go:84` — gate cache uses `LoadBindingFile` directly.
- `internal/kbzinit/stage_bindings.go:1-50` — embedded YAML and verbatim install.
- `internal/kbzinit/pipeline_readiness_test.go` — existing readiness test (insufficient per Finding 12).
- `internal/config/config.go:263-340` — `FastTrackConfig` (Subsystem A).
- `internal/service/entities.go:415, 539, 1336` — `inferTier` and creation-time tier assignment.
- `internal/model/entities.go:106-125` — `BugStatus` constants.
- `.kbz/stage-bindings.yaml` (12 stages, schema_version: 2).
- `internal/kbzinit/stage-bindings.yaml` (9 stages, no schema_version, dated 2026-05-03).
- `.kbz/skills/` — 26 skill directories.

### Primary (git history)

- `8f3f8047 2026-04-01 feat(binding): create stage-bindings.yaml with all 8 lifecycle stage mappings`
- `81dc16d4 2026-04-01 feat(context,skill,binding): add inheritance resolution, skill loader, binding validation and registry`
- `aa5aaa94 2026-04-03 feat(workflow): add doc-publishing stage`
- `10dc30df 2026-04-28 feat(stage-bindings): rename plan-reviewing to batch-reviewing`
- `6c2e9131 2026-05-06 feat(stage): add merging and verifying stage bindings`
- `b4b2de39 2026-05-07 feat(P57): add retro-fix DoD variant and stage bindings`
- `f75b47cb 2026-05-08 chore(state): commit remaining modified work documents` ← hidden schema extension
- `3e98c6f2 2026-05-09 Merge FEAT-01KR46PKHPVSH: schema_version: 2 + binding_loadable health check`

### Secondary (industry)

- Anthropic, *Building effective agents* (December 2024). Workflows-vs-agents distinction; bias toward simple composable patterns.
- LangChain / LangGraph documentation — explicit state-graph routing in code.
- CrewAI documentation — declarative "crew" + "task" assignment.
- Microsoft AutoGen — `GroupChat` manager and explicit speaker-selection policies.
- TaskWeaver (Microsoft) — code-enforced type contracts between sub-agents.
- Cursor `.cursorrules`; GitHub Copilot `.github/copilot-instructions.md` — pure behavioural instructions, no routing.

### Project signals

- `retro(action: "synthesise")` 2026-05-10 — 50 signals, predominantly tool-friction (write-into-worktree pattern); no signal directly about binding governance, which is itself notable: the failures have not yet been reported as retrospective signals.
- `status` 2026-05-10 — 19 active health warnings of the form "bug … reached 'closed' without passing through needs-review", consistent with Finding 7 (no bug pipeline binding).

---

*End of report. Recommendation summary: adopt §6.1 (immediate hardening) and §6.2 (Option D — retire dead bindings, document FastTrack as canonical) as a unit; commit to §6.3 (Option C — tier-aware routing in code, behaviour in YAML) as the structural direction.*
