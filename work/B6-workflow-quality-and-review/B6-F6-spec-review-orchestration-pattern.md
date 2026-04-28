# Review Orchestration Pattern Specification

| Document | Review Orchestration Pattern Specification |
|----------|-------------------------------------------|
| Status   | Draft                                     |
| Created  | 2026-03-28T00:27:18Z                      |
| Updated  | 2026-03-28T00:27:18Z                      |
| Related  | `work/design/code-review-workflow.md` §5, §6, §10, §11, §13 |

---

## 1. Purpose

This specification defines the acceptance criteria for Feature F of the P6 Workflow Quality and Review plan: implementing the review orchestration pattern. The feature documents and validates the procedure by which an orchestrating agent coordinates a code review cycle — querying feature metadata, decomposing work into parallel review units, dispatching sub-agents, collating findings, creating remediation tasks, and driving a feature through the `reviewing` → `needs-rework` → `done` lifecycle using existing MCP tools. No new MCP tool is introduced.

---

## 2. Goals

1. Provide a complete, step-by-step orchestrator procedure inside the review SKILL so any agent can drive a review cycle without prior knowledge of this feature.
2. Confirm that every tool in the review tool chain functions correctly in the review context.
3. Demonstrate a complete single-feature review cycle (analysis → findings → remediation → re-review → `done`) against a real feature.
4. Demonstrate multi-feature orchestration: iterating over all features in `reviewing` state and processing them.
5. Establish clear integration points with `human_checkpoint` for ambiguous or high-stakes findings.

---

## 3. Scope

### 3.1 In scope

