---
Title: 'Devctl TUI Layout: ASCII Baseline'
Ticket: MO-006-DEVCTL-TUI
Status: active
Topics:
    - backend
    - ui-components
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: moments/ttmp/2026/01/06/MO-006-DEVCTL-TUI--create-a-devctl-tui/sources/local/devctl-tui-layout.md
      Note: Full imported ASCII baseline; this doc contains curated excerpts
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-06T15:33:58.815268981-05:00
WhatFor: ""
WhenToUse: ""
---


# Devctl TUI Layout: ASCII Baseline

## Executive Summary

This document captures the baseline TUI layout as ASCII “screenshots”, imported from an external Markdown file. It exists to make the intended UX concrete and reviewable without needing to run any code.

The canonical imported source is `../sources/local/devctl-tui-layout.md`. This doc excerpts the key baseline screens that the implementation design references.

## Problem Statement

We need a stable, human-readable layout spec for the proposed `devctl` TUI so that:
- reviewers can align on screen structure and keybindings,
- implementation can be staged incrementally while preserving the overall UX,
- the ticket retains the imported baseline even if other design docs evolve.

## Proposed Solution

Adopt the following baseline screens as the initial UX target. Some fields shown (CPU/MEM, health polling, plugin-derived event timelines) are optional enhancements and may appear as “N/A” in early milestones.

### 1) Main Dashboard (running state)

```
┌─ DevCtl - moments ──────────────────────────────────────────────────── [↑/↓/q] ─┐
│                                                                                   │
│ ● System Status: Running                                      Uptime: 2h 34m 12s │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Services (3)                                                    [l] logs [r] restart │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ backend           Running    PID 12847   Healthy   CPU 12%   MEM 245MB   │ │
│ │ ✓ frontend          Running    PID 12849   Healthy   CPU  3%   MEM  89MB   │ │
│ │ ✓ postgres          Running    PID 12843   Healthy   CPU  1%   MEM 156MB   │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Recent Events (5)                                             [f] follow [c] clear │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ 14:23:45  backend    ℹ  Request completed in 234ms                          │ │
│ │ 14:23:42  frontend   ℹ  Asset compilation complete                          │ │
│ │ 14:23:40  backend    ℹ  Connected to database                               │ │
│ │ 14:23:38  postgres   ℹ  Database ready for connections                      │ │
│ │ 14:23:35  system     ✓  All health checks passed                            │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Plugins (3 active)                                                                │
│  • moments-config   (priority: 10)   ops: 2   commands: 0                        │
│  • moments-build    (priority: 15)   ops: 3   commands: 1                        │
│  • moments-launch   (priority: 20)   ops: 4   commands: 2                        │
│                                                                                   │
│ [s] services [p] plugins [e] events [h] help [q] quit                            │
└───────────────────────────────────────────────────────────────────────────────────┘
```

### 2) Service detail view (logs + process info)

```
┌─ Service: backend ────────────────────────────────────────────────── [ESC] back ─┐
│                                                                                   │
│ Status: ✓ Running                                           Started: 2h 34m ago  │
│ Health: Healthy (http://localhost:8083/rpc/v1/health)      Last check: 2s ago    │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Process Info                                                                      │
│  PID:         12847                                                               │
│  Command:     go run ./cmd/moments-server serve                                   │
│  Working Dir: /home/user/moments/backend                                          │
│  CPU:         12.3%                                                               │
│  Memory:      245.8 MB                                                            │
│                                                                                   │
│ Environment                                                                       │
│  PORT=8083  DB_URL=postgresql://localhost:5432/moments  ENV=development           │
│                                                                                   │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Live Logs                                                    [↑/↓] scroll [f] find │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ 14:23:45.234  INFO  Request GET /api/moments?limit=20                       │ │
│ │ 14:23:45.456  INFO  Query completed in 234ms (45 rows)                      │ │
│ │ 14:23:42.123  INFO  WebSocket connection established (client: web-abc123)   │ │
│ │ 14:23:40.891  INFO  Database connection pool initialized (max: 10)          │ │
│ │ 14:23:40.567  INFO  Starting HTTP server on :8083                           │ │
│ │ 14:23:40.234  INFO  Configuration loaded from backend/.env                  │ │
│ │ 14:23:39.901  INFO  Service starting...                                     │ │
│ │                                                                              │ │
│ │                                                                              │ │
│ │                                                                              │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ [r] restart [s] stop [k] kill [d] detach [ESC] back                              │
└───────────────────────────────────────────────────────────────────────────────────┘
```

