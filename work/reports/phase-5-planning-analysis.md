# Phase 5 Planning Analysis

| Document | Phase 5 Planning Analysis                    |
|----------|----------------------------------------------|
| Status   | Active                                        |
| Created  | 2026-05-30                                    |
| Author   | Planning review agent                         |
| Related  | `work/plan/phase-4b-review.md`                |
|          | `work/plan/phase-4-scope.md`                  |
|          | `work/design/workflow-design-basis.md`        |

---

## 1. Executive Summary

This report analyzes the current state of the Kanbanzai project and provides a roadmap to a 1.0 release. The analysis is based on a comprehensive review of design documents, specifications, implementation plans, and the Phase 4b post-implementation review.

**Key findings:**

- **Phase 4b is functionally complete** but has 11 remediation items, including 2 critical bugs that block Phase 5
- **Phase 5 is explicitly defined** in the design documents as a Web/Desktop UI for non-technical stakeholders
- **1.0 release is estimated at 6-10 weeks** from remediation completion, assuming disciplined Phase 5 scope
- **The system is ~80% complete** toward 1.0, with orchestration and self-management capabilities proven

---

## 2. What's Left in the Project

### 2.1 Immediate: Phase 4b Remediation

Phase 4b is functionally complete with all acceptance criteria implemented, but the post-implementation review identified 11 issues requiring remediation before Phase 5 can begin.

#### Critical (blocks Phase 5 gate):

- **R4B-1: Incident YAML field ordering non-deterministic**
  - **Location:** `internal/storage/entity_store.go` — `fieldOrderForEntityType`
  - **Issue:** No case for `"incident"`, causing alphabetical ordering instead of canonical order
  - **Impact:** Violates P1-DEC-008 (deterministic canonical serialization) and blocks AC §16.6
  - **Fix:** Add incident field order case, create canonical fixture, add round-trip test

- **R4B-2: `kbz incident` CLI command group entirely absent**
  - **Location:** `cmd/kanbanzai/main.go`
  - **Issue:** MCP tools are implemented, but no CLI commands for incident operations
  - **Impact:** Blocks AC §11.6 and §14 (CLI interface completeness)
  - **Fix:** Create `incident_cmd.go`, wire into main switch, add usage text and tests

#### Medium (should be resolved):

- **R4B-3:** `output_files` MCP parameter declared as object instead of array
- **R4B-4:** `decomposition.max_tasks_per_feature` config key not implemented
- **R4B-5:** No canonical Incident YAML fixture or round-trip test (depends on R4B-1)
- **R4B-6:** `TestServer_ListTools` does not cover Phase 4b tools

#### Minor (cleanup/polish):

- **R4B-7/R4B-8:** Version strings and help text still say "Phase 3"
- **R4B-9:** Spec typo (documentation fix)
- **R4B-10/R4B-11:** Code quality inconsistencies

**Estimated remediation effort:** 3-5 days for all items

---

## 3. Deferred Tasks and Future Considerations

### 3.1 Deferred Entity Types

From design basis §8.3, the following entity types were recognized but intentionally deferred:

| Entity Type | Status | Notes |
|-------------|--------|-------|
| **Project** | Deferred | Cross-project organization; not needed for single-project workflows |
| **Milestone** | Deferred | Time-based progress tracking; not required for agent-driven workflows |
| **Approval** | Deferred | Formal approval workflow beyond document lifecycle |
| **Release** | Deferred | Release management and versioning; needed when self-hosting matures |
| **Incident** | ✅ Implemented | Phase 4b |
| **RootCauseAnalysis** | ✅ Implemented | Phase 4b (as RCA document type) |
| **KnowledgeEntry** | ✅ Implemented | Phase 2b |
| **TeamMemoryEntry** | Deferred | Different from KnowledgeEntry; not yet clearly defined |

### 3.2 Explicitly Defined Future Capabilities

#### Phase 5: Web/Desktop UI for Designers and Managers

