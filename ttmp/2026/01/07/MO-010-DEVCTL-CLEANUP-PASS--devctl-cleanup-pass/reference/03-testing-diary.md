---
Title: Testing Diary
Ticket: MO-010-DEVCTL-CLEANUP-PASS
Status: active
Topics:
    - backend
    - tui
    - refactor
    - ui-components
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/devctl/cmds/logs.go
      Note: Follow/cancel behavior (fixture follow tested via timeout)
    - Path: devctl/cmd/devctl/cmds/smoketest.go
      Note: Primary protocol v2 smoke test
    - Path: devctl/cmd/devctl/cmds/smoketest_e2e.go
      Note: End-to-end up/status/logs/down smoke test
    - Path: devctl/ttmp/2026/01/06/MO-006-DEVCTL-TUI--create-a-devctl-tui/scripts/setup-fixture-repo-root.sh
      Note: MO-006 fixture generator used for CLI loop
    - Path: devctl/ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/scripts/setup-comprehensive-fixture.sh
      Note: MO-009 comprehensive fixture generator used for CLI loop
    - Path: devctl/ttmp/2026/01/07/MO-010-DEVCTL-CLEANUP-PASS--devctl-cleanup-pass/tasks.md
      Note: Manual test matrix and checkoff list
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-07T16:34:53.885126992-05:00
WhatFor: ""
WhenToUse: ""
---


# Testing Diary

## Goal

Record end-to-end, real-world testing for MO-010 (protocol v2 + dynamic commands + capability enforcement), including exact commands, outputs, failures, and task checkoffs.

## Step 1: CLI Smoketests + Basic Sanity Checks

This step ran the existing CLI smoketest commands (which exercise runtime, supervision, and log following) plus basic command sanity checks. The goal was to quickly validate that protocol v2 did not break core devctl flows and that the test commands still pass.

The main “gotcha” encountered was that `devctl version` is not a valid subcommand (the version is exposed via Cobra’s `--version` flag). After correcting that, the smoketest suite executed cleanly.

**Commit (code):** N/A

### What I did
- Ran help output generation:
  - `cd devctl && go run ./cmd/devctl --help`
- Verified version output:
  - Attempted (failed): `cd devctl && go run ./cmd/devctl version`
  - Correct command: `cd devctl && go run ./cmd/devctl --version`
- Ran smoketests:
  - `cd devctl && go run ./cmd/devctl smoketest`
  - `cd devctl && go run ./cmd/devctl smoketest-failures`
  - `cd devctl && go run ./cmd/devctl smoketest-logs`
  - `cd devctl && go run ./cmd/devctl smoketest-supervise`
  - `cd devctl && go run ./cmd/devctl smoketest-e2e`
- Verified plugins list against the shipped example config:
  - `cd devctl && go run ./cmd/devctl --config .devctl.example.yaml plugins list`
- Checked off tasks:
  - `[13]..[20]` (with task 14 edited to use `--version`)

### Why
- These smoketests cover “real behavior” that unit tests won’t: child process management, state file writing/removal, and log reading under follow/cancel.

### What worked
- All smoketests completed successfully (including log following cancel behavior via the dedicated smoketest).
- `plugins list` shows `protocol_version: "v2"` for the sample plugin.

### What didn't work
- `cd devctl && go run ./cmd/devctl version` failed (there is no `version` subcommand):
  - Output:
    - `Error: unknown command "version" for "devctl"`
    - `Run 'devctl --help' for usage.`
    - `exit status 1`

### What I learned
- “Version” is exposed via Cobra’s built-in flag; tests should use `--version` rather than assuming a command.

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- N/A.

### What should be done in the future
- If we want a `version` subcommand for UX parity, add it explicitly (out of scope for MO-010 tests).

### Code review instructions
- Start with the smoketest implementations:
  - `devctl/cmd/devctl/cmds/smoketest.go`
  - `devctl/cmd/devctl/cmds/smoketest_f​ailures.go`
  - `devctl/cmd/devctl/cmds/smoketest_logs.go`
  - `devctl/cmd/devctl/cmds/smoketest_supervise.go`
  - `devctl/cmd/devctl/cmds/smoketest_e2e.go`

### Technical details
- `plugins list` output excerpt (v2 confirmed):
  - `"protocol_version": "v2"`

## Step 2: MO-006 Fixture (CLI-Only)

This step ran the MO-006 fixture generator and exercised a “typical user loop” using only CLI commands (no TUI). The goal was to validate `plan/up/status/logs/down` with a realistic fixture repo that uses the built-in `e2e` plugin.

The first attempt at automating `logs --follow` used backgrounding + `kill -INT`, but it didn’t terminate reliably under `go run` in this harness and caused the overall command to time out. Switching to `timeout 2s ... logs --follow` made the follow test deterministic and avoided harness timeouts.

**Commit (code):** N/A

### What I did
- Created a fixture repo root:
  - `cd devctl && ./ttmp/2026/01/06/MO-006-DEVCTL-TUI--create-a-devctl-tui/scripts/setup-fixture-repo-root.sh`
