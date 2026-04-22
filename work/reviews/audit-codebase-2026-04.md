# Quality Audit: Kanbanzai Codebase
Date: 2026-04-22
Scope: ./...
Index state: current (32,413 nodes, 12,371 edges)
Tools run: go vet, go test -race, knowledge graph (staticcheck: not installed — see static_analysis)

Overall: has_critical_findings (resolved during audit)

---

## Dimensions

### test_health: has_findings → clean (fixed during audit)

Summary: 9 tests failing pre-audit across internal/kbzinit; 0 failing post-fix.
Race detector: no races detected. All other packages: passing (cached).

Findings (all resolved):

- [critical] TestRun_NewProject_SkillFrontmatter: embedded skill files for
    `agents`, `getting-started`, and `workflow` used the old YAML-style managed
    marker (`metadata: kanbanzai-managed: "true"`) instead of the comment-style
    markers (`# kanbanzai-managed: true` / `# kanbanzai-version: dev`) expected
    by `transformSkillContent`.
    (location: internal/kbzinit/skills/{agents,getting-started,workflow}/SKILL.md)
    Resolution: replaced YAML metadata blocks with comment-style markers in all
    three source files. Tests now pass.

- [critical] TestRun_Idempotency, TestRun_Idempotency_Skills,
    TestRun_Skills_CurrentVersion_NoOp, TestRun_Skills_OlderVersion_Overwritten,
    TestRun_UpdateSkills_OnlySkills, TestRun_ReviewerRole_CurrentVersion_NoOp,
    TestInit_ManagedMcpJson_CurrentVersion_NoOp: idempotency checks failed because
    `hasLine(existing, "# kanbanzai-managed:")` returned false on installed skill
    files produced from the old-format source. The second `kbz init` run exited
    non-zero with "exists but is not managed by Kanbanzai".
    (location: internal/kbzinit/skills.go:78, root cause same as above)
    Resolution: same frontmatter fix resolved all seven tests.

- [critical] TestP12_Integration_NewProject (AC-C1): `getting-started` skill
    missing `edit_file` reference required by acceptance criterion AC-C1 (MCP
    write-tools rule).
    (location: internal/kbzinit/skills/getting-started/SKILL.md)
    Resolution: added "Direct File Writes Bypassing MCP Tools" anti-pattern
    section referencing `edit_file`.

- [critical] TestP12_Integration_NewProject (AC-C2): `workflow` skill emergency
    brake section missing `.kbz/state/` reference required by acceptance
    criterion AC-C2 (direct-write condition).
    (location: internal/kbzinit/skills/workflow/SKILL.md)
    Resolution: added direct-write bullet to the emergency brake section.

---

### static_analysis: clean_with_notes

Summary: go vet — 0 findings. staticcheck — not installed (gap noted).

Findings:

- [minor] tooling-gap: staticcheck is not installed in this environment.
    staticcheck catches a wider set of correctness issues than go vet (deprecated
    API usage, redundant nil checks, incorrect format strings, unreachable cases).
    (location: environment — no source file)
    Recommendation: install staticcheck (`go install
    honnef.co/go/tools/cmd/staticcheck@latest`) and add to CI. Run
    `staticcheck ./...` as a pre-merge gate.

---

### structural_quality: has_findings

Summary: 341 zero-inbound-degree function candidates returned by graph query.
After verification with trace_call_path and USAGE-edge checks: 7 confirmed dead
code functions; 0 high fan-out SRP violations (2 incidental fan-out cases noted);
1 change-coupling pair in documentation (not source code, not a concern).

Findings:

- [major] dead_code: AllStates — exported function, zero callers confirmed,
    no USAGE edges. Returns all known states for an EntityKind by iterating
    entryStates, allowedTransitions, and terminalStates maps. Non-trivial (38
    lines, complexity 9). Was likely written in anticipation of schema validation
    that was not yet wired up.
    (location: internal/validate/lifecycle.go:413)
    Recommendation: either wire up a caller (e.g. health check or schema
    validator) or delete. If retained for future use, add a TODO comment
    explaining the intended caller.

- [major] dead_code: CheckProfileHealth — exported function, 54 lines, zero
    callers confirmed, no USAGE edges. Checks role-inheritance health (missing
    inherits references, cycle detection) by accepting two function parameters.
    Health system has since grown its own path via internal/health/; this
    function appears to be a superseded phase-2b implementation.
    (location: internal/validate/phase2b_health.go:127)
    Recommendation: verify whether the health system covers profile health via
    another path; if so, delete this function.

