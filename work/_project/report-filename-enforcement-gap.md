# Filename Enforcement Gap in Task-Execution Skills

**Date:** 2026-05-10
**Status:** Draft
**Author:** System investigation

## Summary

Agents writing documents in the `researching`, `specifying`, `dev-planning`, and
`reviewing` stages do not know where to place files or how to name them. The
conventions exist in `kanbanzai-documents` but nothing in the agent's mandatory
discovery path points to them.

**Result:** files land in non-canonical directories with non-conforming names
(e.g. `work/investigations/orchestration-abandonment-silent-dod-pass.md` instead of
`work/_project/research-orchestration-abandonment-silent-dod-pass.md`).

## Root cause

Every agent follows this discovery path:

```
AGENTS.md  →  stage-bindings.yaml  →  role + skill  →  skill procedure
```

The document-producing task-execution skills (`.kbz/skills/`) contain detailed
procedures for *content* — structure, methodology, evidence grading — but none of
them include a step for *placement and naming*.

The canonical filename template and folder placement rules live in
`.agents/skills/kanbanzai-documents/SKILL.md` § "Document Types and Locations",
which is **not referenced from any task-execution skill** except `write-design`.

This affects both Copilot and non-Copilot agents equally — `copilot-instructions.md`
lists `kanbanzai-documents` in a table but the "How to use the system" flow doesn't
direct agents to read it before writing files.

## Affected skills

| Skill | Produces | References `kanbanzai-documents`? |
|-------|----------|-----------------------------------|
| `write-design` | design | ✅ Yes ("Next steps after design... See `kanbanzai-documents` skill") |
| `write-spec` | specification | ❌ No |
| `write-dev-plan` | dev-plan | ❌ No |
| `write-research` | research | ❌ No |
| `orchestrate-review` | report | ❌ No |

## Fix

Add a placement/naming step to each affected skill's procedure. The step should
direct the agent to consult `.agents/skills/kanbanzai-documents/SKILL.md` §
"Document Types and Locations" for the correct filename template and folder
placement before writing the file.

Minimal example (for `write-research` Step 4 "Draft the Report"):

> Before writing, determine the correct path and filename. Consult
> `.agents/skills/kanbanzai-documents/SKILL.md` § "Document Types and Locations".
> For project-level research, the path is `work/_project/research-{slug}.md`.
> For batch-scoped research, the path is `work/{BatchID}-{batch-slug}/{BatchID}-research-{slug}.md`.

## Examples of the failure mode

Two files found today in non-canonical locations:

| Non-canonical | Canonical |
|---|---|
| `work/investigations/orchestration-abandonment-silent-dod-pass.md` | `work/_project/research-orchestration-abandonment-silent-dod-pass.md` |
| `work/prompts/p62-revisit-prompt.md` | `work/P62-install-skill-quality-remediation/P62-proposal-revisit.md` |