From `work/plan/phase-4-scope.md` §11:

> The kagan comparison identifies a real gap: visibility for non-technical stakeholders. Designers tracking feature progress, managers reviewing sprint health, and stakeholders reading design documents need a read-oriented interface that does not require CLI or MCP access.

**Candidate capabilities for Phase 5 UI:**
- Entity status and lifecycle progress (Epic → Feature → Task dashboard)
- Document access and reading (design docs, specs, dev plans)
- Worktree and branch health visualization
- Estimation and progress metrics
- Knowledge and decision browser scoped to a feature
- Incident status board

**Design constraint:** Phase 4 API surface should be designed with this in mind. Avoid coupling query responses to agent-specific conventions that a human dashboard would not need.

#### Beyond Phase 5 (Phase 6+)

Possible future phases explicitly deferred:

- **GitLab, Bitbucket, or other platform support** — Deferred from Phase 3; GitHub-first is sufficient for 1.0
- **Cross-project knowledge sharing** — Explicitly deferred; single-project scope is sufficient
- **Embedding-based semantic similarity** — Deferred from Phase 3 compaction; Jaccard similarity is working well enough
- **Webhook-based real-time sync** — Not required for agent-driven workflows
- **Automatic context assembly optimization** — Learning which context entries are most useful per task type
- **Confidence score time decay** — For knowledge entries unused over extended periods

### 3.3 Open Questions From Design Basis (§25)

Status of 12 original open questions:

1. ~~Should Feature remain composite, or should Specification become first-class sooner?~~ — ✅ **Resolved:** Document-centric interface model addresses this
2. ~~What final ID allocation strategy should be used?~~ — ✅ **Resolved:** ULIDs (Phase 1)
3. ~~What exact YAML subset or alternative format should be adopted?~~ — ✅ **Resolved:** Block-style YAML with canonical ordering (P1-DEC-008)
4. **How should normalization review be presented to the human?** — Partially addressed through checkpoints; UX refinement remains open
5. **What is the first migration path for an existing project?** — Still open; needed for 1.0 adoption
6. **What correction/undo model should exist above raw Git history?** — Still open; Git history is sufficient for now but may need augmentation
7. ~~How should phase 1 scope be kept from expanding?~~ — ✅ **Resolved:** Phase discipline enforced through decision logs and scope documents
8. ~~When do Incident and RootCauseAnalysis become first-class?~~ — ✅ **Resolved:** Phase 4b
9. **What exact metadata registry format should be used?** — Still open; tags are working, but formal registry may be needed
10. **How should generated projections be stored, cached, or regenerated?** — Handled via SQLite cache, but optimization strategy remains open
11. **At what point should the workflow tool begin managing its own roadmap, bugs, and releases?** — Approaching readiness; Phase 4b validated self-management
12. **What safeguards are needed before the process can safely govern significant changes to itself?** — Partially addressed (merge gates, health checks, human approval for feature transitions); may need additional guardrails

---

## 4. Distance to 1.0 Release

### 4.1 Current State Assessment: **~80% Complete**

**What's complete (Phases 1-4):**

| Phase | Deliverables | Status |
|-------|--------------|--------|
| **Phase 1** | Workflow kernel, entity model, MCP server, validation | ✅ Complete |
| **Phase 2a** | Document intelligence, structural analysis, prefix registry, Plans | ✅ Complete |
| **Phase 2b** | Context profiles, knowledge management, user identity | ✅ Complete |
| **Phase 3** | Git integration, worktrees, GitHub PR, cleanup, merge gates | ✅ Complete |
| **Phase 4a** | Estimation, work queue, dispatch/complete, human checkpoints | ✅ Complete |
| **Phase 4b** | Decomposition, review, conflict analysis, incidents, vertical slicing | ⚠️ Pending R4B-1/R4B-2 |

**What remains for 1.0:**

