# Phase 5 Scope and Planning

| Document | Phase 5 Scope and Planning                   |
|----------|----------------------------------------------|
| Status   | Draft                                        |
| Created  | 2026-05-30                                   |
| Related  | `work/plan/phase-4-scope.md`                 |
|          | `work/plan/phase-4b-review.md`               |
|          | `work/reports/phase-5-planning-analysis.md`  |
|          | `work/design/workflow-design-basis.md`       |

---

## 1. Purpose

Phase 5 adds a web-based UI for non-technical stakeholders who need visibility into project progress but do not need to edit entities or interact with the CLI or MCP interface.

The design goal is to provide read-only access to the workflow state through a simple, information-dense dashboard that runs as part of the kanbanzai binary. Editing, creation, and workflow operations remain the domain of agents (via MCP) and humans (via CLI). The UI is a **view layer**, not a new control surface.

This closes the visibility gap identified in the Phase 4 orchestration landscape review (§8): designers tracking feature progress, managers reviewing sprint health, and stakeholders reading design documents need an interface that does not require terminal access or MCP client configuration.

---

## 2. Background: The Visibility Gap

### 2.1 What Exists Today

As of Phase 4b completion, kanbanzai provides two interfaces:

1. **MCP server** — for AI agents to query, create, and manipulate entities
2. **CLI (`kbz`)** — for humans to perform the same operations via terminal commands

Both interfaces are **write-oriented**: they assume the user wants to perform an action, not just observe state. Both require technical comfort: the MCP interface requires configuring an MCP client (Zed, Claude Desktop, etc.); the CLI requires terminal access and familiarity with command syntax.

### 2.2 Who Is Excluded

Non-technical stakeholders who need visibility but not control:

- **Designers** tracking the implementation status of their designs
- **Product managers** reviewing feature progress and worktree health
- **Engineering managers** monitoring task queue depth and estimation accuracy
- **Stakeholders** reading approved design documents and specifications
- **Incident responders** checking incident status and linked RCA documents

These users should not need to learn CLI syntax or install an MCP client. They need a **dashboard**.

### 2.3 Why This Matters for 1.0

A 1.0 release must be usable by teams, not just individuals. Teams include people who do not write code and do not want to interact with a terminal. The visibility gap is not a nice-to-have; it is a blocker for organizational adoption.

Phase 5 is the **1.0 gate feature**.

---

## 3. Phase 5 Goals

1. **Provide read-only visibility** into workflow state for non-technical stakeholders
2. **Run as part of the kanbanzai binary** — no separate deployment, database, or authentication system
3. **Keep scope disciplined** — defer editing, authentication, mobile optimization, and advanced visualizations to post-1.0
4. **Validate self-management** — develop Phase 5 entirely inside the system using Phase 4 orchestration tools
5. **Design for future extension** — structure the HTTP API and UI components to accommodate editing features in later phases

---

## 4. Design Decisions

### P5-DES-001: Read-Only Scope

**Decision:** Phase 5 UI is strictly read-only. All write operations (create, update, transition) remain the domain of MCP tools and CLI commands.

**Rationale:**
- Agents (MCP) and humans (CLI) already have complete write interfaces
- Adding UI-based editing introduces form validation, error handling, concurrent write conflicts, and authentication — all significant scope expansion
- Read-only dashboard delivers the visibility gap closure without these complexities
- Post-1.0, editing can be added incrementally per entity type as needs emerge

**Alternatives Considered:**
- **Full CRUD UI:** Rejected. Too much scope for Phase 5; would delay 1.0 by months.
- **Hybrid (read with limited writes):** Rejected. "Limited" writes are still writes; scope creep risk is high.

**Consequences:**
- Users who want to update an entity must use CLI or MCP
- The UI can be simpler, faster, and more reliable
- Authentication and authorization can be deferred to deployment environment (reverse proxy, VPN, etc.)

---

### P5-DES-002: Embedded HTTP Server

