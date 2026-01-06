---
Title: 'Devctl TUI: Layout and Implementation Design'
Ticket: MO-006-DEVCTL-TUI
Status: active
Topics:
    - backend
    - ui-components
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/ttmp/2026/01/06/MO-006-DEVCTL-TUI--create-a-devctl-tui/sources/local/01-devctl-tui-layout.md
      Note: Imported ASCII baseline for intended screens
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-06T15:26:28.225316178-05:00
WhatFor: ""
WhenToUse: ""
---




# Devctl TUI: Layout and Implementation Design

## Executive Summary

Add a `devctl` terminal UI (TUI) that provides a live, interactive view of:
1) environment status (running/stopped/errors),
2) supervised services (PID, log paths, health where available),
3) pipeline outcomes (config/build/prepare/validate/launch),
4) plugins and their capabilities,
5) logs with follow/search.

The TUI is designed to ship incrementally: start with a read-only dashboard driven by existing `devctl/pkg/state` + log files, then add actions (up/down/restart), then richer “pipeline + validation” UI, and finally optional enhancements (process CPU/MEM, health polling, plugin stream events).

## Problem Statement

`devctl` currently provides useful primitives (`up`, `down`, `status`, `logs`, `plan`, `plugins list`), but the ergonomics are still “CLI-first”:
- Users must stitch together multiple commands to understand what’s running and why.
- Logs are fragmented per service and require manual tail/follow.
- Pipeline results (especially validation failures) are shown as JSON and are easy to miss/misinterpret in the middle of a dev loop.

A TUI can make the core dev loop (start → inspect → fix → restart) substantially faster by keeping the important state visible and making the common actions one keystroke away.

## Proposed Solution

### Entry point
- Add a new command: `devctl tui`.
- The TUI inherits the same global flags/semantics as the CLI (`--repo-root`, `--config`, `--strict`, `--dry-run`, `--timeout`) and adds UI-specific flags (e.g., `--refresh`, `--tail-lines`).

### Canonical layout (baseline)
The initial layout is derived from the imported ASCII mockups:
- Source (imported, immutable): `../sources/local/01-devctl-tui-layout.md`
- The TUI will implement the same *conceptual* screens and interactions, but specific columns/fields are allowed to be “N/A” in early milestones when the data is not available yet (e.g., CPU/MEM).

### Views
The UI is organized as a small number of screens with a consistent navigation model:

1) **Dashboard**
   - System status (Running/Stopped/Failed) + uptime (from state `CreatedAt`).
   - Services table (from state.json + process liveness):
     - Name, PID, Alive, Cwd, command summary, stdout/stderr log paths.
     - Optional later: health status and CPU/MEM.
   - Recent events pane:
     - TUI-generated events (pipeline phase start/end, errors, restarts, service exit detected).
     - Optional later: plugin-provided stream events via protocol.
   - Plugins summary:
     - Discovered plugins and their priority + capabilities (ops/streams/commands).

2) **Service detail**
   - Process info: name, PID, alive, cwd, command argv, env summary, log file paths.
   - Tabs/panes:
     - stdout log (follow + scrollback)
     - stderr log (follow + scrollback)
   - Actions (scoped to this service where possible; see “Actions”):
     - open logs, restart environment, toggle follow, copy log path.

3) **Pipeline / validation detail**
   - Shows the last run of: config mutation, build steps, prepare steps, validate, launch plan, supervise start.
   - Validation errors/warnings rendered as a table:
     - code, message, plugin/source (if available), suggested fix (if encoded in details or derived).

4) **Help / keybindings overlay**
   - Always accessible; documents global and view-specific keys.

### Navigation & keybindings (initial proposal)
- Global:
  - `q` quit
  - `?` help overlay
  - `tab` cycle major views (Dashboard → Service detail → Pipeline)
  - `/` search/filter (contextual to current list/pane)
- Dashboard:
  - `↑/↓` select service
  - `enter` open service detail
  - `l` open logs (service detail, stdout tab)
  - `p` plugins summary (either a subview or a pane toggle)
  - `e` events pane focus
- Service detail:
  - `esc` back to dashboard
  - `tab` switch stdout/stderr tab
  - `f` toggle follow
- Actions:
  - `u` run “up” pipeline and supervise (start)
  - `d` run “down” (stop)
  - `r` restart (down then up)

