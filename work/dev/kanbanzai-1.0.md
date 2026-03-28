# Kanbanzai 1.0 Delivery Plan

| Document | Kanbanzai 1.0 Delivery Plan        |
|----------|------------------------------------|
| Status   | Draft                              |
| Created  | 2026-03-26T14:41:37Z                         |
| Updated  | 2026-03-26T14:41:37Z                         |
| Related  | `work/design/kanbanzai-1.0.md`     |

---

## 1. Purpose

This document records the implementation sequencing for `P3-kanbanzai-1.0`. It identifies dependencies between features, defines delivery waves, and explains the rationale for ordering decisions. Feature-level dev-plans (task decompositions) are separate documents owned by each feature.

---

## 2. Features

| Feature ID         | Slug                     | Summary                                              | Status      |
|--------------------|--------------------------|------------------------------------------------------|-------------|
| FEAT-01KMKRQSD1TKK | `skills-content`         | Author the six kanbanzai skill files                 | specifying  |
| FEAT-01KMKRQRRX3CC | `init-command`           | `kanbanzai init` command                             | specifying  |
| FEAT-01KMKRQV025FA | `public-schema-interface`| Export Go types and query layer; publish JSON Schema | specifying  |
| FEAT-01KMKRQT9QCPR | `binary-distribution`    | GoReleaser pipeline and GitHub Releases              | specifying  |
| FEAT-01KMKRQWF0FCH | `hardening`              | Robustness, error messages, MCP annotations          | specifying  |
| FEAT-01KMKRQVKBPRX | `user-documentation`     | Populate `docs/` for 1.0                             | specifying  |

---

## 3. Dependencies

### 3.1 Hard dependencies

| Feature | Depends on | Reason |
|---|---|---|
| `init-command` | `skills-content` | `init` embeds the skill files and writes them to the target project. The content must be authored before the command can be implemented. |
| `hardening` (init edge cases) | `init-command` | Edge case handling for `init` (non-git dir, conflicting skills, partial state, no write permission) requires the command to exist first. |
| `binary-distribution` | `init-command` | The binary being packaged and released should include the `init` command. Setting up the release pipeline before `init` is implemented means releasing an incomplete binary. |
| `user-documentation` | all others | Documentation must reflect the implemented, working system. Writing docs before implementation risks inaccuracy and churn. |

### 3.2 Independent features

| Feature | Rationale |
|---|---|
| `skills-content` | Pure content authoring — no dependency on any other feature. |
| `public-schema-interface` | Refactoring and exposure of existing internal types. The schema is already stable; this work can proceed at any time. |
| Hardening (`doc_record_refresh`, MCP annotations) | These two items in the hardening spec are independent of `init-command` and can be implemented at any time. |

---

## 4. Delivery Waves

Features are grouped into waves. Within a wave, work can proceed in parallel. A wave does not begin until all blocking dependencies from the previous wave are met.

### Wave 1 — Foundation (parallel)

| Feature | Notes |
|---|---|
| `skills-content` | No blockers. Author all six skill files. |
| `public-schema-interface` | No blockers. Export types, query layer, and JSON Schema generation. |
| Hardening: `doc_record_refresh` + MCP annotations | Independent sub-items from the hardening feature. Can be completed as part of Wave 1. |

### Wave 2 — Core Command

| Feature | Notes |
|---|---|
| `init-command` | Requires `skills-content` complete. Implements the full `kanbanzai init` command including skill installation and config generation. |

### Wave 3 — Robustness (parallel)

| Feature | Notes |
|---|---|
| Hardening: init edge cases + partial state | Requires `init-command` complete. Covers non-git dir, conflicting skills, no write permission, and interrupted init. |
| `binary-distribution` | Requires `init-command` complete. Wire up GoReleaser and GitHub Actions pipeline. Can proceed in parallel with hardening. |

### Wave 4 — Documentation

| Feature | Notes |
|---|---|
| `user-documentation` | Requires all other features complete. Populate `docs/` with Getting Started, Workflow Overview, Schema Reference, MCP Tool Reference, and Configuration Reference. |

---

## 5. Sequencing Diagram

```
skills-content      ──────────────────────┐
public-schema-interface ──────────────────┤ (Wave 1)
hardening (refresh + annotations) ────────┘
                                          │
                                          ▼
                                    init-command (Wave 2)
                                          │
                              ┌───────────┴───────────┐
                              ▼                       ▼
                    hardening (init)        binary-distribution
                         (Wave 3)               (Wave 3)
                              └───────────┬───────────┘
                                          │
                                          ▼
                                 user-documentation
                                      (Wave 4)
```

---

## 6. Notes

**`hardening` spans waves.** The hardening feature contains work that is independent of `init-command` (the `doc_record_refresh` MCP tool and MCP safety annotations) and work that depends on it (init edge cases and partial state recovery). These can be treated as two separate tasks when the feature is decomposed, allowing the independent work to proceed in Wave 1 without blocking on Wave 2.

**`binary-distribution` is pipeline work.** The GoReleaser configuration and GitHub Actions workflow can be drafted and tested with a stub binary in Wave 1 or 2, then finalised once `init-command` is complete. The formal dependency is on having a production-ready binary to release, not on having the pipeline files committed.

**`user-documentation` is a gate for 1.0.** The plan is not done until documentation is complete. A binary with no docs is not 1.0 by the definition in `work/design/kanbanzai-1.0.md` §1.