**Decision:** The kanbanzai binary serves the UI via an embedded HTTP server when invoked as `kanbanzai serve --ui` or `kanbanzai serve --ui-only`. The MCP server and HTTP server can run concurrently in the same process or separately depending on invocation mode.

**Rationale:**
- No separate deployment artifact; single binary serves all use cases
- Reuses existing service layer; no new business logic required
- Natural extension of the existing `serve` command
- Simplifies installation and configuration

**Invocation modes:**
- `kanbanzai serve` — MCP server only (stdio transport) — existing behavior
- `kanbanzai serve --ui` — HTTP server only (port 8080 by default, configurable)
- `kanbanzai serve --mcp-and-ui` — both MCP (stdio) and HTTP concurrently (advanced use case)

**Alternatives Considered:**
- **Separate binary:** Rejected. More deployment complexity; no benefit.
- **External static site generator:** Rejected. Dynamic queries require a server; pre-generated static HTML cannot reflect live state.

**Consequences:**
- HTTP server is a new dependency (`net/http` from stdlib; no third-party frameworks required)
- Configuration gains `ui.port` and `ui.bind_address` keys
- Users who do not run the UI pay no runtime cost (server is only instantiated in `--ui` modes)

---

### P5-DES-003: Static SPA with REST API

**Decision:** The UI is a single-page application (SPA) built with a modern frontend framework (React or similar) and served as static assets by the kanbanzai HTTP server. The SPA calls a REST API exposed by the same server to fetch workflow state.

**Rationale:**
- Clear separation between presentation (frontend) and logic (backend)
- Frontend can be developed and tested independently
- Standard tooling (npm, webpack/vite, etc.) for frontend build
- Backend REST API is reusable by other future clients (mobile app, CLI JSON output, third-party integrations)

**API design:**
- Endpoints mirror MCP tool surfaces where possible (e.g., `GET /api/entities/:type/:id` maps to `get_entity` tool)
- JSON responses use the same structures returned by MCP tools
- No new query logic in the HTTP handler layer; delegate to existing service layer

**Alternatives Considered:**
- **Server-side rendered HTML:** Rejected. Less interactive; harder to build modern UX.
- **GraphQL API:** Rejected. Overkill for Phase 5 scope; REST is simpler and sufficient.

**Consequences:**
- Frontend build step required (npm run build)
- Compiled static assets (HTML, JS, CSS) are embedded in the Go binary via `embed` package
- API versioning strategy needed (start with `/api/v1/`)

---

### P5-DES-004: No Authentication in Phase 5

**Decision:** The kanbanzai HTTP server does not implement authentication or authorization. Access control is the responsibility of the deployment environment (reverse proxy, VPN, firewall, etc.).

**Rationale:**
- Authentication is a large scope expansion: user accounts, password hashing, session management, role-based access control
- Different deployment environments have different requirements (SSO, LDAP, OAuth, mTLS, etc.)
- For 1.0, the expected deployment is single-user or small-team on a trusted network
- Deferring authentication allows Phase 5 to focus on the UI and API surface

**Deployment patterns:**
- **Local development:** `kanbanzai serve --ui --bind 127.0.0.1` — only accessible from localhost
- **Trusted network:** Bind to internal IP; rely on network isolation
- **Public network:** Front with nginx/caddy and add HTTP Basic Auth, OAuth, or mTLS

**Alternatives Considered:**
- **Built-in HTTP Basic Auth:** Considered but deferred. Can be added in Phase 5.1 without architectural changes.
- **Token-based auth:** Rejected. Requires user/token management; too much scope.

**Consequences:**
- Users deploying on untrusted networks must configure a reverse proxy
- Documentation must include reverse proxy examples (nginx, caddy)
- Future Phase 5.x can add optional built-in auth without breaking existing deployments

---

### P5-DES-005: Mobile-Responsive but Desktop-First

**Decision:** The UI is designed for desktop and tablet viewports. Mobile phones are supported (responsive layout) but are not the primary design target.

**Rationale:**
- Primary users (designers, managers, stakeholders) work on laptops and desktops
- Dashboard use cases (reviewing task lists, reading documents, checking worktree health) benefit from larger screens
- Mobile optimization requires different interaction patterns (touch targets, navigation, reduced information density)

