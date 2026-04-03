# Design: Remove Epic Entity

- Status: approved
- Date: 2026-06-10
- Feature: FEAT-01KN85HMQP2CX

---

## Overview

The `Epic` entity type was a Phase 1 concept superseded by `Plan` in Phase 2.
All code paths that reference it are already marked deprecated. There are no
live epic state files in `.kbz/state/` — the kanbanzai project itself has fully
migrated to Plan. This design authorises removal of all Epic-related code.

## Goals and Non-Goals

**Goals:** Remove all Epic code, types, tests, and testdata from the codebase.

**Non-Goals:** No data migration; no behaviour changes to any other entity type.

## Design

Remove all code, types, tests, and testdata related to Epic. This covers the
model struct, status constants, service methods, CLI handlers, MCP tool
descriptions, validation rules, lifecycle graph entries, cache logic, storage
field ordering, ID allocation, and the document-intelligence entity-ref pattern.
The legacy `epic` parent field on `Feature` (Phase 1 compat fallback) is also
removed.

No data migration is required.

## Alternatives Considered

Keep the deprecated code indefinitely. Rejected: dead code adds noise and
maintenance burden with zero benefit given no live state exists.

## Dependencies

None.