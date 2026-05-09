# Write-Research Examples

Worked examples of correct and incorrect research report patterns.
Linked from `.kbz/skills/write-research/SKILL.md`.

---

## BAD: Unsupported Recommendation

> ## Research Question
> Which testing framework should we use?
>
> ## Findings
> After looking at several options, testify seems like the most popular
> choice in the Go ecosystem. It has good assertion helpers and suite support.
>
> ## Recommendations
> Use testify for all tests.

**WHY BAD:** No methodology stated. Single finding cites no sources and uses "seems like" instead of evidence. No alternatives compared. No trade-off analysis. Recommendation has no confidence level and no conditions. No limitations section. A decision-maker cannot evaluate whether this recommendation is well-founded.

---

## BAD: Scope Creep with Ungraded Evidence

> ## Research Question
> How should we implement caching for the entity API?
>
> ## Scope and Methodology
> Reviewed caching strategies for Go applications.
>
> ## Findings
> Redis is fast. A blog post from 2019 says Redis handles 100k ops/sec.
> Memcached is also fast. Someone on Reddit said they prefer Memcached.
> We should also consider CDN caching for static assets and maybe
> implement a service mesh while we're at it.

**WHY BAD:** Scope creep (CDN caching and service mesh are unrelated to entity API caching). Evidence is ungraded — a 2019 blog post and a Reddit comment are treated as authoritative. No distinction between primary and secondary sources. Findings drift from the stated research question.

---

## GOOD: Structured Research with Graded Evidence

> ## Research Question
> What approach should the skills system use for validation scripts:
> POSIX shell or Go executables? This informs the implementation plan
> for the skills system redesign.
>
> ## Scope and Methodology
> **In scope:** Runtime characteristics, portability, maintainability, and
> implementation cost of POSIX shell vs. Go for validation scripts.
> **Out of scope:** Alternative languages (Python, Node). Script content
> and logic.
> **Methodology:** Code analysis of existing scripts in the repository,
> official POSIX and Go documentation review, measurement of startup
> times on the development environment.
>
> ## Findings
>
> ### Finding 1: Startup Time
> POSIX shell scripts start in ~5ms. Go binaries compiled with
> `go build` start in ~15-30ms. For validation scripts that run at
> stage gates, both are well within the 5-second budget.
>
> Source: Direct measurement on macOS 14 (primary, current).
>
> ### Finding 2: Portability
> POSIX shell is available on all target environments (macOS, Linux)
> without additional tooling. Go requires the Go toolchain for
> compilation, but the project already depends on it.
>
> Source: POSIX.1-2017 specification §2 (primary, current);
> project dependency analysis of go.mod (primary, current).
>
> ### Finding 3: Maintainability
> The existing codebase is 100% Go. Shell scripts introduce a second
> language that contributors must know. However, validation scripts
> are simple (grep for headings) and unlikely to grow complex.
>
> Source: Repository language analysis via `tokei` (primary, current).
>
> ## Trade-Off Analysis
>
> | Criterion       | POSIX Shell        | Go Executable       |
> |-----------------|--------------------|---------------------|
> | Startup time    | ~5ms               | ~15-30ms            |
> | Portability     | Universal          | Requires Go toolchain |
> | Maintainability | Second language    | Consistent with codebase |
> | Complexity fit  | Good for simple checks | Over-engineered for grep |
> | Test coverage   | Harder to unit test | Standard Go testing |
>
> ## Recommendations
>
> **Recommendation:** Use POSIX shell for validation scripts.
> **Confidence:** High.
> **Based on:** Findings 1-3. Validation scripts perform simple
> structural checks (grep for headings). Shell is the natural fit for
> this complexity level. The maintainability cost is low because the
> scripts are small and unlikely to grow.
> **Conditions:** This recommendation assumes validation scripts remain
> simple structural checks. If validation logic grows to require parsing
> or complex conditionals, revisit in favour of Go.
>
> ## Limitations
>
> - Did not evaluate Python or Node as alternatives (out of scope)
> - Startup measurements taken on a single machine; CI environments may differ
> - Maintainability assessment is subjective and based on current team composition

**WHY GOOD:** Clear research question tied to a specific decision. Scope and methodology stated upfront. Three findings with cited primary sources and recency noted. Trade-off matrix compares alternatives across specific dimensions. Recommendation includes confidence level, evidence basis, and conditions. Limitations acknowledge boundaries honestly. A decision-maker can evaluate each claim independently.