**Consequences:**
- Layout is responsive (flexbox/grid), so it works on mobile, but may not be optimal
- Advanced mobile optimization can be added post-1.0 if demand emerges

---

## 5. Phase 5 Feature Breakdown

### 5.1 Core Dashboard Views

#### 5.1.1 Entity Hierarchy View

**Capability:** Visual representation of Plan → Feature → Task hierarchy with lifecycle status indicators.

**User Stories:**
- As a product manager, I want to see all features under a plan with their status so I can track progress toward the plan's goals.
- As a designer, I want to see which tasks under my feature are in progress, blocked, or done so I can anticipate when implementation will be complete.
- As a stakeholder, I want to see epic/plan completion percentage so I can report progress to leadership.

**UI Components:**
- Tree view or nested list showing hierarchy
- Color-coded status badges (proposed, designing, ready, active, needs-review, done, etc.)
- Click to expand/collapse levels
- Link to entity detail view

**API Endpoints:**
- `GET /api/v1/plans` — list all plans with summary stats
- `GET /api/v1/plans/:id` — plan detail with linked features
- `GET /api/v1/features/:id` — feature detail with linked tasks
- `GET /api/v1/tasks/:id` — task detail

---

#### 5.1.2 Work Queue View

**Capability:** Display of ready tasks, active tasks, and blocked tasks with dependency information.

**User Stories:**
- As an engineering manager, I want to see how many tasks are ready for dispatch so I can assess whether the team has sufficient backlog.
- As an orchestrating agent operator, I want to see which tasks are active and who is working on them so I can avoid dispatching conflicting work.
- As a developer, I want to see which tasks are blocked and on what so I can prioritize unblocking work.

**UI Components:**
- Three-column layout: Ready | Active | Blocked
- Task cards with estimate, parent feature, dependency count, age
- Click to view task detail
- Optional conflict risk annotation (from `work_queue --conflict-check`)

**API Endpoints:**
- `GET /api/v1/queue` — work queue data (reuses `work_queue` MCP tool logic)
- Query parameters: `?role=<profile_id>`, `?conflict_check=true`

---

#### 5.1.3 Document Browser

**Capability:** Browse and read design documents, specifications, development plans, research notes, and decision records.

**User Stories:**
- As a stakeholder, I want to read the approved design document for a feature so I understand what is being built.
- As a designer, I want to see which documents are in draft status and which are approved so I know what is still under review.
- As a developer, I want to read the specification for the feature I'm implementing so I can understand requirements.

**UI Components:**
- List view: documents grouped by type (design, specification, dev-plan, research, policy)
- Status filter: draft, approved, superseded
- Document viewer with markdown rendering
- Section navigation (TOC sidebar)
- Link to document metadata (owner, approved_by, supersedes, etc.)

**API Endpoints:**
- `GET /api/v1/documents` — list all documents with optional filters (`?type=`, `?status=`, `?owner=`)
- `GET /api/v1/documents/:id` — document metadata
- `GET /api/v1/documents/:id/content` — document content (markdown)
- `GET /api/v1/documents/:id/outline` — structural outline (from doc_outline tool)

---

#### 5.1.4 Worktree and Branch Health

**Capability:** Visualize active worktrees, branch staleness, merge conflicts, and merge gate status.

**User Stories:**
- As an engineering manager, I want to see which features have active worktrees and how stale their branches are so I can identify work that may have fallen behind.
- As a developer, I want to see merge gate status (CI, review, conflicts) for my feature so I know what blocks merging.
- As a product manager, I want to see which features are ready to merge and which are blocked so I can track release readiness.

**UI Components:**
- Table view: entity | worktree path | branch name | staleness | drift from main | merge gate status
- Color-coded health indicators (green = healthy, yellow = stale, red = conflicted)
- Click to view detailed merge gate report
- Link to GitHub PR if present

