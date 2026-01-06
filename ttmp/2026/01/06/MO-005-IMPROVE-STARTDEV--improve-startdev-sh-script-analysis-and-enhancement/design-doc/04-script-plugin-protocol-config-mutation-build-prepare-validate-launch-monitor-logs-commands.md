---
Title: 'Script Plugin Protocol: Config Mutation, Build/Prepare, Validate, Launch/Monitor, Logs, Commands'
Ticket: MO-005-IMPROVE-STARTDEV
Status: active
Topics:
    - dev-tooling
    - scripts
    - bash
    - dev-environment
DocType: design-doc
Intent: long-term
Owners:
    - team
RelatedFiles:
    - Path: ttmp/2026/01/06/MO-005-IMPROVE-STARTDEV--improve-startdev-sh-script-analysis-and-enhancement/design-doc/02-updated-architecture-moments-dev-go-stdio-phase-plugins.md
      Note: Prior protocol context
    - Path: ttmp/2026/01/06/MO-005-IMPROVE-STARTDEV--improve-startdev-sh-script-analysis-and-enhancement/design-doc/03-moments-as-plugins-repo-specific-phases-moments-server-dev-interface.md
      Note: Repo-specific plugin contract example
ExternalSources: []
Summary: A script-first stdio plugin protocol that makes it easy to extend a dev orchestrator with config mutation, build/prepare, validate, launch/monitor, log tailing/streaming, and command registration (git-xxx style).
LastUpdated: 2026-01-06T13:55:36.041211882-05:00
WhatFor: Define an ergonomic, robust contract so teammates can write small scripts that plug into a generic dev tool without learning Go or internal types.
WhenToUse: When designing/extending the generic dev tool plugin system or onboarding colleagues to write plugins.
---


# Script Plugin Protocol: Config Mutation, Build/Prepare, Validate, Launch/Monitor, Logs, Commands

## Executive Summary

We want it to be *trivially easy* for scripts to provide:

- **Config mutation**: augment/manipulate a config struct (shape can be “anything”)
- **Build / prepare**: run steps using config and return success/error
- **Validate**: return structured errors/warnings or success
- **Launch + monitor**: either return a launch plan *or* run as a long-running controller
- **Logs**: tail/follow logs or emit long-running stdout-like output
- **Commands**: register additional commands (like `git-xxx`) that show up under the orchestrator CLI

This document specifies a **script-first stdio protocol** that makes these extensions ergonomic:

- **NDJSON** on stdout (one JSON object per line, no multi-line framing complexity)
- **stderr-only for human logs** (stdout is reserved for protocol frames)
- **handshake + capabilities** for discovery
- **request/response** for normal ops and **event streams** for long-running log/monitor output
- **config patches** (set/unset by dotted paths) instead of “return the whole config” so merges are deterministic

## Problem Statement

Without a stable plugin contract, dev tooling tends to rot into one of two bad outcomes:

1. A monolithic script nobody wants to touch.
2. A pile of ad-hoc scripts that “kind of work”, but can’t be composed deterministically and can’t be supervised safely.

We need a contract that:

- is easy enough for a teammate to implement in bash/python without a framework,
- composes multiple scripts predictably (deterministic merge rules),
- supports long-running output for monitoring and logs without corrupting the protocol,
- gives the orchestrator enough structure to provide good UX (status, errors, follow logs, etc.).

## Proposed Solution

### The mental model

The orchestrator is like `git`, and plugins are like `git-foo` binaries:

- the orchestrator has core commands
- plugins can register additional commands and implement lifecycle hooks

Plugins are just executables. The orchestrator communicates with them over stdin/stdout using NDJSON.

### Diagram: request/response + streaming

```
┌──────────────────────────┐
│        devctl core        │
│--------------------------│
│ - discover plugins        │
│ - merge config patches    │
│ - run build/prepare       │
│ - supervise services      │
│ - tail/follow logs        │
│ - expose commands         │
└───────────┬──────────────┘
            │ NDJSON (stdout protocol)
            │ stderr = human logs
    ┌───────▼─────────┐
    │ plugin executable │
    │------------------│
    │ handshake         │
    │ request handlers  │
    │ optional streams  │
    └──────────────────┘
```

### Transport and framing: NDJSON v1

**Hard rules**:
- Plugin **stdout** MUST contain only JSON frames (one per line).
- Plugin **stderr** is for human logs (free-form).
- The orchestrator treats any non-JSON stdout as a protocol error.

This choice makes bash/python plugins easy to write (no length-prefix parsing).

### Core envelope (messages)

#### 1) Handshake (plugin → orchestrator, first frame)

```json
{"type":"handshake","protocol_version":"v1","plugin_name":"example","capabilities":{"ops":["config.mutate","validate.run"],"streams":["logs.follow"],"commands":["db-reset"]},"declares":{"side_effects":"process","idempotent":false}}
```

