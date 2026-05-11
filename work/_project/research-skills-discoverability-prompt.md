# Prompt: Holistic Investigation into Skills/Roles Discoverability and Pipeline Integrity

**Role:** Lead Software Architect (`.kbz/roles/architect.yaml`)
**Skill:** `write-research` (`.kbz/skills/write-research/SKILL.md`)
**Document type:** Research report
**Audience:** Project leadership evaluating whether to continue with instruction-based orchestration or pivot to code-managed pipelines.

---

## Research Question

The Kanbanzai skills/roles system — which maps workflow stages to agent identities and procedures via `stage-bindings.yaml` — has begun failing in ways that suggest systemic fragility rather than isolated bugs. Two concrete failures surfaced simultaneously:

1. `retro_fix` features (created via `retro(action: "create_fix")`) silently fall through to the generic `developing` pipeline — the `implement-retro-fix` skill on disk is **completely unreachable**.
2. `bug_fix` bugs have **no pipeline at all** — zero stage bindings exist for any bug lifecycle status.

Both failures reproduce identically in consumer projects because the relevant files are embedded in the binary and installed verbatim by `kbz init`.

These are not the first skills-system failures. The system has been repaired multiple times. Each fix addressed a symptom. None addressed the root cause.

**The question:** Is this system salvageable through hardening, or have we reached the practical complexity ceiling for instruction-based orchestration — where the gap between "instructions are advice" and "the system must behave deterministically" becomes unbridgeable?

---

## Scope and Methodology

### In scope

1. **Root-cause analysis of current failures.** Trace every layer of the pipeline from entity creation through context assembly to skill loading. Identify exactly where `retro_fix`, `bug_fix`, and any other non-standard tiers lose their routing.

2. **Regression timeline.** Determine when these specific failures were introduced. Were they ever working? If so, what change broke them? Use `git log` archaeology against `stage-bindings.yaml`, `internal/context/pipeline.go`, `internal/binding/model.go`, and `internal/binding/validate.go`.

3. **Brittleness diagnosis.** The system has: a YAML binding file parsed at server start, a `validStages` hardcoded map in Go source, a pipeline that treats feature `status` as the binding key, and tier-awareness fields (`Tier`, `Modes`, `Profile`, `Verifying`) that are parsed but never consumed. Identify every synchronization dependency between these components and enumerate the failure modes when they drift.

4. **Completeness audit.** Audit every entry in `stage-bindings.yaml` against the pipeline's actual consumption. Which bindings are reachable? Which skills on disk are reachable? Which skills on disk are unreachable? Produce a reachability matrix: binding × skill × pipeline path.

5. **Consumer-project impact.** Analyze `internal/kbzinit/` to understand exactly which files are embedded/installed. Determine whether consumer projects can diverge from the embedded bindings (and if so, whether the server handles divergence correctly).

6. **Industry comparison.** Survey:
   - How do other AI-coding orchestration systems handle agent-procedure binding? (Cursor rules, GitHub Copilot instructions, OpenHands/Devin-style agent frameworks, LangChain/LangGraph agent routing)
   - What does Anthropic's published research say about the reliability of instruction-following vs. structured tool use for agent orchestration?
   - What do modern agent-framework best practices say about the "instructions are advice not scripts" boundary?
   - How do systems like TaskWeaver, AutoGen, and CrewAI handle stage-to-agent routing? Are any of them instruction-based, or do they all use code-level routing?

7. **Testing strategy.** Propose concrete, implementable regression tests for the skills/roles system. Consider:
   - **Static analysis:** Can we write a Go test that loads `stage-bindings.yaml`, walks every feature lifecycle status, and asserts a binding exists for each?
   - **Reachability tests:** Can we test that every skill on disk is reachable from at least one binding?
   - **Schema conformance:** Can we test that every binding's `roles` and `skills` fields point to files that exist on disk?
   - **Consumer parity:** Can we test that embedded bindings are byte-identical to the source-of-truth file?
   - **Integration tests:** Can we create mock entities of each tier and verify the pipeline resolves the correct skill?
   - **Canary tests for consumer projects:** What would a `kbz health` check need to surface to detect binding/skill mismatches in consumer installs?

8. **Long-term architectural recommendation.** Given the evidence gathered, make a recommendation with explicit confidence level:
   - **Option A: Harden the instruction system.** Complete the tier-aware routing, add validation at every layer, add comprehensive regression tests. Risk: the system remains instruction-based, and instructions are fundamentally advisory.
   - **Option B: Move to code-managed pipelines.** Replace instruction-based routing with explicit Go code that maps `(status, tier) → pipeline`. Bindings become code, not YAML. Skills remain as guidance documents but selection is deterministic. Risk: loses the configurability that YAML bindings provide.
   - **Option C: Hybrid.** Keep YAML bindings for human-authored stages, move tier-aware routing into code. Bindings declare "what"; code enforces "how to get there."
   - **Option D: Something else** suggested by the industry survey.