- [minor] dead_code: FindGitRoot — exported function, zero callers confirmed,
    no USAGE edges. Traverses the directory tree upward to find .git. The
    internal/kbzinit package uses git shell invocations for its own init logic;
    this function is unused alongside it.
    (location: internal/kbzinit/git.go:12)
    Recommendation: delete, or unexport and wire up if it fills a genuine need.

- [minor] dead_code: HasCommits — exported function in same file as FindGitRoot,
    zero callers confirmed. Checks whether a git repo has any commits by running
    `git rev-parse HEAD`.
    (location: internal/kbzinit/git.go)
    Recommendation: delete alongside FindGitRoot review.

- [minor] dead_code: DefaultCheckOptions — exported zero-argument constructor,
    zero callers confirmed, no USAGE edges. Returns CheckOptions with
    DefaultBranchThresholds and IncludeOK: false.
    (location: internal/health/check.go:38)
    Recommendation: delete if CheckOptions is always constructed inline at call
    sites, or wire up where options are currently constructed manually.

- [minor] dead_code: DeriveGraphProject — exported 9-line utility, zero callers
    confirmed, no USAGE edges. Converts an absolute repo path to a
    hyphen-separated graph project name (mirrors the codebase-memory naming
    convention).
    (location: internal/config/user.go:151)
    Recommendation: verify whether the graph project name is derived elsewhere
    (e.g. hardcoded at call sites); if so, delete and centralise on this
    function, or delete if the derivation is intentionally inline.

- [minor] dead_code: NewCompositeTransitionHook — exported constructor, 3
    lines, zero callers confirmed, no USAGE edges. Wraps multiple
    StatusTransitionHook values into a CompositeTransitionHook. The composite
    pattern was built but never wired into the service layer.
    (location: internal/service/status_transition_hook.go:65)
    Recommendation: delete if single-hook dispatch is sufficient; otherwise wire
    up at the service wiring site.

Notes (not findings):

- docTool (16 outbound calls) and knowledgeTool (12 outbound calls): both are
  large MCP tool handlers that dispatch over action parameters. High outbound
  degree is incidental to the dispatch-table pattern, not an SRP violation.
  No action required.

- Change coupling: entity-structure-and-document-pipeline.md <->
  phase-2-scope.md (coupling_score: 0.80). Both are work/ documentation files,
  not source code. Expected co-evolution. No action required.

---

### style_conformance: clean_with_notes

Summary: 1 minor observation. No acronym-casing violations found in exported
identifiers. No unguarded error discards in production code; all `_ =` usages
are either: (a) explicitly commented as best-effort with a spec reference
(FR-018 in actionlog), (b) cleanup/close operations where the primary error path
is already handled, or (c) test helpers.

Findings:

- [minor] dead-assignment: CheckDependencyCycles assigns `parent` variable then
    immediately discards it with `_ = parent`. Comment reads "parent is used in
    cycle path reconstruction (future enhancement)." This is a suppressor for a
    not-yet-implemented feature, not a logic error.
    (location: internal/health/phase4a.go:69)
    Recommendation: either implement the cycle path reconstruction and use
    `parent`, or remove the variable and its assignment until the feature is
    needed. Carrying suppressed assignments as TODOs degrades signal-to-noise.

---

## Finding Summary

  Critical: 4  (all resolved during audit — see test_health)
  Major:    2
  Minor:    6
  Total:    12

---

## Next Actions

  1. [RESOLVED] Fix 9 test failures in internal/kbzinit (embedded skill
     frontmatter format mismatch + two missing acceptance-criterion content
     requirements). All tests now pass.

  2. [major] Investigate and resolve AllStates dead code
     (internal/validate/lifecycle.go:413) — wire up or delete.

  3. [major] Investigate and resolve CheckProfileHealth dead code
     (internal/validate/phase2b_health.go:127) — verify superseded by health
     system, then delete.

  4. [minor] Install staticcheck and add to CI as a pre-merge gate.

  5. [minor] Review and delete confirmed minor dead code cluster:
     FindGitRoot, HasCommits (internal/kbzinit/git.go),
     DefaultCheckOptions (internal/health/check.go:38),
     DeriveGraphProject (internal/config/user.go:151),
     NewCompositeTransitionHook (internal/service/status_transition_hook.go:65).

  6. [minor] Resolve CheckDependencyCycles dead assignment
     (internal/health/phase4a.go:69) — implement or remove.