### 3) Startup sequence (pipeline progress)

```
┌─ DevCtl - moments ──────────────────────────────────────────────────────────────┐
│                                                                                   │
│ ⚙ System Status: Starting                                    Phase: Build [2/5]  │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Pipeline Progress                                                                 │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ Config Mutation     3 plugins applied              0.2s                   │ │
│ │ ▶ Build               Running...                     5.3s                   │ │
│ │   Prepare             Pending                                                │ │
│ │   Validate            Pending                                                │ │
│ │   Launch              Pending                                                │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Build Steps (2/3 complete)                                                        │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ backend-deps        pnpm install                   2.1s                   │ │
│ │ ✓ frontend-deps       pnpm install                   3.2s                   │ │
│ │ ▶ backend-compile     go build ./cmd/moments-server  5.3s (running)         │ │
│ │   frontend-build      pnpm build                          (pending)         │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Live Output                                                                       │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ [backend-compile] Building target: cmd/moments-server                       │ │
│ │ [backend-compile] Compiling 247 packages...                                 │ │
│ │ [backend-compile] ██████████████████░░░░░░░░░░ 65%                          │ │
│ │                                                                              │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Applied Config Patches                                                            │
│  • services.backend.port → 8083                          (moments-config)        │
│  • services.postgres.image → postgres:15-alpine         (moments-config)        │
│  • build.cache_enabled → true                            (moments-build)         │
│                                                                                   │
│ [Ctrl+C] cancel                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

### 4) Error state (validation failure)

```
┌─ DevCtl - moments ──────────────────────────────────────────────────────────────┐
│                                                                                   │
│ ✗ System Status: Failed                                   Phase: Validate [4/5]  │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                                                   │
│ Validation Errors (2)                                                             │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✗ EPORT_IN_USE                                          (moments-config)    │ │
│ │   Port 8083 is already in use                                               │ │
│ │   Fix: Run 'devctl stop' or choose another port in config                   │ │
│ │                                                                              │ │
│ │ ✗ EENVMISSING                                           (moments-build)     │ │
│ │   Required environment variable not set: DATABASE_URL                       │ │
│ │   Fix: Add DATABASE_URL to backend/.env                                     │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Validation Warnings (1)                                                           │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ⚠ WDEPRECATED                                           (moments-build)     │ │
│ │   Node.js version 16 is deprecated, upgrade to 18+                          │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Pipeline Status                                                                   │
│ ┌─────────────────────────────────────────────────────────────────────────────┐ │
│ │ ✓ Config Mutation     3 plugins applied              0.2s                   │ │
│ │ ✓ Build               4 steps completed              8.7s                   │ │
│ │ ✓ Prepare             2 steps completed              1.3s                   │ │
│ │ ✗ Validate            Failed with 2 errors           0.5s                   │ │
│ │ ⊘ Launch              Skipped due to errors                                 │ │
│ └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│ Actions                                                                           │
│  [r] retry   [f] fix manually   [l] view logs   [q] quit                         │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

## Design Decisions

- Keep this doc focused on “layout intent” and keep behavioral/implementation detail in `01-devctl-tui-layout-and-implementation-design.md`.
- Treat CPU/MEM, health polling, and stream-driven events as optional enhancements; the baseline layout includes them as placeholders.

## Alternatives Considered

- N/A (this doc records an imported baseline rather than proposing alternatives).

## Implementation Plan

- This doc does not define the implementation plan; see `01-devctl-tui-layout-and-implementation-design.md`.

## Open Questions

- Which additional baseline screens (plugin list, multi-service event/log stream) should be excerpted here vs left only in the imported source?

## References

- Full imported baseline (includes additional screens): `../sources/local/devctl-tui-layout.md`
- Implementation design: `01-devctl-tui-layout-and-implementation-design.md`