**API Endpoints:**
- `GET /api/v1/worktrees` — list all worktrees with health data
- `GET /api/v1/worktrees/:entity_id/status` — branch health (reuses `branch_status` tool)
- `GET /api/v1/merge/:entity_id/readiness` — merge gate report (reuses `merge_readiness_check` tool)

---

#### 5.1.5 Knowledge and Decision Browser

**Capability:** Search and browse knowledge entries and decision records, filtered by scope, confidence, topic, and tags.

**User Stories:**
- As a new team member, I want to search knowledge entries for patterns and conventions so I can learn how the project works.
- As a product manager, I want to read decision records related to a feature so I understand the rationale behind design choices.
- As an orchestrating agent operator, I want to see high-confidence knowledge entries so I can manually review what the system has learned.

**UI Components:**
- Search bar with filters: scope, tier, status, tags, min_confidence
- Result cards showing topic, content snippet, confidence, use_count, last_used
- Click to expand full entry with metadata (learned_from, git_anchors, status)
- Decision records shown separately with date, rationale, affects

**API Endpoints:**
- `GET /api/v1/knowledge` — list knowledge entries with filters
- `GET /api/v1/knowledge/:id` — knowledge entry detail
- `GET /api/v1/decisions` — list decision records
- `GET /api/v1/decisions/:id` — decision detail

---

#### 5.1.6 Incident Status Board

**Capability:** Track open incidents, MTTR metrics, linked bugs, and RCA status.

**User Stories:**
- As an incident responder, I want to see all open incidents with severity and time-since-detection so I can prioritize response.
- As an engineering manager, I want to see which incidents have linked RCA documents and which do not so I can ensure proper post-incident review.
- As a stakeholder, I want to see MTTR statistics so I can assess system reliability.

**UI Components:**
- Incident cards grouped by severity (critical, high, medium, low)
- Status indicators: reported, triaged, investigating, mitigated, resolved
- Timestamp display: detected, triaged, mitigated, resolved (for MTTR calculation)
- Linked bugs and RCA document links
- Warning indicator for incidents without RCA past threshold

**API Endpoints:**
- `GET /api/v1/incidents` — list incidents with optional filters (`?status=`, `?severity=`)
- `GET /api/v1/incidents/:id` — incident detail

---

### 5.2 Estimation and Progress Metrics

**Capability:** Display story point estimates, rollup totals, and completion percentages for plans, features, and tasks.

**User Stories:**
- As a product manager, I want to see estimated vs completed story points for a plan so I can track progress toward goals.
- As an engineering manager, I want to see feature-level estimation rollup so I can assess team capacity planning accuracy.

**UI Components:**
- Progress bars showing completed / total story points
- Breakdown by status (done, active, ready, blocked)
- Estimation confidence indicators (if estimates are present)

**API Endpoints:**
- `GET /api/v1/estimates/:entity_id` — estimate query (reuses `estimate_query` tool)

---

### 5.3 Health Check Dashboard

**Capability:** Display health check results including validation errors, stale worktrees, incidents without RCA, and orphaned entities.

**User Stories:**
- As an engineering manager, I want to see health check status at a glance so I know if the project is in a valid state.
- As a developer, I want to see which entities have validation errors so I can fix them.

**UI Components:**
- Traffic light indicator: green (all pass), yellow (warnings), red (errors)
- Expandable list of findings grouped by category
- Click to view affected entity

**API Endpoints:**
- `GET /api/v1/health` — health check report (reuses `health_check` tool)

---

## 6. API Design

### 6.1 REST API Principles

1. **Reuse service layer logic** — do not duplicate query logic; HTTP handlers delegate to existing `EntityService`, `DocumentService`, etc.
2. **Mirror MCP tool surfaces** — where an MCP tool exists, the REST endpoint should return equivalent data
3. **JSON responses** — consistent structure, snake_case keys
4. **Standard HTTP status codes** — 200 OK, 404 Not Found, 500 Internal Server Error, 400 Bad Request
5. **Pagination support** — for large result sets, support `?limit=` and `?offset=` query parameters
6. **Versioned** — all endpoints under `/api/v1/` to allow future breaking changes

