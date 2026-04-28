# `decompose_feature` Workflow Friction: Report for Kanbanzai 2.1 Design

| Document | Report                                              |
|----------|-----------------------------------------------------|
| Status   | Draft                                               |
| Date     | 2026-03-27T21:22:31Z                                |
| Related  | `work/design/workflow-retrospective.md`             |
|          | `work/spec/workflow-retrospective.md`               |
|          | `work/spec/kanbanzai-2.0-specification.md` §29      |
|          | `work/design/entity-structure-and-document-pipeline.md` |

---

## 1. Summary

During the decomposition of Plan P5 (Workflow Retrospective) into features and
tasks, the `decompose_feature` tool failed for all three features with the error:

```
decompose_feature failed: feature FEAT-01KMRJ7Z2PVR9 has no linked specification document
```

The tool requires a specification document whose `owner` field matches the feature's
own ID. A plan-level specification document — owned by `P5-workflow-retrospective`
and covering all three features — was invisible to the tool. The decomposition was
performed manually instead.

This report documents what happened, why, and what should change for 2.1.

---

## 2. What Happened, Step by Step

1. **Spec written at plan level.** A single specification document
   (`work/spec/workflow-retrospective.md`, registered as
   `P5-workflow-retrospective/specification-workflow-retrospective`) was written
   covering all three phases of P5. This was the natural structure: the three phases
   share a common signal schema (§5), have interdependent acceptance criteria, and
   tell a coherent story as a unit. Splitting into three separate spec documents
   would have required duplicating §5 three times and would have lost the
   cross-phase narrative.

2. **Features created from the spec.** Three features were created under P5
   (`retro-signal-collection`, `retro-synthesis-tool`,
   `retro-experiment-tracking`), each with their `design` field pointing to the
   plan-level spec document. This is the natural "spec defines the features"
   ordering.

3. **`decompose_feature` called on all three features.** All three calls failed
   immediately. The tool does not follow the feature's `design` field, does not
   look up to the parent plan's spec, and does not accept an explicit spec
   reference. From the tool's perspective, the features had no spec at all.

4. **Manual decomposition performed.** Because the spec was recent and thoroughly
   understood, tasks were created directly from it. The result was accurate, but
   the tool provided no value and the failure produced no actionable guidance.

---

## 3. Root Causes

### 3.1 The tool assumes one spec per feature

`decompose_feature` was designed around the Pattern A workflow:

```
feature created → per-feature spec written → decompose_feature called
```

This assumption is baked into the ownership lookup: it searches for a document
with `owner = feature_id` and type `specification`. There is no fallback.

### 3.2 Plan-level specs are a legitimate and common pattern

Pattern A works well for independent features with distinct concerns. But a
second natural pattern exists:

```
unified design → unified spec → features extracted from spec → decompose using shared spec
```

Pattern B is the better approach for:

- **Phased plans**, where phases build on shared structures defined earlier in the spec.
- **Cross-cutting features**, where a single coherent spec avoids duplication.
- **Plan-first design**, where the specification defines what the features are,
  rather than being written after the features are known.

P5 is a clean example of Pattern B. The spec §5 (signal schema) is foundational
to all three phases. Writing it three times — once per feature — would have
introduced drift risk and reduced clarity.

The tool does not support Pattern B at all.

### 3.3 The `design` field on features is inert

Features have a `design` field intended to reference a governing document. In
this case it was set to the plan-level spec. The field is recorded in the feature
YAML but ignored by `decompose_feature`. This creates a false impression of
linkage: the feature appears to reference a spec, but the tool does not see it.

### 3.4 The stage gate documentation implies Pattern A only

`AGENTS.md` Stage 4 states:

> Gate: Features must exist before writing specification.

This implies features come before specs. For Pattern B, the spec defines and
precedes the features. The documentation offers no guidance for this case, so an
agent following the documentation faithfully would either produce poorly-scoped
pre-feature stubs, or (as happened here) write a plan-level spec and then discover
the tool gap only at decomposition time.

### 3.5 No pre-flight warning

The error is only surfaced when `decompose_feature` is called. Nothing during
feature creation, spec registration, or plan activation warns that the feature
group will lack decomposable specs. An earlier check — or a clear pre-condition
in the tool description — would have allowed the agent to choose a different path
before investing in the features.

---

## 4. How It Was Handled

