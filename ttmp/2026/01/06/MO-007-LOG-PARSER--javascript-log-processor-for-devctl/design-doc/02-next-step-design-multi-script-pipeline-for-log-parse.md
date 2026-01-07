---
Title: 'Next Step Design: Multi-Script Pipeline for log-parse'
Ticket: MO-007-LOG-PARSER
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: devctl/cmd/log-parse/main.go
      Note: CLI will grow module list and validate/print-pipeline modes
    - Path: devctl/examples/log-parse
      Note: Example scripts; will expand to pipeline directory
    - Path: devctl/pkg/logjs/module.go
      Note: Current MVP; will be refactored into multi-module Pipeline
ExternalSources: []
Summary: Evolve log-parse from single-script to a multi-script pipeline (multi-parse + shared filter/transform stages), with better validation, introspection, and per-module stats.
LastUpdated: 2026-01-06T19:25:59-05:00
WhatFor: Make the JS log parsing system more consequential by supporting real-world setups (multiple formats, shared enrichment, reusable scripts) without requiring devctl integration yet.
WhenToUse: Use when implementing the next iteration of log-parse/logjs or reviewing pipeline semantics and safety boundaries.
---


# Next Step Design: Multi-Script Pipeline for log-parse

## Executive Summary

The MVP `log-parse` supports a single JavaScript file that registers one module via `register({ ... })`. The next meaningful step is to support **many scripts** and a **pipeline** that composes them deterministically:

- **Many parsers**: support multiple `parse(line, ctx)` modules (e.g. JSON logs + logfmt logs + ad-hoc regex logs) and choose first match or all matches.
- **Shared event stages**: support many `filter(event, ctx)` / `transform(event, ctx)` modules applied to every parsed event (enrichment, normalization, dropping noise).
- **Better ergonomics**: `--module` repeated flags, `--modules-dir`, and a config file mode.
- **Safer by default**: keep filesystem/exec capabilities off by default; `require()` remains off in this ticket (sandboxing is tracked in `MO-008-REQUIRE-SANDBOX`).
- **Operational clarity**: add `--validate` and `--print-pipeline` modes, and per-module stats.

This design keeps the runtime model simple and safe:

- one `goja.Runtime` per worker (still `--workers=1` initially)
- all scripts loaded into the runtime; each script calls `register(...)` exactly once
- pipeline is implemented in Go by calling the hooks in order, not by allowing scripts to call each other directly

## Problem Statement

The MVP is useful for one-off experiments, but real log workflows quickly need:

1) **Multiple formats** in a single stream (application logs, infra logs, proxy logs).
2) **Reusable enrichment/normalization** steps shared by multiple parsers.
3) **Composability** across teams/services (small scripts that do one thing well).
4) **Confidence tools**: validate scripts before running; print what’s loaded; show per-module error counts and drops.

Without multi-script support, users either:

- jam everything into one large script (hard to maintain), or
- want `require()` / multi-file scripts (which is doable but needs sandboxing).

We want to get the “consequential” value first: many scripts *loaded by Go* and composed as a pipeline, without opening `require()` yet.

## Proposed Solution

### Pipeline model

We define two conceptual stages:

1) **Parse stage** (line → 0..N events)
2) **Event stage** (event → 0..1 event, via filter/transform)

ASCII overview:

```
input lines
   │
   ▼
┌──────────────────────────────────────────────────────────┐
│ Parse stage (one or many modules)                         │
│  - parseMode: first | all                                 │
│  - each module: parse(line, ctx) => event|null            │
└───────────────┬───────────────────────────────────────────┘
                │ events (0..N)
                ▼
┌──────────────────────────────────────────────────────────┐
│ Event stage (many modules, deterministic order)           │
│  - filter(event, ctx) => boolean (drop if false)          │
│  - transform(event, ctx) => event|null (drop if null)     │
└───────────────┬───────────────────────────────────────────┘
                │ events
                ▼
output (ndjson/pretty)
```

