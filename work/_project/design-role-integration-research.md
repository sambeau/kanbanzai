| Field  | Value                                                      |
|--------|------------------------------------------------------------|
| Date   | 2026-04-28T00:18:24Z                                       |
| Status | Draft                                                      |
| Author | Sam Phillips (researcher role, Kanbanzai project)          |

## Research Question

How should UX/UI design roles be integrated into the Kanbanzai skill and role system
to support web design, web application UI/UX, and React Native mobile app design
workflows? Specifically:

1. What roles and skills are needed for design work?
2. What do other AI-assisted design setups do?
3. Can effective design outcomes be achieved with roles and skills alone, or is MCP
   tooling required?
4. What is the value of the Sketch MCP server in this integration?

## Scope and Methodology

**In scope:**
- Design roles and skills for the Kanbanzai workflow system
- MCP tooling for design applications (Sketch MCP server, community Figma servers)
- Industry practices for AI-assisted design workflows
- Architectural fit of design roles within the stage-binding system
- Medium-specific design concerns (web marketing, web applications, React Native mobile)

**Out of scope:**
- Implementation of the roles and skills (this is a design-stage decision)
- Non-visual design disciplines (interaction design research, service design)
- Specific component library or design system tooling choices
- Build pipeline integration for design assets

**Methodology:**
- Codebase inspection of the current Kanbanzai role/skill architecture
- Documentation review of the Sketch MCP server and community Figma MCP servers
- Analysis of the Model Context Protocol reference server catalog
- Review of Kanbanzai's existing `document-centric-interface.md` design document for
  prior art on UI/UX integration
- Analysis of LLM capability boundaries for visual design output

## Findings

### Finding 1: Kanbanzai already acknowledges UI/UX work but has no dedicated roles