The decomposition was performed manually using direct knowledge of the spec.
Twelve tasks were created across three features, with a full dependency chain
enforcing phase ordering. The tasks are accurate and complete.

The workaround worked because:

- The spec had just been written and was fresh in context.
- The agent (Claude) had authored the spec and could decompose it without a tool.
- The plan was small enough (3 features, 12 tasks) that manual decomposition
  was tractable.

The workaround would be less reliable if:

- The spec was written by a different agent in a previous session.
- The plan was larger (many features, complex dependencies).
- The decomposing agent did not have the spec in context.

In those cases, the absence of `decompose_feature` support for plan-level specs
would produce either incorrect tasks (from incomplete spec knowledge) or an
unacceptable volume of manual work.

---

## 5. Recommendations for 2.1

### R1: Extend `decompose_feature` with a spec resolution chain

When the tool cannot find a spec with `owner = feature_id`, it should try:

1. The document referenced in the feature's `design` field (if that document is
   of type `specification` or `design`).
2. The specification document owned by the feature's parent plan (if one exists).
3. Fail with a clear error listing all three locations that were checked and found
   empty.

This is a backwards-compatible change. Existing Pattern A users are unaffected.
Pattern B users get the tool working without any workflow changes.

Priority: **high**. This is the minimal fix and covers the common case.

### R2: Add an explicit `spec_id` parameter to `decompose_feature`

Allow the caller to supply a document ID directly, bypassing ownership resolution
entirely:

```
decompose_feature(
  feature_id: "FEAT-01KMRJ7Z2PVR9",
  spec_id: "P5-workflow-retrospective/specification-workflow-retrospective"
)
```

This gives full flexibility: any document can serve as the spec for any feature,
regardless of ownership or plan hierarchy. It also makes the resolution explicit
and auditable.

Priority: **medium**. R1 solves the common case. R2 solves edge cases (e.g., a
spec owned by a sibling feature, or a spec registered under a project-level ID).

### R3: Surface the spec gap earlier

At feature creation time, or when `decompose_feature`'s preconditions are not
met, the system should warn proactively rather than failing silently at call time.

Options:

- `entity(action: create, type: feature)` could warn: "No specification document
  found for this feature. You will need one before calling `decompose_feature`."
- The `status` dashboard could flag features in `proposed` status with no
  resolvable spec as an attention item.
- `doc_gaps` (already available) surfaces this per-feature, but it is not called
  automatically.

Priority: **low**. Useful but not blocking. R1 removes the urgency.

### R4: Update the stage gate documentation for Pattern B

`AGENTS.md` Stage 3 and Stage 4 should acknowledge the plan-level spec pattern:

> For plans where a single specification covers multiple features (phased plans,
> cross-cutting concerns), the specification may be written at the plan level
> before features are created. In this case: (a) register the spec as owned by
> the plan, not any individual feature; (b) create features that reference the
> plan spec via their `design` field; (c) use `decompose_feature` with an
> explicit `spec_id` pointing to the plan spec (see R2).

Priority: **medium**. Prevents future agents from hitting this gap silently.

---

## 6. Impact Assessment

| Severity | Classification | Notes |
|----------|---------------|-------|
| Workflow friction | `tool-gap` | The tool does not support a legitimate and common decomposition pattern |
| Workaround cost | Low (this instance) | Spec was fresh; manual decomposition was accurate |
| Workaround cost | High (general case) | Without fresh spec context, manual decomposition is unreliable at scale |
| Regression risk | None | The workaround produced correct output; no correctness issue |
| Recurrence likelihood | High | Any phased plan or plan-level spec will hit this |

---

## 7. Conclusion

The `decompose_feature` tool has a genuine gap: it only supports the Pattern A
workflow (feature-owned spec), and silently fails for the Pattern B workflow
(plan-owned spec). Pattern B is not a misuse of the system — it is the natural
and preferable structure for any plan that specifies a coherent body of work
across multiple interdependent features.

The fix is straightforward (R1: spec resolution chain, R2: explicit `spec_id`
parameter) and does not affect existing users. The documentation update (R4) is
equally important: without it, agents will continue to write plan-level specs and
discover the tool gap only at decomposition time.

The retrospective system being built under P5 would itself capture signals like
this one. It is fitting that the first friction report was generated manually
during P5's own decomposition.