### Multi-script loading UX

Add to `log-parse`:

- `--module <path>` (repeatable): load scripts in the order provided.
- `--modules-dir <dir>` (repeatable): load all `*.js` under dir (non-recursive MVP; recursive future), in lexicographic order.
- `--config <path>`: YAML/JSON config describing pipeline (optional; recommended once stable).

Examples:

```bash
# multiple scripts in explicit order
log-parse \
  --module ./parsers/json.js \
  --module ./parsers/logfmt.js \
  --module ./stages/drop-debug.js \
  --module ./stages/add-service.js \
  --input ./mixed.log

# load every script under a directory (lexicographic)
log-parse --modules-dir ./pipeline/ --input ./mixed.log
```

### Module contract updates (JS)

We keep the same `register({ ... })` contract, but add optional metadata fields to clarify intent:

```js
register({
  name: "drop-debug",
  kind: "stage",        // optional: parser | stage | both
  order: 100,           // optional: numeric ordering within the stage

  filter(event, ctx) { return event.level !== "DEBUG"; },
});
```

Rules:

- `name` still required and must be unique within a run.
- A module may provide:
  - `parse` (participates in parse stage)
  - `filter` and/or `transform` (participates in event stage)
- `kind` is advisory; actual participation is determined by which hooks exist.
- `order` is optional; if present, it sorts modules within their stage (stable by script load order as tie-breaker).

### Parse mode

We need a clear rule when multiple parsers exist:

- `--parse-mode first` (default): stop at the first parser that returns an event; produce at most 1 event per line.
- `--parse-mode all`: run all parsers and emit one event per parser that matches.

This keeps behavior predictable and allows users to opt into “fan-out”.

### Validation and introspection

Add commands/flags to make this safe and debuggable:

- `log-parse validate --module ...`:
  - load/compile scripts
  - verify exactly one `register(...)` call per script
  - verify unique names
  - verify that at least one module provides `parse`
  - print a pipeline summary (which hooks exist)
- `--print-pipeline`:
  - print module list + stage classification + execution order
- `--stats`:
  - print per-module `Stats` and global totals on exit (to stderr)

### Go implementation approach (incremental, minimal refactor)

Current MVP has `logjs.Module` representing one script + one module.
For multi-script, we introduce:

```go
// devctl/pkg/logjs/pipeline.go (new)
type Pipeline struct {
	Modules []*LoadedModule
	Parse   []*LoadedModule // modules with parse hook
	Stages  []*LoadedModule // modules with filter/transform hooks
	Options PipelineOptions
}

type LoadedModule struct {
	Name string
	Path string

	HasParse      bool
	HasFilter     bool
	HasTransform  bool
	HasInit       bool
	HasShutdown   bool
	HasOnError    bool

	ParseFn      goja.Callable
	FilterFn     goja.Callable
	TransformFn  goja.Callable
	InitFn       goja.Callable
	ShutdownFn   goja.Callable
	OnErrorFn    goja.Callable

	State *goja.Object // per-module state object
	Stats Stats        // per-module stats (not shared)
}

type PipelineOptions struct {
	HookTimeout time.Duration
	ParseMode   string // first | all
}
```

Key runtime decision:

- **One goja runtime per pipeline instance** (per worker).
- Load all scripts into the same runtime.
- Each script calling `register(config)` captures:
  - the config object for that module
  - the Go callables for each hook
  - a new per-module state object

This avoids cross-runtime event marshalling and keeps performance reasonable.

### Pseudocode: pipeline execution

