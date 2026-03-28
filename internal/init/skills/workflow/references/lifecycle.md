# Entity Lifecycle Transitions

Legal state transitions for each entity type. Agents **must not** perform
transitions not listed here. If a transition is needed that does not appear,
surface it to the human rather than guessing.

## Feature

```
proposed → designing → specifying → dev-planning → developing → reviewing → done
```

**Backward transitions (design rework):**
- specifying → designing
- dev-planning → specifying
- developing → dev-planning

**From reviewing:**
- reviewing → done
- reviewing → needs-rework → developing
- reviewing → needs-rework → reviewing

**Terminal states:** superseded, cancelled
- Reachable from any non-terminal state
- Also reachable from done

## Task

```
queued → ready → active → done
```

**Additional transitions:**
- active → blocked → active
- active → needs-review → done
- active → needs-review → needs-rework → active
- active → needs-rework → active

**Terminal states:** done, not-planned, duplicate
- not-planned, duplicate reachable from queued, ready, or active

## Bug

```
reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed
```

**Additional transitions:**
- triaged → cannot-reproduce → triaged (loop back)
- triaged → planned (skip reproduced)
- needs-review → needs-rework → in-progress
- triaged → not-planned
- triaged → duplicate
- reported → duplicate

**Terminal states:** closed, duplicate, not-planned

## Plan

```
proposed → designing → active → reviewing → done
```

**Additional transitions:**
- reviewing → active (rework)

**Terminal states:** superseded, cancelled
- Reachable from any non-terminal state
- Also reachable from done

## Decision

```
proposed → accepted → superseded
proposed → rejected
```

**Terminal states:** rejected, superseded

## Incident

```
reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed
```

**Additional transitions:**
- root-cause-identified → investigating (root cause revised)
- mitigated → investigating (mitigation incomplete)

**Direct to closed from:** reported, triaged, investigating, root-cause-identified, mitigated

**Terminal state:** closed