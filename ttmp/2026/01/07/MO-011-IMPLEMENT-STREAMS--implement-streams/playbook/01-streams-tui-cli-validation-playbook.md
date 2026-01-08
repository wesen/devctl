---
Title: 'Streams: TUI + CLI validation playbook'
Ticket: MO-011-IMPLEMENT-STREAMS
Status: active
Topics:
    - streams
    - tui
    - plugins
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/devctl/cmds/stream.go
      Note: devctl stream start CLI used in playbook
    - Path: devctl/pkg/tui/models/streams_model.go
      Note: Streams view steps reference this UI
    - Path: devctl/pkg/tui/stream_runner.go
      Note: UIStreamRunner used by the playbook
    - Path: devctl/testdata/plugins/long-running-plugin/plugin.py
      Note: logs.follow long-running fixture plugin
    - Path: devctl/testdata/plugins/telemetry/plugin.py
      Note: telemetry.stream fixture plugin
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-07T20:37:17.082300366-05:00
WhatFor: Provide a repeatable, copy/paste procedure to validate devctl stream plumbing end-to-end (TUI UIStreamRunner + Streams view + devctl stream CLI) using only repo-local fixture plugins.
WhenToUse: After changes to stream protocol/runtime, UIStreamRunner, Streams view, or devctl stream CLI; when debugging stream hangs, missing events, or cleanup issues.
---


# Streams: TUI + CLI validation playbook

## Purpose

Validate protocol streams end-to-end in a realistic way:
- start a stream from the TUI (Streams view → JSON prompt),
- observe stream events delivered via Watermill → transformer → Bubble Tea,
- stop a long-running stream and confirm cleanup,
- and verify equivalent behavior via `devctl stream start`.

## Environment Assumptions

- You are in the `devctl` git repo.
- `python3` is available.
- You can run `go` commands for this repo.
- No network access required.

This playbook intentionally uses the fixture plugins under `devctl/testdata/plugins/`:
- `telemetry` (`telemetry.stream` emits a finite stream and ends)
- `long-running-plugin` (`logs.follow` emits ticks until stdin closes; ideal for stop/cleanup tests)

## Commands

```bash
set -euo pipefail

# 1) Locate devctl repo root
DEVCTL_ROOT="$(pwd)"

# 2) Create a temporary "repo root" that only contains .devctl.yaml.
#    This simulates a real project repo using devctl, while referencing fixture plugins by absolute path.
REPO_ROOT="$(mktemp -d)"
cat >"$REPO_ROOT/.devctl.yaml" <<YAML
plugins:
  - id: telemetry
    path: python3
    args:
      - "$DEVCTL_ROOT/testdata/plugins/telemetry/plugin.py"
    priority: 10

  - id: follow
    path: python3
    args:
      - "$DEVCTL_ROOT/testdata/plugins/long-running-plugin/plugin.py"
    priority: 20
YAML

# 3) Sanity-check: verify plugins list and that telemetry stream is supported
go run ./cmd/devctl --repo-root "$REPO_ROOT" plugins list
```

## Exit Criteria

### A) TUI validation

1) Run the TUI against the temporary repo root:

```bash
go run ./cmd/devctl --repo-root "$REPO_ROOT" tui
```

2) Navigate to the Streams view:
- press `tab` until you reach `streams` (dashboard → events → pipeline → plugins → streams)

3) Start a finite telemetry stream:
- press `n`
- paste:

```json
{"op":"telemetry.stream","plugin_id":"telemetry","input":{"count":5,"interval_ms":50}}
```

- press `enter`

Expected:
- Stream appears in the Streams list with status `running`, then transitions to `ended`.
- Events viewport shows 5 `metric` events (counter values 0..4) and then an `end`.

4) Start a long-running stream and stop it:
- press `n`
- paste:

```json
{"op":"logs.follow","plugin_id":"follow","input":{}}
```

- press `enter`, observe `tick N` logs arriving
- press `x` to stop the selected stream

Expected:
- Stream transitions to `error` with an `[end]`/ended status driven by stop (implementation currently marks stop as not-ok).
- Events stop arriving after `x`.
- No UI freeze/hang.

### B) CLI validation

1) Start a finite telemetry stream (human output):

```bash
go run ./cmd/devctl --repo-root "$REPO_ROOT" stream start \
  --plugin telemetry \
  --op telemetry.stream \
  --input-json '{"count":3,"interval_ms":10}'
```

Expected:
- Prints a header `plugin=telemetry op=telemetry.stream stream_id=...`
- Prints three `[metric] ...` lines and then an `[end ok=true]` line.

2) Start a long-running follow stream (raw JSON) and interrupt it:

```bash
go run ./cmd/devctl --repo-root "$REPO_ROOT" stream start \
  --plugin follow \
  --op logs.follow \
  --json
```

Expected:
- First line is a small JSON header with `plugin_id`, `op`, `stream_id`.
- Subsequent lines are JSON-encoded `protocol.Event` objects.
- Ctrl+C terminates the command and does not leave a stuck `python3` plugin process.

## Notes

- If a stream start hangs: confirm the plugin declares the op in `capabilities.ops` (streams-only declarations are intentionally rejected for invocation).
- TUI Streams view selection uses `j/k`, while the event viewport scroll uses `↑/↓`.