### 6.2 Example Endpoint Structure

```
GET /api/v1/plans
GET /api/v1/plans/:id
GET /api/v1/plans/:id/features

GET /api/v1/features/:id
GET /api/v1/features/:id/tasks

GET /api/v1/tasks/:id

GET /api/v1/documents
GET /api/v1/documents/:id
GET /api/v1/documents/:id/content
GET /api/v1/documents/:id/outline

GET /api/v1/worktrees
GET /api/v1/worktrees/:entity_id/status

GET /api/v1/queue
GET /api/v1/health
GET /api/v1/knowledge
GET /api/v1/decisions
GET /api/v1/incidents
GET /api/v1/estimates/:entity_id
```

---

## 7. Frontend Architecture

### 7.1 Technology Choices

- **Framework:** React (or Preact for smaller bundle size)
- **Routing:** React Router
- **State management:** React Context or Zustand (avoid Redux; overkill for read-only)
- **Styling:** Tailwind CSS or plain CSS with CSS modules
- **Markdown rendering:** `react-markdown` or `marked`
- **Build tool:** Vite (fast, modern, good DX)

### 7.2 Component Structure

```
src/
├── App.tsx              — root component, routing setup
├── api/                 — API client functions (fetch wrappers)
├── components/
│   ├── EntityTree.tsx   — hierarchical entity view
│   ├── TaskCard.tsx     — task display component
│   ├── DocumentList.tsx
│   ├── DocumentViewer.tsx
│   ├── WorktreeTable.tsx
│   ├── IncidentBoard.tsx
│   ├── KnowledgeBrowser.tsx
│   └── HealthDashboard.tsx
├── pages/
│   ├── Dashboard.tsx    — landing page (overview)
│   ├── Plans.tsx
│   ├── Features.tsx
│   ├── Queue.tsx
│   ├── Documents.tsx
│   ├── Worktrees.tsx
│   ├── Knowledge.tsx
│   └── Incidents.tsx
└── utils/
    ├── formatters.ts    — date, status badge helpers
    └── constants.ts     — status enums, color mappings
```

### 7.3 Build and Embed

1. Frontend is built as a static bundle: `npm run build` produces `dist/` directory
2. `dist/` contents are embedded in the Go binary via `//go:embed` directive
3. HTTP server serves embedded assets at `/` and API at `/api/v1/`
4. SPA routing: all unmatched routes serve `index.html` (client-side routing)

---

## 8. Configuration

New `.kbz/config.yaml` keys:

```yaml
ui:
  enabled: true                # enable/disable UI server
  port: 8080                   # HTTP port
  bind_address: "127.0.0.1"    # bind to localhost by default
  base_url: ""                 # optional: for reverse proxy setups
```

---

## 9. Dependencies

### 9.1 Internal Dependencies (Prior Phases)

- **Phase 1:** Entity model, storage, validation
- **Phase 2a:** Document records, structural analysis
- **Phase 2b:** Knowledge entries, context profiles
- **Phase 3:** Worktree tracking, merge gates
- **Phase 4a:** Work queue, estimation
- **Phase 4b:** Incidents, decomposition

All dependencies are complete. Phase 5 has no blockers.

### 9.2 External Dependencies

**Backend (Go):**
- `net/http` — stdlib, no third-party HTTP framework needed
- `embed` — stdlib, for static asset embedding
- No new Go dependencies

**Frontend (JavaScript):**
- React or Preact
- React Router
- Tailwind CSS or similar
- Markdown renderer
- Build tool (Vite)

All standard, mature, well-supported libraries.

---

## 10. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Scope creep into editing features** | High | High | Strict adherence to read-only scope; defer editing to Phase 5.1+ |
| **Frontend complexity** | Medium | Medium | Keep UI simple; prioritize information density over interactivity |
| **Performance with large projects** | Low | Medium | Pagination, lazy loading, cache optimization |
| **Self-management validation gap** | Low | High | Use Phase 4 orchestration tools throughout Phase 5 development |
| **API versioning churn** | Low | Low | Start with `/api/v1/`; allow breaking changes post-1.0 as v2 |