1. **Phase 4b remediation** — Fix critical bugs blocking gate
2. **Phase 5: Web/Desktop UI** — Visibility for non-technical stakeholders
3. **Production hardening** — Error recovery, edge cases, performance tuning
4. **User documentation** — Getting started guides, tutorials, API reference
5. **Migration tooling** — Import from other systems, safe rollout strategies
6. **1.0 validation workload** — Real-world testing, bug fixing, refinement

### 4.2 Critical Path to 1.0

| Work Item | Estimated Effort | Dependency | Notes |
|-----------|------------------|------------|-------|
| **Fix Phase 4b critical bugs** | 1-2 days | — | R4B-1, R4B-2 must be fixed to open Phase 5 gate |
| **Resolve Phase 4b medium items** | 2-3 days | R4B-1 | R4B-3 through R4B-6 |
| **Phase 5 specification** | 3-5 days | 4b remediation | Detailed acceptance criteria for UI |
| **Phase 5 implementation** | 4-6 weeks | Phase 5 spec | Read-only dashboard, no editing |
| **Production hardening** | 1-2 weeks | Phase 5 impl | Error handling, edge cases, performance |
| **User documentation** | 1 week | Phase 5 impl | Tutorials, API reference, examples |
| **Migration tooling** | 1 week | Phase 5 impl | Import utilities, rollout guides |
| **1.0 validation workload** | 1-2 weeks | All above | Real-world testing, final polish |

**Total timeline:** 6-10 weeks from Phase 4b remediation completion

### 4.3 1.0 Scope Definition

**Must have for 1.0:**
- ✅ All current entity types and workflows (Phases 1-4)
- ✅ MCP server and CLI fully functional
- ✅ Self-management demonstrated through Phase 4b development
- ✅ GitHub integration working
- ⚠️ Basic web UI for visibility (Phase 5) — read-only dashboard
- ⚠️ Solid documentation and migration guides
- ⚠️ Production stability and error handling

**Explicitly not required for 1.0:**
- Additional platform support (GitLab, Bitbucket) — can be added post-1.0
- Cross-project knowledge sharing — single-project scope is sufficient
- Embedding-based search — Jaccard similarity is working well
- Webhook sync — not needed for agent-driven workflows
- Advanced entity types (Project, Milestone, Approval, Release) — can be added as needs emerge
- Editing features in web UI — read-only is sufficient for initial release

### 4.4 Key Assumptions

1. **Phase 4b remediation is straightforward** — No architectural changes required
2. **Phase 5 UI scope is disciplined** — Read-only dashboard, defer editing to later
3. **Self-management is working** — System can develop Phase 5 inside itself
4. **No major architectural pivots** — Foundation is stable
5. **GitHub is sufficient** — Other platforms can be added post-1.0

---

## 5. Recommended Phase 5 Scope

Based on the design documents and gap analysis, Phase 5 should deliver:

### 5.1 Core Capabilities

**Entity visualization:**
- Dashboard showing Epic → Feature → Task hierarchy
- Entity lifecycle status indicators
- Estimation and progress metrics
- Dependency graph visualization

**Document access:**
- Design document browser
- Specification viewer with section navigation
- Dev plan and implementation progress tracking
- Decision and knowledge entry search

**Workflow health:**
- Worktree and branch health indicators
- Merge gate status per feature
- Active task and worker allocation view
- Incident status board with MTTR metrics

**Context and knowledge:**
- Knowledge entry browser filtered by scope and confidence
- Decision record timeline
- Context profile viewer (read-only)

### 5.2 Non-Goals for Phase 5

**Explicitly out of scope:**
- Editing entities through the UI (CLI/MCP remain the write interface)
- User authentication and authorization (defer to deployment environment)
- Real-time updates via websockets (polling is sufficient)
- Mobile-optimized interface (desktop/tablet first)
- Advanced visualizations (Gantt charts, burndown, velocity) — defer to post-1.0
- Integration with external tools beyond GitHub (Slack, JIRA, etc.)