```text
init:
  for m in Modules in deterministic order:
    if m.init exists: call init(ctx{hook:"init", state:m.state})

for each input line:
  events = []
  for p in Parse modules in order:
    v = p.parse(line, ctx{hook:"parse", state:p.state})
    if v != null:
      events.append(normalize(v))
      if parseMode == "first": break

  for each event in events:
    for s in Stage modules in order:
      if s.filter exists:
        keep = s.filter(event, ctx{hook:"filter", state:s.state})
        if !keep: drop event; continue next event
      if s.transform exists:
        v2 = s.transform(event, ctx{hook:"transform", state:s.state})
        if v2 == null: drop event; continue next event
        event = normalize(v2)
    emit event

shutdown:
  for m in Modules in reverse order:
    if m.shutdown exists: call shutdown(ctx{hook:"shutdown", state:m.state})
```

### “Further things that make sense” (next step backlog)

Once multi-script is in place, the next high-leverage additions are:

1) **Config file** (`log-parse.yaml`) for repeatability:
   - list of modules (paths/directories)
   - parseMode, timeout, output format
   - optional per-module enable/disable

2) **Module discovery and packaging**:
   - `--modules-dir` recursive
   - glob patterns
   - `log-parse init` to scaffold a pipeline directory

3) **Structured error output / dead-letter**:
   - `--errors ndjson` to emit error records as NDJSON (to stderr or a file)
   - include module name, hook name, source line number, raw line

4) **Per-module debug toggles**:
   - `--trace module=foo` to show inputs/outputs for that module (bounded sampling)

5) **Sandboxed require()** (tracked separately):
   - enable `require("./helpers")` safely (see `MO-008-REQUIRE-SANDBOX`)

## Design Decisions

### 1) Compose scripts in Go, not in JS

We intentionally avoid JS-to-JS module loading for now (no `require()`), and instead load scripts from Go and compose via deterministic pipeline rules.

Rationale:
- simpler security story
- simpler runtime behavior (no module cache surprises)
- fewer decisions required about filesystem access and path traversal

### 2) One runtime per pipeline instance

Rationale:
- avoids expensive conversion between runtimes
- keeps `goja.Value` objects usable across hook calls

### 3) Parse mode explicitly configurable

Rationale:
- “first match” is often what users want for mixed-format logs
- “all match” is sometimes useful for fan-out and enrichment

### 4) Per-module state objects

Rationale:
- scripts can safely keep counters/caches without stepping on each other
- we avoid global state collisions and encourage small modules

## Alternatives Considered

### 1) Require every script to provide the full hook set

Rejected:
- makes small “stage-only” scripts awkward
- encourages copy/paste and bloat

### 2) One goja runtime per script

Rejected:
- too expensive and complicated (event must be marshalled between runtimes)
- difficult to keep consistent semantics for JS objects (Date, RegExp, etc.)

### 3) Enable require() and let scripts compose themselves

Deferred:
- feasible, but needs sandboxing and explicit policy (tracked in `MO-008-REQUIRE-SANDBOX`)

## Implementation Plan

1. Extend CLI to accept repeated `--module` and `--modules-dir`.
2. Refactor `devctl/pkg/logjs`:
   - split “runtime + hook execution + normalization” into reusable helpers
   - introduce `Pipeline` type (multi-module)
3. Implement `validate`/`--print-pipeline`.
4. Add tests:
   - multi-parser: parseMode first/all behavior
   - stage ordering and order stability
   - per-module state isolation
5. Add example pipeline directory under `devctl/examples/log-parse/pipeline/`.

## Open Questions

1) Should we allow parse modules to emit multiple events per line (array return) in this phase?
2) Should `order` live in JS module metadata, or in the CLI/config file only?
3) How do we want to represent “module version” and compatibility in the future?

## References

- MVP design: `design-doc/01-mvp-design-javascript-log-parser-goja.md`
- MVP engine: `/home/manuel/workspaces/2026-01-06/log-parser-module/devctl/pkg/logjs/module.go`
- MVP CLI: `/home/manuel/workspaces/2026-01-06/log-parser-module/devctl/cmd/log-parse/main.go`
- Require sandbox follow-up: `../../MO-008-REQUIRE-SANDBOX--sandbox-goja-nodejs-require-for-log-parse/design-doc/02-analysis-sandboxing-goja-nodejs-require.md`
