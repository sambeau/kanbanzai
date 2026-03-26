# Entity Lifecycle Transitions

Legal state transitions for each entity type. Agents must not perform
transitions not listed here. If a transition is needed that does not appear,
ask the human.

## Feature

```
proposed → designing → spec-ready → dev-ready → active → done
                                                       → needs-rework → active
Any non-terminal → not-planned
```

## Task

```
queued → ready → active → done
                       → needs-review → done
                       → needs-rework → active
Any non-terminal → not-planned
```

## Bug

```
reported → triaged → active → done
                            → needs-review → done
                            → needs-rework → active
Any non-terminal → not-planned
Any non-terminal → duplicate
```

## Plan

```
proposed → designing → active → done
Any non-terminal → superseded
Any non-terminal → cancelled
```