### 5.3 Technical Approach

**Read-only HTTP API:**
- REST endpoints exposing entity queries
- Reuse existing service layer (no new business logic)
- JSON responses suitable for dashboard consumption

**Static web UI:**
- Single-page application (React or similar)
- Served by kanbanzai binary in `serve --ui` mode
- No separate deployment complexity

**Data flow:**
- UI → HTTP API → Service layer → Storage/Cache
- No write path through the UI (agents use MCP; humans use CLI for writes)

---

## 6. Risk Assessment

### 6.1 Phase 5 Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **UI scope creep** | High | High | Strict adherence to read-only scope; defer editing features |
| **Self-management validation gap** | Medium | High | Run Phase 5 development entirely inside the system; surface issues early |
| **UX complexity** | Medium | Medium | Focus on information density, not interactivity; simple layouts |
| **Performance with large projects** | Low | Medium | Cache optimization; pagination; lazy loading |
| **Browser compatibility** | Low | Low | Target modern evergreen browsers only |

### 6.2 1.0 Release Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Adoption complexity** | Medium | High | Excellent getting-started documentation; migration guides; examples |
| **Edge case bugs** | Medium | Medium | Validation workload; real-world testing; rapid bug fix cycle |
| **Performance at scale** | Low | Medium | Benchmarking; cache optimization; incremental indexing |
| **GitHub API rate limits** | Low | Medium | Document rate limit handling; guidance on token management |

---

## 7. Success Criteria for 1.0

A 1.0 release is justified when:

1. **Self-management validated** — Kanbanzai successfully managed its own Phase 5 development
2. **UI functional** — Non-technical stakeholders can track progress without CLI/MCP access
3. **Documentation complete** — Clear getting-started guide, tutorials, and API reference exist
4. **Production-ready** — Error handling, logging, and recovery are robust
5. **Migration path clear** — Users can adopt Kanbanzai on existing projects without starting over
6. **Health checks green** — All validation passes on a real workload
7. **Community-ready** — Open-source release with contribution guidelines

---

## 8. Recommendations

### 8.1 Immediate (Next 1-2 weeks)

1. **Complete Phase 4b remediation** — Fix R4B-1 and R4B-2 (critical), then R4B-3 through R4B-6 (medium)
2. **Validate self-management** — Run a complete feature through dispatch → complete → review cycle
3. **Draft Phase 5 specification** — Detailed acceptance criteria for UI capabilities

### 8.2 Near-term (Next 4-6 weeks)

1. **Implement Phase 5 UI** — Read-only dashboard for entity and document visualization
2. **Develop inside the system** — Use Phase 4 orchestration tools to manage Phase 5 work
3. **Write user documentation** — Getting started guide, tutorials, examples

### 8.3 Pre-release (Final 2-3 weeks)

1. **Production hardening** — Error recovery, edge cases, performance tuning
2. **Migration tooling** — Import utilities, rollout strategies
3. **1.0 validation workload** — Real-world testing on a non-trivial project
4. **Community preparation** — README, CONTRIBUTING.md, LICENSE, release notes

---

## 9. Conclusion

Kanbanzai is well-positioned for a 1.0 release within 6-10 weeks. The core workflow system is complete and proven through self-management of Phase 4b. Phase 5 adds the missing visibility layer for non-technical stakeholders, completing the human-agent collaboration model.

The critical path is clear:
1. Fix Phase 4b critical bugs (1-2 days)
2. Implement Phase 5 UI (4-6 weeks, developed inside the system)
3. Polish, document, and validate (3-4 weeks)

The system's architecture is sound, the scope is disciplined, and the self-management capability has been demonstrated. With focused execution on Phase 5 and production readiness, Kanbanzai can reach 1.0 as a production-ready tool for human-AI collaborative software development.

**Recommendation:** Treat Phase 5 UI as the **1.0 gate feature**. Once functional and validated through self-managed development, declare 1.0 and shift to maintenance/enhancement mode.