- Ran the CLI loop (capturing outputs under `/tmp/`):
  - `devctl plan`
  - `devctl up`
  - `devctl status --tail-lines 25`
  - `devctl logs --service http`
  - `devctl logs --service http --stderr`
  - `timeout 2s devctl logs --service spewer --follow` (expected non-zero exit due to timeout)
  - `devctl down`
  - `devctl status` after down (expected failure: missing state)
  - `devctl up --dry-run` (expected: JSON output and no state persisted)
- Checked off tasks:
  - `[30]..[39]`

### Why
- This fixture closely matches the “devctl as a wrapper around small local binaries + python plugin” real-world usage pattern.

### What worked
- `plan/up/status/logs/down` succeeded end-to-end on the fixture.
- Followed logs contained output within 2 seconds and terminated deterministically under `timeout`.
- After `down`, `status` failed as expected with `read state: ... no such file or directory`.

### What didn't work
- First attempt at follow-cancel automation caused the harness command to time out after 10 minutes:
  - Command (abridged):
    - `... && go run ./cmd/devctl --repo-root "$REPO_ROOT" logs --service spewer --follow ... & pid=$!; sleep 1; kill -INT $pid; wait $pid || true; ...`
  - Result:
    - `command timed out after 600005 milliseconds`

### What I learned
- For “manual follow” commands in an automated harness, prefer `timeout` rather than signal delivery to a backgrounded `go run` process.

### What was tricky to build
- The log-follow UX is inherently interactive; forcing it into a scripted test requires carefully-chosen termination mechanics.

### What warrants a second pair of eyes
- N/A.

### What should be done in the future
- If we want this to be fully automated in CI, add a dedicated CLI test helper or reuse `smoketest-logs` semantics.

### Code review instructions
- Review fixture generator:
  - `devctl/ttmp/2026/01/06/MO-006-DEVCTL-TUI--create-a-devctl-tui/scripts/setup-fixture-repo-root.sh`
- Review log-follow implementation:
  - `devctl/cmd/devctl/cmds/logs.go`

### Technical details
- `status` after down error excerpt:
  - `Error: read state: open .../.devctl/state.json: no such file or directory`

## Step 3: MO-009 Comprehensive Fixture (CLI-Only)

This step ran the comprehensive fixture generator and exercised the same CLI loop, plus the “short-lived service exit info” behavior. The goal was to validate that protocol v2 removed the original startup stall (no `commands.list`) and that a high-entropy environment still works end-to-end.

The fixture came up successfully (regression validated: no wrapper ready-file timeout), and after waiting >30s, `status --tail-lines` showed `short-lived` as dead with an `exit` payload. Notably, the exit info PID differs from the service PID stored in state, which matches the wrapper/child PID indirection documented elsewhere in MO-010.

**Commit (code):** N/A

### What I did
- Created the fixture:
  - `cd devctl && ./ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/scripts/setup-comprehensive-fixture.sh`
- Ran CLI loop (capturing outputs under `/tmp/`):
  - `devctl plan`
  - `devctl up`
  - `devctl status --tail-lines 25`
  - `devctl logs --service <svc>` and `--stderr` for: `backend`, `worker`, `log-producer`, `flaky`, `short-lived`
  - `sleep 35` then `devctl status --tail-lines 25` to validate exit info capture for `short-lived`
  - `devctl down`
- Checked off tasks:
  - `[40]..[44]`

### Why
- This is the closest approximation to “real UI usage” without running the TUI: multiple services, varied health checks, and high log volume.

### What worked
- `devctl up` succeeded on the comprehensive fixture (regression fix validated).
- After 35s, `short-lived` showed `alive: false` with a populated `exit` payload.
- Logs are accessible for all services via `devctl logs`.

### What didn't work
- N/A.

### What I learned
- Exit info PID mismatch is expected in wrapper mode:
  - `state.ServiceRecord.PID` is the wrapper PID (kill handle)
  - `exit.pid` is the child PID written by `__wrap-service`

### What was tricky to build
- Waiting for deterministic exit info requires an explicit sleep; the fixture is intentionally time-based.

### What warrants a second pair of eyes
- The wrapper/child PID semantics (including how they’re surfaced in status output) are subtle; if we want to make status more intuitive, we should make the PID indirection explicit in output.

### What should be done in the future
- Consider adding a `status` output field for wrapper pid vs child pid (out of scope for this test run).

### Code review instructions
- Review fixture generator:
  - `devctl/ttmp/2026/01/06/MO-009-TUI-COMPLETE-FEATURES--complete-tui-features-per-mo-006-design/scripts/setup-comprehensive-fixture.sh`
- Review wrapper PID handling:
  - `devctl/pkg/supervise/supervisor.go`
  - `devctl/cmd/devctl/cmds/wrap_service.go`
  - `devctl/cmd/devctl/cmds/status.go`

### Technical details
- `short-lived` status excerpt (PID mismatch):
  - `pid` (state) != `exit.pid` (child), with `exit_code: 0` and `stderr_tail` lines.

<!-- Provide background context needed to use this reference -->

## Quick Reference

<!-- Provide copy/paste-ready content, API contracts, or quick-look tables -->

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
