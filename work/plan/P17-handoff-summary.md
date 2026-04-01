# P17 Handoff Summary: Kanbanzai 3.0 Workflow Engine and MCP Tooling

**Plan:** P17-workflow-and-tooling-3.0
**Design:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` (approved)
**Status:** Active — all 8 features in `developing`, 52 tasks created with dependencies

---

## Plan Structure

| # | Feature | Label | Tasks | Spec | Dev Plan |
|---|---------|-------|-------|------|----------|
| 1 | `FEAT-01KN5-8J24S2XW` mandatory-stage-gates | `stage-gates` | 6 | `work/spec/3.0-mandatory-stage-gates.md` | `work/plan/3.0-mandatory-stage-gates.md` |
| 2 | `FEAT-01KN5-8J257X02` tool-description-audit | `aci-audit` | 8 | `work/spec/3.0-tool-description-audit.md` | `work/plan/3.0-tool-description-audit.md` |
| 3 | `FEAT-01KN5-8J25K4QD` stage-aware-context-assembly | `context-assembly` | 7 | `work/spec/3.0-stage-aware-context-assembly.md` | `work/plan/3.0-stage-aware-context-assembly.md` |
| 4 | `FEAT-01KN5-8J2606B0` review-rework-loop | `review-loop` | 5 | `work/spec/3.0-review-rework-loop.md` | `work/plan/3.0-review-rework-loop.md` |
| 5 | `FEAT-01KN5-8J26CH63` decomposition-quality-validation | `decompose-quality` | 5 | `work/spec/3.0-decomposition-quality-validation.md` | `work/plan/3.0-decomposition-quality-validation.md` |
| 6 | `FEAT-01KN5-8J26RSB6` document-structural-checks | `doc-checks` | 7 | `work/spec/3.0-document-structural-checks.md` | `work/plan/3.0-document-structural-checks.md` |
| 7 | `FEAT-01KN5-8J275BWJ` action-pattern-logging | `observability` | 7 | `work/spec/3.0-action-pattern-logging.md` | `work/plan/3.0-action-pattern-logging.md` |
| 8 | `FEAT-01KN5-8J27H83N` binding-registry-gate-integration | `registry-gates` | 7 | `work/spec/3.0-binding-registry-gate-integration.md` | `work/plan/3.0-binding-registry-gate-integration.md` |

---

## ⚠️ Critical: Coordination with P16 (Skills and Roles 3.0)

Another agent is implementing **P16-skills-and-roles-3.0** concurrently. A conflict analysis was performed and the following constraints apply:

### ✅ Safe to implement now (no P16 overlap)

| Feature | Write Set | Why it's safe |
|---|---|---|
| **1. mandatory-stage-gates** | `internal/validate/`, `internal/model/`, `internal/mcp/entity_tool.go`, `internal/health/` | P16 doesn't touch validation, entity model, or entity tool |
| **4. review-rework-loop** | `internal/model/`, `internal/checkpoint/`, `internal/mcp/status_tool.go`, `internal/validate/` | P16 doesn't touch checkpoint, status tool |
| **5. decomposition-quality-validation** | `internal/mcp/decompose_tool.go` | P16 doesn't touch decompose |
| **6. document-structural-checks** | `internal/docint/`, `internal/mcp/doc_tool.go` | P16 doesn't touch docint or doc tool |
| **7. action-pattern-logging** | new `internal/actionlog/`, MCP server dispatch hook | Entirely new package, no overlap |

These 5 features have **28 tasks** and can proceed immediately.

### ⛔ Must wait for P16 to merge specific features first

| P17 Feature | Blocked By | Risk Level | Reason |
|---|---|---|---|
| **3. stage-aware-context-assembly** | P16's `context-assembly-pipeline` feature | 🔴 HIGH | Both rewrite `internal/context/assemble.go`, `internal/mcp/handoff_tool.go`, and the `assembledContext` struct. Guaranteed merge conflicts on the same functions. P17 must build stage-awareness on top of P16's new pipeline. |
| **8. binding-registry-gate-integration** | P16's `binding-registry` feature | 🟡 DATA DEP | P17 reads the binding registry schema that P16 defines. Not a file conflict, but can't implement without the API existing. |
| **2. tool-description-audit** | ALL MCP tool changes from both plans | 🟠 MEDIUM | Rewrites description strings in every `internal/mcp/*.go` file. Any concurrent work on those files creates merge conflicts. **Run this feature dead last** as a final cosmetic sweep after both plans' MCP layer work is done. |

### P17-internal dependency: Feature 1 before Feature 4

Features 1 (mandatory-stage-gates) and 4 (review-rework-loop) both modify `internal/validate/lifecycle.go` and `internal/model/entities.go`. Feature 1 should merge first since Feature 4's review cycle logic builds on the gate infrastructure.

---

## Recommended Implementation Order

```
Phase 1 (start now, parallel):
  Feature 1: mandatory-stage-gates     (6 tasks)
  Feature 5: decomposition-quality     (5 tasks)
  Feature 6: document-structural-checks (7 tasks)
  Feature 7: action-pattern-logging    (7 tasks)

Phase 2 (after Feature 1 merges):
  Feature 4: review-rework-loop        (5 tasks)

Phase 3 (after P16 context-assembly-pipeline + binding-registry merge):
  Feature 3: stage-aware-context-assembly  (7 tasks)
  Feature 8: binding-registry-gate-integration (7 tasks)

Phase 4 (after ALL MCP tool work from both plans):
  Feature 2: tool-description-audit    (8 tasks)
```

---

## Task Dependency Summary

Within each feature, tasks have explicit `depends_on` relationships. Use `next` to see what's ready — it respects dependencies and auto-promotes tasks when their predecessors complete.

**Root tasks ready now (no dependencies):**

| Feature | Ready Tasks |
|---|---|
| mandatory-stage-gates | `override-record-model`, `gate-prerequisites` |
| tool-description-audit | `test-scenarios`, `rewrite-p1-descriptions`, `error-audit-priority` |
| stage-aware-context-assembly | `stage-config-data`, `finish-summary-limit` |
| review-rework-loop | `entity-review-fields` |
| decomposition-quality-validation | `finding-severity-wiring` |
| document-structural-checks | `section-definitions`, `promotion-state`, `quality-eval-schema` |
| action-pattern-logging | `log-entry-writer`, `eval-suite-framework` |
| binding-registry-gate-integration | `registry-cache` |

**Note:** Even though Features 2, 3, and 8 have ready tasks, they should NOT be started until their P16 blockers are resolved (see above).

---

## Key References

- **Design document:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md`
- **Go style guide:** `refs/go-style.md`
- **Test conventions:** `refs/testing.md`
- **Agent instructions:** `AGENTS.md`
- **Related design (P16):** `work/design/skills-system-redesign-v2.md`
