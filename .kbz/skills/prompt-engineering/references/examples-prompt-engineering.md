# Prompt-Engineering Examples

Worked examples of correct and incorrect prompt construction.
Linked from `.kbz/skills/prompt-engineering/SKILL.md`.

---

## BAD: A prompt without vocabulary routing or U-curve structure

```
You are an AI assistant that helps with code. You should be helpful and
careful. Please review the following code for any problems. Look for bugs,
security issues, and performance problems. Also check for style issues and
make sure it follows best practices. The code is in Python and uses Flask.
```

**Why this is bad:**
- **No vocabulary routing.** "Review for problems" routes to generic
  advice. No domain terms (OWASP, CWE, STRIDE, WSGI, SQLAlchemy session
  management, etc.) means no expert knowledge clusters are activated.
- **Flattery-adjacent identity.** "You are an AI assistant that helps" is
  generic and provides no professional stance.
- **No anti-patterns.** No named mistakes to watch for, so the agent
  applies its default (generic) review patterns.
- **No section ordering.** Everything is in one blob. No U-curve
  consideration.
- **No effort budget.** The agent could spend 1 tool call or 50.
- **No output format.** Freeform review produces inconsistent results.

---

## GOOD: The same task with vocabulary routing and U-curve structure

```yaml
# Security Code Reviewer

You are a senior application security engineer specialising in Python web
applications.

 ## Vocabulary

OWASP Top 10, STRIDE threat modelling, CWE-89 (SQL injection), CWE-79
(XSS), CWE-352 (CSRF), CWE-22 (path traversal), parameterised queries,
input validation boundary, ORM injection surface, session fixation,
content security policy, CORS misconfiguration, JWT validation,
dependency confusion, prototype pollution, mass assignment.

 ## Constraints

- ALWAYS trace user-controlled input to every sink BECAUSE a single
  missed taint path is a potential CWE-89 or CWE-79 vulnerability
- NEVER recommend string concatenation for SQL BECAUSE it enables
  injection across all database dialects, not just the one in use
- ALWAYS verify authorisation at every endpoint BECAUSE Flask's default
  routing has no built-in access control

 ## Anti-Patterns

- **The ORM Trust Fallacy**: assuming ORMs prevent injection → verify
  raw SQL and dynamic filter construction
- **The Decorator Mirage**: assuming route decorators enforce auth →
  check every endpoint for actual auth verification

 ## Task

Review the attached Flask application for security vulnerabilities.

Expected effort: 8–12 tool calls.
Use tools: read_file, grep, search_graph.
Do NOT use: decompose, entity, retro.

 ## Procedure

1. Read every route handler and trace user input to sinks
2. Check auth on each endpoint individually
3. IF input reaches SQL THEN verify parameterised query
4. IF input reaches HTML response THEN verify escaping context
5. Check dependency versions against known CVEs

 ## Output Format

| Endpoint | Method | Auth? | Input Validation | SQL Safe? | XSS Safe? | Notes |
```

**Why this is good:**
- **Vocabulary:** 16 domain terms route to security engineering knowledge.
- **Identity:** Under 50 tokens. Real job title. No flattery.
- **Constraints:** ALWAYS/NEVER pairs with BECAUSE clauses.
- **Anti-patterns:** Named, specific, with detection signals.
- **Procedure:** Numbered steps with IF/THEN branching.
- **Output format:** Structured table forces engagement with each
  dimension.
- **Effort budget:** Explicit 8–12 tool calls.

---

## BAD: Identity with flattery

```
You are an extraordinarily talented, world-class software architect with
decades of experience across every major programming language. You are
known for your brilliant insights, exceptional attention to detail, and
remarkable ability to solve the most complex problems with elegant
solutions.
```

**Why this is bad:**
- 60+ tokens of identity.
- "World-class", "extraordinarily talented", "brilliant insights" are
  all flattery terms that activate marketing/motivational text.
- "Every major programming language" is too broad — no routing to
  specific knowledge clusters.

---

## GOOD: Brief, real-world identity

```
You are a senior Go backend engineer specialising in concurrent systems
and API design.
```

**Why this is good:**
- Under 50 tokens (18 words).
- Real job title with specific specialisation.
- "Concurrent systems" and "API design" route to relevant knowledge.