### Data sources and “what is reliable”
The TUI should be explicit about what it knows and how it knows it:
- **Running services**: `devctl/pkg/state` (`.devctl/state.json`) + `state.ProcessAlive(pid)`.
- **Logs**: paths stored in state; file-follow via the same logic as `cmds/logs.go` (tail/follow).
- **Plugins**: `devctl/pkg/discovery` + `devctl/pkg/runtime` handshake capabilities.
- **Pipeline results** (when initiated from within the TUI): returned structs from `devctl/pkg/engine.Pipeline` methods.

Some fields in the mockups are not available in the current persisted state (so they are **not MVP requirements**):
- CPU/MEM utilization: not currently tracked; requires process sampling.
- Health endpoint per service: exists in `engine.ServiceSpec.Health` during `up`, but is not stored in state.json.

This design intentionally supports shipping without those fields first; adding them later is treated as an enhancement milestone.

### MVP vs optional fields (dashboard/services table)
- **MVP**: `name`, `pid`, `alive`, `cwd`, `command`, `stdout/stderr log path`
- **Optional**: `health status`, `health endpoint/url`, `cpu%`, `mem`, “recent events” derived from plugin streams

### State staleness policy (MVP)
- If `.devctl/state.json` exists but all PIDs are dead, the UI should surface “stale state” clearly and offer a one-keystroke cleanup path (equivalent to `devctl down` / state removal).

## Design Decisions

### Use an in-process TUI (not shelling out to `devctl` subcommands)
Rationale:
- Reuse existing packages directly (`engine`, `runtime`, `discovery`, `supervise`, `state`).
- Avoid parsing JSON output intended for humans.
- Make it possible to show progress and intermediate results without relying on stdout formatting.

### Treat the imported ASCII mockups as a baseline, not a contract
Rationale:
- Some mock fields are not currently available; forcing parity would require invasive state changes up front.
- The UI should remain responsive and correct, even if “nice-to-have” metrics are absent.

### Event log is first “TUI-native”, later “plugin-native”
Rationale:
- Today there is no end-to-end stream of pipeline events and no persisted event store.
- The runtime protocol has `Event` frames; we can extend plugins/engine later, but the TUI can already provide value by emitting its own events (phase start/end, service exit detection, etc.).

## Alternatives Considered

### Make `devctl` output a richer JSON and render it in a generic TUI viewer
Rejected because:
- It pushes UX into an output format and complicates non-TUI usage.
- It doesn’t naturally support live follow, actions, or interactive navigation.

### Rebuild supervision/pipeline as a long-running daemon with a UI client
Rejected for the first iteration because:
- Substantially larger scope (daemon lifecycle, IPC, persistence, compatibility).
- The existing CLI is synchronous and file-based (state.json + logs), which is sufficient for an MVP TUI.

## Implementation Plan

This plan is milestone-driven so the TUI can ship early and get usage feedback.

### Milestone 0: Skeleton + read-only dashboard
- `devctl tui` command starts a TUI that:
  - loads state.json if present,
  - refreshes periodically,
  - shows services list and whether each PID is alive,
  - can open a service detail view and show log paths.

### Milestone 1: Logs viewer (stdout/stderr) with follow + search
- Embed a log viewport with:
  - scrollback buffer (configurable cap),
  - follow mode,
  - simple substring search and highlight.

### Milestone 2: Actions (up/down/restart) initiated from the TUI
- Add keybindings that call the same underlying packages used by `cmds/up.go` and `cmds/down.go`.
- Provide a “confirm” dialog for destructive actions unless `--yes` is passed.

### Milestone 3: Pipeline results + validation UI
- When running “up” from the TUI:
  - show phase status (config/build/prepare/validate/plan/supervise),
  - render validation errors/warnings in a human-first table view,
  - persist the “last pipeline outcome” in-memory for later inspection (optional persistence can be a later milestone).

### Milestone 4 (optional): Enrich status (health polling, CPU/MEM)
- If we decide it’s worth it:
  - store the launch plan (or health specs) in state.json, or re-compute plan on demand.
  - sample process stats periodically for CPU/MEM display.

### Milestone 5 (optional): Plugin stream integration
- Extend plugins and/or engine to emit `protocol.Event` stream frames during build/prepare/validate so the TUI can render a real event timeline.

## Open Questions

1) Should `devctl tui` be the default (e.g., `devctl` with no args), or remain an explicit subcommand?
2) Do we want to persist any “last pipeline run” result on disk (separate from state.json), or keep it ephemeral?
3) Should restart be environment-wide only (down+up), or do we also want per-service restart (requires new supervisor semantics and/or launch plan persistence)?
4) How strict should the UI be about terminal capabilities (colors, mouse, alternate screen)?

## References

- Imported ASCII layout baseline (source): `../sources/local/01-devctl-tui-layout.md`
- Ticket index: `../index.md`