- Orchestrator procedure documentation added to the review SKILL (extending Feature E's SKILL with the orchestration-level instructions).
- Tool chain verification: each tool listed in §10 of the design is exercised and confirmed to work as required.
- Single-feature end-to-end review cycle run against a real feature in this repository.
- Multi-feature review: iteration over two or more features simultaneously in `reviewing` state.
- Human checkpoint integration for ambiguous findings.
- Remediation task creation (children of the reviewed feature) and re-review of affected sections after remediation.
- Review document created and associated with the reviewed feature, containing structured findings.

### 3.2 Deferred

- A dedicated MCP tool for review orchestration (explicitly deferred per design §10.2).
- Full codebase audit mode (all features reviewed in one pass) — deferred per design §11.
- Automated CI/CD trigger that initiates a review cycle on merge to a protected branch.
- Review history and trend reporting across multiple review cycles.

### 3.3 Explicitly excluded

- Modification of source files by sub-agents during the analysis phase. Sub-agents in the analysis phase are read-only.
- Changes to existing MCP tool interfaces to accommodate review use cases.
- Review of documents other than specification and design documents (e.g., markdown prose-only content outside `work/spec/` and `work/design/`).

---

## 4. Acceptance Criteria

### Orchestration procedure documentation

**AC-F-01** The review SKILL contains an `## Orchestration Procedure` section that covers both roles: the orchestrating agent's coordination steps and the sub-agent reviewer's review steps. The section must be distinct from the sub-agent review instructions already present from Feature E.

**AC-F-02** The orchestration procedure documents the two-phase structure: Analysis Phase (read-only, parallelisable) and Remediation Phase (write operations, sequential or parallelisable with conflict check). Each phase lists its steps in order.

**AC-F-03** The Analysis Phase procedure specifies the following steps in order:
1. Transition the target feature to `reviewing` using `update_status`.
2. Query feature metadata: feature entity, linked spec document via `doc_outline`, task list via `list_entities_filtered(entity_type="task", parent=<feature_id>)`, and review profile via `context_assemble(role="reviewer")`.
3. Decompose into review units: 1 unit for features with ≤10 modified files; 3–8 units for larger features. Each unit specifies source files, relevant spec sections, review dimensions, and the review profile.
4. Dispatch sub-agents in parallel via `spawn_agent`, each receiving their unit's context packet, the review SKILL, and the structured output format. Sub-agents do not modify files.
5. Collate findings: deduplicate overlapping findings across units, categorise each as blocking or non-blocking, determine per-dimension verdict, and compute aggregate verdict.
6. Write findings to a review document associated with the feature.
7. Apply the verdict transition: no blocking findings → `done`; blocking findings → `needs-rework`; ambiguous → invoke `human_checkpoint`.

**AC-F-04** The Remediation Phase procedure specifies the following steps in order:
1. Create remediation tasks as children of the feature using `create_task`, one per blocking finding or logical group of related findings.
2. Run `conflict_domain_check` across the remediation tasks to determine which can be parallelised safely.
3. Dispatch remediation tasks.
4. After all remediation tasks reach a terminal state, re-review only the affected spec sections and source files (not the full feature).
5. Apply verdict transition again; repeat the remediation cycle if new blocking findings are found.

**AC-F-05** The SKILL documents the context budget strategy from design §6: the orchestrator holds metadata only (feature entity, task list, finding summaries) and never loads full source file content into its own context. Source file content is handled exclusively by sub-agents within their review unit context packets.

**AC-F-06** The SKILL documents the human checkpoint integration points from design §13: `human_checkpoint` is invoked when (a) a finding cannot be categorised as clearly blocking or non-blocking, (b) the feature is explicitly marked high-stakes and explicit human approval is required before `done`, or (c) there is disagreement between review dimensions on the aggregate verdict.

### Tool chain verification

**AC-F-07** `list_entities_filtered(entity_type="feature", status="reviewing")` returns the correct set of features in `reviewing` state and is used as the entry point for the multi-feature iteration path.

**AC-F-08** `doc_outline` called on a feature's linked specification document returns a navigable section tree that the orchestrator can use to assign spec sections to review units.

**AC-F-09** `doc_section` called on specific section paths from the outline returns section content that can be included in a sub-agent's context packet.

**AC-F-10** `context_assemble(role="reviewer")` returns a well-formed context packet that sub-agents can use directly, including the reviewer profile and relevant knowledge entries.

**AC-F-11** `spawn_agent` is used to dispatch sub-agents for parallel review units. The SKILL documents what each sub-agent invocation must include: the context packet, the review SKILL reference, the assigned source files, the assigned spec sections, and the structured finding output format.

**AC-F-12** `create_task` is used to create remediation tasks as children of the reviewed feature. Each created task has the feature ID as its `parent_feature`, a slug derived from the finding it addresses, and a summary that references the finding.

**AC-F-13** `update_status` is used to drive the feature through its lifecycle: into `reviewing` at the start of analysis, to `needs-rework` when blocking findings exist, and to `done` when no blocking findings remain.

**AC-F-14** `conflict_domain_check` is called before dispatching remediation tasks when two or more tasks exist, and its output is used to determine whether tasks run in parallel or must be serialised.

**AC-F-15** `human_checkpoint` is invoked and the orchestrator halts further automated transitions until the checkpoint receives a response, as verified by the orchestration procedure documentation and a tested scenario.

### Single-feature review cycle

**AC-F-16** A complete review cycle is run against a real feature in this repository: the feature transitions through `reviewing`, findings are produced, at least one blocking finding results in the feature transitioning to `needs-rework`, at least one remediation task is created as a child of the feature, and after the remediation task completes a re-review is performed.

**AC-F-17** The re-review in AC-F-16 covers only the spec sections and files affected by the remediation task, not the full feature. The SKILL documents this targeted re-review scope.

**AC-F-18** The feature in AC-F-16 ultimately transitions to `done` after all blocking findings are resolved.

**AC-F-19** A review document is created during the cycle in AC-F-16. The document is associated with the reviewed feature (via `doc_record_submit` with the feature ID as `owner`), contains structured findings (each finding includes: location, dimension, blocking status, description, and suggested remediation), and has a summary aggregate verdict.

### Multi-feature review

**AC-F-20** The orchestrator procedure documents and demonstrates iteration over multiple features in `reviewing` state: `list_entities_filtered` returns all such features, and the orchestrator initiates an independent review cycle for each. Since features are independent, their analysis phases may run concurrently.

**AC-F-21** A scenario with two or more features simultaneously in `reviewing` state is run. Both features complete their review cycles. The orchestrator does not conflate findings or state between the two features.

### Human checkpoint integration

**AC-F-22** A scenario is documented or run in which an ambiguous finding causes the orchestrator to invoke `human_checkpoint` rather than making an automated `done` or `needs-rework` transition. The orchestrator's handling of both possible human responses (proceed to `done` vs. create remediation task) is specified in the SKILL.

### Review document

**AC-F-23** The review document produced during a cycle is registered with `doc_record_submit` (type `report`, owner set to the reviewed feature ID) and is discoverable via `doc_record_list(owner=<feature_id>)`.

**AC-F-24** The review document structure is standardised. The SKILL specifies the required sections: summary verdict, per-dimension verdicts, blocking findings (with location, description, suggested remediation), non-blocking findings, and reviewer unit breakdown (which files and spec sections were assigned to each sub-agent).

---

## 5. Verification

### Manual verification steps

1. Read the updated review SKILL and confirm the `## Orchestration Procedure` section is present and covers all steps in AC-F-03 and AC-F-04.
2. Confirm the context budget strategy (AC-F-05) and human checkpoint integration points (AC-F-06) are explicitly documented in the SKILL.
3. Run a single-feature review cycle (AC-F-16 through AC-F-19) and record: feature ID used, tools invoked at each step, findings produced, remediation task IDs, and final `done` transition.
4. Run a two-feature review cycle (AC-F-20 through AC-F-21) and record: feature IDs, that cycles did not conflate state, and that both features reached terminal state.
5. Simulate an ambiguous finding scenario (AC-F-22): confirm `human_checkpoint` is invoked and the orchestrator correctly handles both response branches.
6. After each cycle, confirm the review document is discoverable via `doc_record_list(owner=<feature_id>)` (AC-F-23) and contains the required sections (AC-F-24).

### Tool chain spot-checks

For each tool listed in AC-F-07 through AC-F-15, record the tool call and its result as part of the verification evidence. Evidence may be included inline in the review SKILL as worked examples or in a companion verification note.

### Definition of done for this feature

- All 24 acceptance criteria are met.
- The review SKILL is updated and the orchestration procedure section is present and complete.
- At least one complete single-feature review cycle has been run and the review document is registered.
- At least one two-feature review cycle has been run.
- No new MCP tool has been added (confirmed by checking that `internal/mcp/` has no new tool registrations introduced by this feature).