#### 2) Request (orchestrator → plugin)

```json
{"type":"request","request_id":"req-1","op":"config.mutate","ctx":{"repo_root":"/abs/path","cwd":"/abs/path","deadline_ms":30000,"dry_run":false},"input":{"config":{},"patches":[]}}
```

#### 3) Response (plugin → orchestrator)

```json
{"type":"response","request_id":"req-1","ok":true,"output":{},"warnings":[],"notes":[{"level":"info","message":"did thing"}]}
```

#### 4) Event (plugin → orchestrator; long-running streams)

```json
{"type":"event","stream_id":"stream-1","event":"log","level":"info","message":"line...","fields":{"source":"backend"}}
```

#### 5) Stream end

```json
{"type":"event","stream_id":"stream-1","event":"end","ok":true}
```

### Core ops: what scripts can implement

Plugins advertise which ops they support via the handshake.

#### Config mutation

**Op**: `config.mutate`

**Input**:

```json
{"config":{ "...": "any shape" }, "patches":[{"set":{...},"unset":[...]}]}
```

**Output**:

```json
{
  "config_patch": { "set": { "services.backend.port": 8083 }, "unset": [] },
  "notes": [ {"level":"info","message":"set services.backend.port"} ]
}
```

##### Why “patches” instead of “return config”?

If scripts return whole config blobs, merges become arbitrary and unreviewable. A structured patch gives deterministic composition.

#### Build (using config)

**Op**: `build.run`

**Input**:

```json
{"config":{...},"steps":["backend","web"],"env":{}}
```

**Output**:

```json
{
  "ok": true,
  "steps": [
    {"name":"backend","ok":true,"duration_ms":1234},
    {"name":"web","ok":true,"duration_ms":5678}
  ],
  "artifacts": { "backend_bin": "backend/dist/moments-server" }
}
```

#### Prepare (bootstrap/migrations/keys; using config)

**Op**: `prepare.run`

Same input/output shape as build (it’s a step runner).

#### Validate

**Op**: `validate.run`

**Input**: `{ "config": {...} }`

**Output**:

```json
{
  "valid": false,
  "errors": [
    {"code":"EPORT_IN_USE","message":"port 8083 is in use","fix":"devctl stop or choose another port"}
  ],
  "warnings": []
}
```

#### Launch + monitor (two modes)

We support two distinct patterns:

1) **Plan mode**: plugin returns a plan; orchestrator launches + supervises.
2) **Controller mode**: plugin itself is long-running and emits events.

**Op**: `launch.plan`

**Output**:

```json
{"services":[{"name":"backend","cwd":"backend","command":["go","run","./cmd/moments-server","serve"],"env":{"PORT":"8083"},"health":{"type":"http","url":"http://localhost:8083/rpc/v1/health","timeout_ms":30000}}]}
```

**Op**: `launch.controller.start`

**Output**:

```json
{"stream_id":"stream-launch-1","mode":"controller"}
```

Then the plugin emits `event` frames describing status (ready, unhealthy, etc.) until it exits or is cancelled.

#### Logs: tail/follow

**Op**: `logs.list`

**Output**:

```json
{"sources":[{"name":"backend","kind":"file","path":"tmp/backend-20260106.log"},{"name":"frontend","kind":"file","path":"tmp/frontend-20260106.log"}]}
```

**Op**: `logs.follow`

**Input**:

```json
{"source":"backend","since":"-5m"}
```

**Output**:

```json
{"stream_id":"stream-logs-1"}
```

Then the plugin emits `event` frames like:

```json
{"type":"event","stream_id":"stream-logs-1","event":"log","fields":{"source":"backend"},"message":"..."}
```

#### Commands: git-xxx style

**Op**: `commands.list`

**Output**:

```json
{"commands":[{"name":"db-reset","help":"Reset local DB (dangerous)","args_spec":[{"name":"--force","type":"bool"}]}]}
```

**Op**: `command.run`

**Input**:

```json
{"name":"db-reset","argv":["--force"],"config":{...}}
```

**Output**:

```json
{"ok":true,"exit_code":0}
```

The orchestrator exposes these commands as first-class subcommands:

- `devctl db-reset --force`

### Deterministic composition rules

To keep “many scripts” sane, the orchestrator defines merge rules:

- **Config**: apply `config_patch` in plugin order; later wins on the same key.
- **Build/prepare**: steps are additive; same step name collision warns (or errors under `--strict`).
- **Validate**: valid = AND; errors and warnings append.
- **Launch plan**: services merged by name; adding is always allowed; modifying existing requires explicit allowlist or strictness policy.
- **Commands**: union by name; collisions warn/error depending on strictness.

### Orchestrator-side API signatures (Go)