### Out of scope

- Fixing the actual bugs (this is an investigation, not implementation)
- Redesigning the skill content itself (this is about the routing/discoverability system, not skill quality)
- Evaluating whether the orchestration model (orchestrator-workers vs. single-agent) is correct for each stage
- Broader Kanbanzai architecture beyond the skills/roles pipeline

### Methodology

1. **Code archaeology** — trace every code path from entity creation through context assembly to skill loading. Read `internal/context/pipeline.go`, `internal/binding/model.go`, `internal/binding/registry.go`, `internal/binding/loader.go`, `internal/binding/validate.go`, `internal/kbzinit/`, and all related test files.
2. **Git archaeology** — use `git log --follow` on `stage-bindings.yaml`, `pipeline.go`, `model.go`, and `validate.go` to identify when each regression was introduced.
3. **Reachability analysis** — for every binding in `stage-bindings.yaml` and every skill in `.kbz/skills/`, determine whether the pipeline can route to it.
4. **Literature review** — search for Anthropic research on instruction-following reliability, agent framework best practices, and comparable orchestration systems.
5. **Synthesis** — produce findings, a trade-off matrix, and recommendations with confidence levels.

---

## Critical Context: What We Already Know

A preliminary investigation has established these facts. They should be verified, not taken as given:

### Fact 1: The pipeline resolves `status → binding` with no tier awareness

```go
// pipeline.go stepResolveStage (step 1)
status, _ := state.Input.FeatureState["status"].(string)
state.Stage = status  // ← "developing", "reviewing", etc.

// pipeline.go stepLookupBinding (step 2)
state.Binding, err = p.Bindings.Lookup(state.Stage)  // ← Lookup("developing")
```

A feature with `tier: retro_fix` and `status: developing` resolves to the standard `developing` binding. The `retro-fixing` binding is never consulted because no feature ever has `retro-fixing` as a lifecycle status.

### Fact 2: The `StageBinding` struct has tier-awareness fields that are never consumed

```go
// model.go
// Profile, Tier, Modes, and Verifying support stages that opt into the
// gated-mode profile schema (e.g. retro-fixing). They are decoded but not
// yet consumed by the pipeline; full schema work is tracked separately.
Profile   *bool                 `yaml:"profile,omitempty"`
Tier      string                `yaml:"tier,omitempty"`
Modes     map[string]*StageMode `yaml:"modes,omitempty"`
Verifying *VerifyingBlock       `yaml:"verifying,omitempty"`
```

No pipeline step reads these fields. The comment says "full schema work is tracked separately."

### Fact 3: `validStages` in `model.go` does not include `retro-fixing`

The hardcoded `validStages` map (used by `ValidateBindingFile`) contains only standard lifecycle stages:
```go
"designing", "specifying", "dev-planning", "developing", "reviewing",
"researching", "documenting", "plan-reviewing"
```

`retro-fixing` is absent. There are also no bug-lifecycle stages (`bug-in-progress`, `bug-needs-review`, etc.).

### Fact 4: The `retro-fixing` binding lacks required fields

The binding in `stage-bindings.yaml` has `profile: true`, `tier: retro_fix`, `modes`, and `verifying` — but no `orchestration`, `roles`, or `skills` field. `ValidateBinding` requires all three. It is unclear whether `ValidateBindingFile` is actually run against the production file or whether a lenient path exists.

### Fact 5: `implement-retro-fix` skill exists but is unreachable

The skill at `.kbz/skills/implement-retro-fix/SKILL.md` is a complete, well-structured 300+ line document. No binding lists it in its `skills` field. No pipeline path reaches it.

### Fact 6: Consumer projects receive identical files

`internal/kbzinit/stage_bindings.go` embeds and installs `stage-bindings.yaml` verbatim. Consumer projects get the same dead `retro-fixing` binding.

---

## Questions to Investigate

These are not exhaustive — the architect should follow the evidence wherever it leads.

### Root cause and regression

1. Was `retro_fix` routing ever functional? If so, at what commit did it break? What was the change?
2. Was `bug_fix` pipeline routing ever designed and implemented? If not, was it intentionally deferred or simply overlooked?
3. When were the `Tier`/`Modes`/`Profile`/`Verifying` fields added to `StageBinding`? What feature/plan were they part of? Why was the routing layer never completed?
4. When was `retro-fixing` added to `stage-bindings.yaml`? Was it added knowing the routing didn't exist, or was the routing expected to work?