The document `work/design/document-centric-interface.md` (§§4.4–4.6) mentions that
projects with UI/UX work may include art files, wireframes, logos, and icons as part
of design documents and specifications. However, the current stage-binding system has
no design-specific stage, role, or skill. The `architect` role that owns the `designing`
stage is systems-architecture focused (vocabulary: "coupling analysis," "blast radius
assessment," "inversion of control"), not visual/interaction-design focused.

**Source:** `work/design/document-centric-interface.md` (primary, 2025–2026)
**Confidence:** High — directly observed in codebase structure.

### Finding 2: The current role/skill pattern is well-suited to design disciplines

The role YAML (identity, vocabulary, anti-patterns, tools) and skill SKILL.md
(procedure, output format, evaluation criteria, examples) pattern maps cleanly onto
design work. The key insight from the existing roles is that **vocabulary activates
domain-specific reasoning**. The `implementer-go` role, for example, routes the model
into Go-specific knowledge through terms like "goroutine leak," "table-driven test,"
"functional option pattern." A design role would similarly route the model into design
knowledge through terms like "visual hierarchy," "design tokens," "atomic design,"
"WCAG 2.2," "iOS HIG."

**Source:** Codebase inspection of `.kbz/roles/` (all 18 roles) and `.kbz/skills/`
(all 18 skills); `AGENTS.md` (primary, 2025–2026)
**Confidence:** High — the pattern is consistent across all existing roles and has
proven effective for other disciplines.

### Finding 3: LLMs are capable of design reasoning but limited in design execution

Current LLMs demonstrate strong capability in:
- Information architecture and user flow design
- Accessibility pattern selection and reasoning
- Design system composition and token hierarchy
- Responsive layout strategy and breakpoint reasoning
- Platform-convention-aware pattern selection

LLMs are weaker in:
- Pixel-precise visual layout (spacing, alignment at the sub-component level)
- Native asset creation (icons, illustrations without image generation tools)
- Real-time visual iteration (cannot "see" the output without tooling)
- Platform-specific rendering quirks (CSS browser bugs, React Native layout edge cases)

This gap between reasoning and execution is the same gap that exists in code — Kanbanzai
bridges the code execution gap with worktree + terminal tools. A parallel exists for
design: the role + skill provides the reasoning framework, and MCP tooling (like
Sketch's `run_code`) provides the execution bridge.

**Source:** Model capability analysis based on training data cutoff and observed LLM
behaviour patterns (primary, 2025–2026); design system literature (secondary,
2014–2025)
**Confidence:** Medium — model capabilities evolve rapidly and specific capability
boundaries are not rigorously documented.

### Finding 4: The Sketch MCP server uses a code-execution model uniquely suited to LLM agents

Sketch's MCP server (version 2025.2.4+) exposes two tools:
- `get_selection_as_image` — captures the current Sketch selection as an image for
  visual feedback
- `run_code` — executes arbitrary JavaScript against the full SketchAPI

This two-tool model is significant because it mirrors how Kanbanzai already operates:
agents write code, execute it, check output, and iterate. The Sketch MCP server's
`run_code` tool gives the agent access to the full SketchAPI — it can create artboards,
manipulate layers, define colour variables, set text styles, generate symbols, export
assets, and audit design systems. The AI writes JavaScript → runs it inside Sketch →
sees the result via `get_selection_as_image` → retries on error.

Critically, this is **not** a thin REST wrapper. It is a code execution engine. This
distinction matters because LLMs are better at writing code than orchestrating many
API calls.

The server runs locally on macOS, is off by default, and requires user opt-in. It is
available in the standard Sketch subscription with no additional cost, add-ons, or
token pricing.

Sketch maintains a skills repository with pre-built agent skills, including an
`implement-design` skill for design-to-code conversion.

**Source:** `https://www.sketch.com/ai/` (primary, 2026); `https://www.sketch.com/docs/mcp-server/`
(primary, 2026-04-07)
**Confidence:** High — directly observed from official documentation and testimonials.

### Finding 5: Community Figma MCP servers exist but use a fundamentally different model

Community-built Figma MCP servers (e.g., `figma-mcp-server` by glipsnort on GitHub)
operate via the Figma REST API — they make HTTP calls with file keys and personal
access tokens. This requires the agent to orchestrate multiple API calls (get file
metadata → get nodes → get images → get components → get styles, etc.) to accomplish
tasks that Sketch's `run_code` model can handle in a single script.

The Figma API model introduces friction points:
- File key management (the agent needs to know or discover file keys)
- Token-based authentication
- Rate limiting concerns on the API
- No direct visual feedback loop (requires separate screenshot tooling)
- No official Figma MCP server exists (all are community-maintained)

**Source:** GitHub search for `figma-mcp-server`; MCP registry at `modelcontextprotocol.io`
(primary, 2025–2026)
**Confidence:** Medium — community Figma MCP servers are not exhaustively catalogued
and capabilities may vary.

### Finding 6: No design-specific lifecycle stage exists; integration points are well-defined

The current Kanbanzai workflow stages are: `designing`, `specifying`, `dev-planning`,
`developing`, `reviewing`, `plan-reviewing`, `researching`, `documenting`,
`doc-publishing`. None are design-specific. However, the `designing` stage's existing
structure (single-agent orchestration, document_type: design, required sections)
already accommodates design documents — it simply needs design-specific roles and
skills to populate it.

The natural integration point is to add design roles as alternatives to `architect`
in the `designing` stage binding, with design-specific skills registered alongside
`write-design`. A design review stage could follow the existing `reviewing` pattern
(parallel specialist reviewers).

**Source:** `.kbz/stage-bindings.yaml` (primary, 2025–2026)
**Confidence:** High — directly observed from the binding registry.

### Finding 7: Three distinct design specialisations warrant separate roles

Web design (marketing sites, landing pages), web application UI/UX (data-heavy
interfaces, dashboards, complex interactions), and React Native mobile app design
differ in:

| Dimension | Web Design | Web App UI/UX | React Native Mobile |
|-----------|------------|---------------|---------------------|
| Primary concern | Visual impact, brand, conversion | Information density, task efficiency | Platform conventions, touch ergonomics |
| Key constraints | Browser compatibility, responsive | State complexity, data visualisation | Safe areas, dynamic type, offline |
| Design system | Token-based, atomic design | Compound components, slot patterns | Platform adaptation (iOS/Material) |
| Key references | WCAG 2.2, CSS Grid/Flexbox | WCAG 2.2, ARIA patterns | iOS HIG, Material Design 3 |

These differences justify separate roles with distinct vocabularies, analogous to how
`implementer-go` inherits from `implementer` but adds Go-specific vocabulary. The base
pattern would be a shared `designer` role (inheriting `base`) with specialisations
for web, app, and mobile.

**Source:** Industry design practice; platform design guidelines (iOS HIG, Material
Design 3, WCAG 2.2) (secondary, 2023–2025)
**Confidence:** Medium — the exact boundaries between specialisations are a design
decision, not a research finding. The three-way split is a proposal grounded in
observable platform differences.

## Trade-Off Analysis

### Approach comparison: how to integrate design into Kanbanzai

| Criterion | Roles + skills only | Roles + skills + MCP tooling |
|-----------|---------------------|------------------------------|
| Agent can reason about design | Yes — vocabulary + anti-patterns activate domain knowledge | Yes — same vocabulary activation |
| Agent can produce design documents | Yes — follow output format from skill | Yes — plus can include screenshots and extracted tokens |
| Agent can generate design files | No — cannot create/read Sketch documents | Yes — `run_code` writes to Sketch, `get_selection_as_image` reads |
| Agent can audit design consistency | Partial — can reason about code, not design files | Yes — can walk the actual symbol/component tree |
| Agent can export assets for dev | No — must describe what to export | Yes — can export SVGs, PNGs, tokens programmatically |
| Agent can close the design iteration loop | No — human must implement in design tool | Yes — can write code, run it, verify visually, fix |
| Setup complexity | Low — just YAML and markdown files | Medium — requires Sketch, MCP config, SketchAPI knowledge |
| Platform dependency | None — text-based | macOS only (Sketch requirement) |
| Long-term portability | High — skills are tool-agnostic | Lower — tied to Sketch as the design tool |

### MCP server comparison: Sketch vs Figma

| Criterion | Sketch MCP (official) | Figma MCP (community) |
|-----------|----------------------|----------------------|
| Execution model | Code execution (JavaScript) | REST API calls |
| Agent-friendliness | High — single `run_code` tool | Moderate — multiple endpoint calls |
| Visual feedback | Built-in `get_selection_as_image` | Requires separate tooling |
| Official support | Yes — maintained by Sketch BV | No — community-maintained |
| Authentication | None (local only) | Personal access token |
| Cost | Included in Sketch subscription | Free (plus Figma account) |
| API coverage | Full SketchAPI (anything a plugin can do) | Limited to wrapped endpoints |
| Platform | macOS only | Cross-platform (API-based) |
| Iteration speed | Fast — code → execute → image → fix in milliseconds | Slower — HTTP round-trips per endpoint |

## Recommendations

### Recommendation 1: Create three design roles with a shared base

**Recommendation:** Create a `designer` base role (inheriting `base`) with shared
design vocabulary, then `designer-web`, `designer-app`, and `designer-mobile` roles
that inherit from `designer` and add medium-specific vocabulary and anti-patterns.

**Confidence:** High
**Based on:** Finding 2 (role pattern fits design), Finding 7 (three specialisations)
**Conditions:** Applies when the project has UI/UX work. For pure backend or CLI
projects, the design roles would remain unused (equivalent to how `implementer-go` is
unused in Python projects).

### Recommendation 2: Create a `design-ui` skill for the designing stage

**Recommendation:** Create a `design-ui` skill that follows the `write-design`
pattern but for visual/interaction design. The skill should encode the design process
(information architecture → wireframe → component design → design token definition →
specification), design-specific anti-patterns, and output format for design documents.
The skill should be registered under the `designing` stage alongside `write-design`.

**Confidence:** High
**Based on:** Finding 2 (skill pattern fits), Finding 6 (integration points are clear)
**Conditions:** The skill must be medium-constraint (not high-constraint like
`write-design`) to allow flexibility for the three different design contexts.

### Recommendation 3: Integrate Sketch MCP awareness into the skill from the start

**Recommendation:** The `design-ui` skill procedure should include optional Sketch MCP
tool steps for teams that use Sketch. This includes: using `get_selection_as_image`
for visual feedback during design iteration, using `run_code` for creating and
modifying design files, and using `run_code` for exporting assets and tokens. The
skill should reference the SketchAPI documentation and common scripting patterns.

**Confidence:** Medium
**Based on:** Finding 4 (Sketch MCP's code-execution model), Finding 3 (LLM execution
gap)
**Conditions:** This adds complexity to the skill. If the team does not use Sketch,
these steps are dead weight. Consider making Sketch integration an optional appendix
to the skill rather than a core procedure step. This should be evaluated during the
design stage.

### Recommendation 4: Defer platform-specific skills to a later iteration

**Recommendation:** Start with a single `design-ui` skill rather than three separate
skills (`design-ui-web`, `design-ui-app`, `design-ui-mobile`). The role vocabulary
differences will route the agent into the right domain knowledge; a single skill with
appropriate IF/THEN branches per design medium is simpler than three skills. Split
into separate skills only if the single skill exceeds the 500-line budget.

**Confidence:** Medium
**Based on:** Finding 7 (three specialisations), `write-skill` conventions (500-line
budget)
**Conditions:** The skill must clearly distinguish between medium-specific procedures.
If the web design procedure and the mobile design procedure differ substantially,
separate skills will be warranted. This should be evaluated in the design stage.

### Recommendation 5: Do not build custom MCP tooling for design at this stage

**Recommendation:** Rely on Sketch's official MCP server for design execution rather
than building custom MCP tools. The Sketch MCP server is well-designed for agentic
workflows, officially maintained, and free. Building a custom MCP server for a
different design tool would replicate the same code-execution pattern at significant
development cost.

**Confidence:** Medium
**Based on:** Finding 5 (Figma MCP gap), Finding 4 (Sketch's execution model)
**Conditions:** This recommendation is conditional on the team using Sketch. If the
team uses Figma or another tool, the recommendation would shift to using available
community MCP servers or building a lightweight bridge.

## Limitations

- **Model capability boundaries are not precisely measured.** Finding 3 is based on
  observed behaviour, not formal evaluation. The specific limits of LLM visual design
  reasoning may differ from the analysis here.
- **Sketch API coverage was not exhaustively tested.** Finding 4 is based on
  documentation, not hands-on experimentation with the MCP server. Real-world
  limitations may surface during implementation.
- **No other design tool MCP servers were evaluated in depth.** Only Sketch's official
  server and the existence of community Figma servers were considered. Penpot,
  Adobe XD, and other tools may have MCP integrations not covered here.
- **The three-way role split is a design hypothesis, not a research finding.**
  Whether `designer-web`, `designer-app`, and `designer-mobile` are the right
  specialisations should be evaluated in the design stage.
- **This research does not address design review roles.** A full design workflow would
  likely need design-specific reviewers (accessibility reviewer, visual consistency
  reviewer, platform-convention reviewer) mirroring the existing `reviewer-conformance`,
  `reviewer-security` pattern. This was excluded from scope.
- **The research assumes Sketch as the primary design tool.** If the team transitions
  to another design tool, the Sketch-specific recommendations will need re-evaluation.