---

## 11. Future Considerations (Phase 5.1+)

### Editing Features (Deferred)

Once the read-only UI proves stable, incremental editing features can be added:

- **Entity creation forms** — create plan, feature, task, bug, incident through the UI
- **Status transitions** — click to transition task from ready → active, etc.
- **Document approval** — approve draft documents without CLI
- **Inline editing** — update entity summaries, titles, tags

These can be added per entity type as needs emerge, without requiring a redesign.

### Authentication (Deferred)

Optional built-in authentication for deployments that do not use a reverse proxy:

- **HTTP Basic Auth** — simplest; username/password stored in config
- **OAuth / OIDC** — for SSO integration
- **API tokens** — for programmatic access

### Advanced Visualizations (Deferred)

Richer data views that require more complex frontend logic:

- **Gantt chart** — timeline view of features and tasks with dependencies
- **Burndown chart** — story point burn over time
- **Velocity tracking** — completed story points per week
- **Dependency graph** — interactive graph of task dependencies

### Mobile Optimization (Deferred)

Touch-optimized UI for phones:

- Simplified navigation
- Larger touch targets
- Reduced information density per view

---

## 12. Next Steps

### 12.1 Pre-Implementation

1. **Validate Phase 4b remediation complete** — All R4B items resolved; tests green
2. **Write Phase 5 specification** — Detailed acceptance criteria for each dashboard view
3. **Design API contract** — Finalize REST endpoint structure and response schemas
4. **Choose frontend stack** — React vs Preact; Tailwind vs plain CSS; final decision

### 12.2 Implementation Tracks

Track 1: **Backend API**
- HTTP server setup (`serve --ui` mode)
- REST endpoint handlers (delegate to service layer)
- Static asset embedding
- Configuration (ui.port, ui.bind_address)

Track 2: **Frontend Foundation**
- React app scaffold, routing setup
- API client layer (fetch wrappers)
- Layout and navigation components
- Build and embed workflow

Track 3: **Dashboard Views**
- Entity hierarchy view (plans, features, tasks)
- Work queue view
- Document browser
- Worktree and branch health
- Knowledge and decision browser
- Incident status board

Track 4: **Polish and Testing**
- Responsive layout refinement
- Loading states and error handling
- End-to-end smoke tests
- Documentation (deployment guide, reverse proxy examples)

### 12.3 Validation

1. **Develop Phase 5 inside the system** — Use `decompose_feature`, `dispatch_task`, `complete_task`, and `review_task_output` throughout
2. **Real workload test** — Deploy the UI, use it to monitor Phase 5's own development
3. **Stakeholder feedback** — Have non-technical users (designer, PM, manager) use the UI and report gaps
4. **Gate criteria** — UI functional, self-management validated, documentation complete → declare 1.0

---

## 13. Summary

Phase 5 closes the visibility gap by adding a read-only web UI for non-technical stakeholders. The UI runs as part of the kanbanzai binary, reuses the existing service layer, and provides dashboard views for plans, features, tasks, documents, worktrees, knowledge, and incidents.

The scope is strictly disciplined: no editing, no authentication (defer to reverse proxy), no mobile optimization, no advanced visualizations. These can be added post-1.0 as needs emerge.

Phase 5 is the **1.0 gate feature**. Once the UI is functional and the system has successfully managed its own Phase 5 development using Phase 4 orchestration tools, kanbanzai is ready for public release.

| Phase | Core deliverables | Gate |
|---|---|---|
| 4b (complete) | Decomposition, review, conflict analysis, incidents/RCA | All remediation items resolved |
| 5 (this phase) | Read-only web UI, REST API, dashboard views | UI functional; self-managed development validated |
| 1.0 | Production hardening, documentation, migration tooling | Phase 5 complete; validation workload clean; community-ready |

**Timeline estimate:** 4-6 weeks for Phase 5 implementation + 2-3 weeks for hardening/documentation = **6-10 weeks to 1.0**.