### Brittleness analysis

5. How many implicit contracts exist between `stage-bindings.yaml`, `validStages`, feature lifecycle statuses, and the pipeline? List every pair of components that must stay synchronized.
6. When a new lifecycle stage or tier is added, how many places must be updated? Is there a checklist, or is it tribal knowledge?
7. Does the system fail loudly (error at startup) or silently (wrong skill loaded) when these contracts are violated? For each failure mode identified: loud or silent?
8. Are there any automated tests that would catch a binding→skill→pipeline mismatch? If not, what would the minimum viable test look like?

### Completeness

9. Produce a reachability matrix: every binding in `stage-bindings.yaml` × whether the pipeline can route to it × whether its skills exist on disk × whether those skills are actually loadable.
10. Which skills on disk are **not** referenced by any binding? Which bindings reference skills that don't exist on disk?
11. Are there any bindings that would fail `ValidateBinding` if it were run against the production file? (Test this — don't assume.)

### Consumer project impact

12. Can a consumer project modify its `stage-bindings.yaml` to add custom stages? If so, does the server validate custom stages correctly?
13. When the embedded bindings are updated in a new binary version, what happens to consumer projects that have customized their bindings? Is there a merge strategy, a version check, or silent overwrite?

### Industry and research

14. What do Anthropic's published materials say about the reliability of "agent reads instructions and follows them" vs. "agent is routed by code to the correct procedure"? Is there research on instruction-following degradation as complexity increases?
15. How do comparable AI-coding orchestration systems (OpenHands, Devon, Cursor rules, GitHub Copilot extensions) handle the mapping from task type to agent behavior? Code routing, instruction routing, or hybrid?
16. In the broader agent-framework ecosystem (LangGraph, CrewAI, AutoGen, TaskWeaver), is there a consensus on whether agent routing should be code-level or prompt-level?
17. What testing strategies do these systems use to verify correct agent behavior at scale? Are there published approaches for testing LLM-based pipelines?

### Strategy

18. What is the maximum number of stages, tiers, and skills the current architecture can support before the synchronization burden becomes unmanageable? Is there a known complexity threshold from software architecture literature?
19. Given the project's trajectory (growing feature set, growing consumer base), does the evidence support continuing with instruction-based orchestration, or does it support a transition to code-managed pipelines?
20. If a transition is recommended: what is the migration path? Can it be done incrementally, or must it be a cutover?

---

## Output

A research report following the `write-research` skill format.

### Required sections

1. **Research Question** — restated and refined based on investigation findings.
2. **Scope and Methodology** — refined from above, with actual methodology documented.
3. **Findings** — organized by theme. Each finding cites specific code (file:line), git commits, or external sources. Each finding has an evidence grade.
4. **Reachability Matrix** — a table showing every binding, its skills, whether the pipeline can route to it, and whether its skills are loadable.
5. **Trade-Off Analysis** — comparing architectural options (harden vs. code-managed vs. hybrid vs. other) across dimensions: correctness guarantee, maintainability, configurability, migration cost, consumer impact, and complexity ceiling.
6. **Recommendations** — with confidence levels and supporting findings.
7. **Limitations** — what was not investigated, assumptions made, conditions that could change conclusions.

### Evidence grading

Use the `write-research` skill's evidence grading system:
- **Primary source** — source code, git history, official documentation, direct experimentation
- **Secondary source** — blog posts, tutorials, third-party comparisons
- **Confidence** — high / medium / low for each recommendation

### Before writing

- [ ] Call `retro(action: "synthesise")` to surface all signals about pipeline/skills/binding issues
- [ ] Call `knowledge(action: "list")` with relevant topic filters
- [ ] Call `now` for the document date
- [ ] Use `doc(action: "path", type: "research", parent: "<appropriate-entity>")` for the output path

---

## Evaluation Criteria

1. Does the report identify the exact commit(s) where routing regressions were introduced? (required)
2. Does the report include a complete reachability matrix for every binding and skill? (required)
3. Does the report cite specific code (file:line) for every finding about pipeline behaviour? (required)
4. Does the report include an evidence-based trade-off analysis of architectural options? (required)
5. Does the report reference industry practices or published research, not just internal analysis? (high)
6. Does the report include concrete, implementable regression test designs? (high)
7. Does the report distinguish between loud failures (errors at startup) and silent failures (wrong skill loaded)? (high)
8. Does the report analyze consumer-project impact separately from the kanbanzai project itself? (medium)
9. Does the report identify the maximum complexity ceiling for the current architecture? (medium)