```go
type AnyConfig = map[string]any

type ConfigPatch struct {
    Set   map[string]any `json:"set,omitempty"`
    Unset []string       `json:"unset,omitempty"`
}

type ValidateError struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Fix     string         `json:"fix,omitempty"`
    Field   string         `json:"field,omitempty"`
    Details map[string]any `json:"details,omitempty"`
}

type ValidateResult struct {
    Valid    bool            `json:"valid"`
    Errors   []ValidateError `json:"errors,omitempty"`
    Warnings []ValidateError `json:"warnings,omitempty"`
}
```

## Design Decisions

### Why NDJSON?

Because it’s the easiest thing a bash script can implement correctly: “print one JSON line”. If we ever need to embed binary blobs or guarantee framing across very large payloads, we can introduce a v2 length-prefixed transport, but v1 should optimize for teammate ergonomics.

### Why config patches?

Because “config is anything” is fine, but “config merging is arbitrary” is not. A patch keeps the system deterministic and reviewable.

### Why two launch modes?

Most repos should use plan mode. Controller mode exists for cases where launch and monitor are inseparable (custom supervisors, remote sessions, etc.).

### Why commands as first-class?

So that the core tool doesn’t become a dumping ground for repo-specific workflows. Plugins can add commands and ship them as scripts.

## Implementation Plan

1. Implement plugin runner in the orchestrator:
   - discover plugins, validate handshake
   - request/response with timeouts
   - strict stdout-only protocol enforcement
2. Implement config patch application (set/unset by dotted path).
3. Implement phase pipeline wiring (minimum set):
   - `config.mutate` → `build.run` → `prepare.run` → `validate.run` → `launch.plan` → (optional) `logs.follow`
4. Implement command registration:
   - `commands.list` and `command.run`
5. Publish a `plugins/examples/` directory with copy/paste templates (bash + python).

## Script Templates (copy/paste)

### Minimal bash plugin: config.mutate

```bash
#!/usr/bin/env bash
set -euo pipefail

# stdout: protocol frames only
# stderr: human logs only

echo '{"type":"handshake","protocol_version":"v1","plugin_name":"example-bash","capabilities":{"ops":["config.mutate"]},"declares":{"side_effects":"none","idempotent":true}}'

while IFS= read -r line; do
  op=$(echo "$line" | jq -r '.op')
  rid=$(echo "$line" | jq -r '.request_id')

  if [[ "$op" == "config.mutate" ]]; then
    resp=$(jq -n --arg rid "$rid" '{
      type:"response",
      request_id:$rid,
      ok:true,
      output:{
        config_patch:{ set:{ "services.backend.port": 8083 }, unset:[] },
        notes:[{level:"info",message:"set services.backend.port=8083"}]
      }
    }')
    echo "$resp"
  else
    resp=$(jq -n --arg rid "$rid" --arg op "$op" '{
      type:"response",
      request_id:$rid,
      ok:false,
      error:{ code:"EUNSUPPORTED", message:("unsupported op: "+$op) }
    }')
    echo "$resp"
  fi
done
```

### Minimal python plugin: commands.list + command.run

```python
#!/usr/bin/env python3
import json, sys

def emit(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()

emit({
    "type": "handshake",
    "protocol_version": "v1",
    "plugin_name": "example-python",
    "capabilities": {"ops": ["commands.list", "command.run"]},
    "declares": {"side_effects": "process", "idempotent": False},
})

for line in sys.stdin:
    req = json.loads(line)
    rid = req.get("request_id")
    op  = req.get("op")

    if op == "commands.list":
        emit({
            "type":"response","request_id":rid,"ok":True,
            "output":{
                "commands":[{"name":"db-reset","help":"Reset local DB (dangerous)","args_spec":[{"name":"--force","type":"bool"}]}]
            }
        })
    elif op == "command.run":
        name = req["input"]["name"]
        if name != "db-reset":
            emit({"type":"response","request_id":rid,"ok":False,"error":{"code":"ENOENT","message":"unknown command"}})
            continue
        print("[db-reset] would run nuke+bootstrap here", file=sys.stderr)
        emit({"type":"response","request_id":rid,"ok":True,"output":{"exit_code":0}})
    else:
        emit({"type":"response","request_id":rid,"ok":False,"error":{"code":"EUNSUPPORTED","message":f\"unsupported op {op}\"}})
```

## Alternatives Considered

- **Plugins return whole config**: simplest, but merging becomes arbitrary and surprising.
- **Length-prefixed protocol**: more robust, but much harder for bash; consider in v2.
- **Everything as controller**: too heavy; plan mode covers most cases.

## Open Questions

- Do we want JSONPath-style paths (`$.services.backend.port`) instead of dotted keys?
- Should `launch.controller.start` be MVP, or can we defer it until needed?
- Should `logs.follow` emit plain lines or structured log records (source, level, ts)?

## References

- `design-doc/02-updated-architecture-moments-dev-go-stdio-phase-plugins.md`
- `design-doc/03-moments-as-plugins-repo-specific-phases-moments-server-dev-interface.md`
