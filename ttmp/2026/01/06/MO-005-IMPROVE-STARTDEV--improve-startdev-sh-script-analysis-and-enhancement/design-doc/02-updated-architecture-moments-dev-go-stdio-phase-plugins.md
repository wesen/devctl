---
Title: 'Updated Architecture: moments-dev (Go) + Stdio Phase Plugins'
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
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-06T13:22:56.697523573-05:00
WhatFor: ""
WhenToUse: ""
---

## Updated Architecture: `moments-dev` (Go) + stdio phase plugins

### Executive summary

This is the updated, implementable design for replacing `moments/scripts/startdev.sh` with a Go CLI (**`moments-dev`**) while keeping the system extensible via **stdio phase plugins**. It directly addresses the review findings by:

- **Tight MVP scope**: feature parity plus real readiness checks + structured logs; defers “platform” extras (log search/indexing, marketplace, etc.).
- **Correct configuration**: `moments-dev` integrates **directly** with `backend/pkg/appconfig` (blank-import registrations) and deterministically derives `VITE_*` env vars.
- **Hardened plugin protocol**: explicit framing (NDJSON or length-prefixed), **stdout reserved for protocol**, **stderr for human logs**, mandatory handshake, deterministic ordering/merging, cancellation rules, and side-effect declarations enabling `--dry-run`.

### Problem statement

`startdev.sh` orchestrates backend + frontend, but:

- It’s monolithic bash (hard to test, evolve, and extend).
- Lifecycle/readiness is weak (port checks, kill-by-port).
- The “configuration phase” is inconsistent with backend `appconfig`. In particular, the script calls `moments-config get` with dot-notation keys that the CLI does not return, so it often falls back to defaults.
- Extensibility requires editing a central script instead of adding plugins.

We want: correctness (same config semantics as backend), observability, composability (restart/rebuild/logs), and a safe extensibility model for teammates.

### Goals / non-goals

#### Goals

- Replace `startdev.sh` with `moments-dev start` (backend + Vite frontend).
- Use backend `appconfig` as the single source of truth for dev config.
- Deterministically derive `VITE_*` env for Vite + proxies.
- Provide commands: `start`, `stop`, `status`, `logs`, `restart`, `rebuild`.
- Provide phase plugins over stdio that are easy to implement in any language.

#### Non-goals (initially)

- Full log indexing/search across history (tail/follow is enough for MVP).
- Untrusted plugin sandboxing.
- Multi-project “framework” ambitions beyond repo-root + config path abstraction.

### Proposed solution (architecture)

`moments-dev` is a Go CLI (Cobra) built around seven phases:

1. **configuration**
2. **building**
3. **prepare**
4. **validate**
5. **launch + monitor**
6. **logs**
7. **commands**

Each phase has:

- a **core implementation** (owned by `moments-dev`)
- optional **plugin hooks** (stdio protocol)
- explicit **inputs/outputs**, so the pipeline is testable and composable

### MVP definition (explicit)

#### `moments-dev start` (MVP)

- **Config**
  - resolve repo root
  - initialize `backend/pkg/appconfig` directly (same precedence rules as backend)
  - derive canonical `VITE_*` env map
- **Build**
  - backend: `make -C backend build` (or equivalent targeted builds)
  - frontend: `pnpm -C web install --prefer-offline` + `pnpm -C web generate-version`
- **Prepare**
  - `make -C backend bootstrap` (keeps DB/migrations/keys behavior)
- **Validate**
  - ports available (or explicit opt-in kill behavior)
  - backend readiness via `GET /rpc/v1/health`
  - frontend readiness via port-listen (optional HTTP GET `/`)
- **Launch**
  - start backend + frontend as process groups
  - manage lifecycle (SIGTERM → wait → SIGKILL)
- **Logs**
  - timestamped log files under `moments/tmp/`
  - `logs --follow` UX

#### MVP commands

- `moments-dev stop`
- `moments-dev status`
- `moments-dev logs [--follow] [--service backend|frontend]`
- `moments-dev restart [backend|frontend|all]`
- `moments-dev rebuild [backend|frontend|all] [--restart]`

### Configuration architecture (direct `appconfig`)

#### Core rule

`moments-dev` uses `backend/pkg/appconfig` directly:

- imports `github.com/mento/moments/backend/pkg/appconfig`
- blank-imports `github.com/mento/moments/backend/pkg/appconfig/registrations`
- calls `appconfig.InitializeFromConfigFiles(envPrefix, configPaths)`
- reads typed settings via `appconfig.Must[T]()`

This keeps dev tooling aligned with backend runtime config semantics.

#### Config sources and precedence

Sources:
- YAML: `config/app/base.yaml`, optional env file, optional `local.yaml`
- env vars: `MOMENTS_*`
- CLI flags: `--repo-root`, `--config-env`, `--env-prefix`, `--config-override <path>`

Precedence (low → high):
1. base.yaml
2. env.yaml (optional)
3. local.yaml (optional in dev)
4. `MOMENTS_*` env vars
5. CLI overrides

### VITE configuration (explicit derivation)

All `VITE_*` comes from a single core function (deterministic map) that is derived from typed settings.

#### Initial mapping

- `VITE_IDENTITY_BACKEND_URL` = `platform.Settings.MentoServiceIdentityBaseURL` (fallback `http://localhost:8083`)
- `VITE_IDENTITY_SERVICE_URL` = same value (compat alias because `vite.config.mts` checks both)
- `VITE_BACKEND_URL` = **one canonical choice**:
  - if local dev uses a single backend for both identity + rpc, set equal to identity URL
  - else map from `platform.Settings.MentoServicePublicBaseURL`
- `VITE_STYTCH_PUBLIC_TOKEN` = `stytchcfg.Settings.StytchPublicToken` (optional; redact in console)

`moments-dev` prints a redacted summary and logs the derived map under `--verbose`.

### Process lifecycle and monitoring

- spawn backend/frontend in their own process groups
- stop sequence: SIGTERM → grace wait → SIGKILL
- readiness:
  - backend: `GET http://localhost:<port>/rpc/v1/health`
  - frontend: port-listen + optional HTTP GET

MVP state: in-memory while `moments-dev` runs; later we can add optional persisted state for cross-shell status.

### Extensibility: stdio phase plugins (hardened protocol)

#### Goals for plugins

- easy to implement in any language
- deterministic behavior
- no stdout contamination
- clear cancellation semantics
- safe defaults (plugins can add; destructive changes require explicit allow)

#### Transport and framing (choose one)

Option A (**NDJSON**, MVP-friendly):
- stdout: 1 JSON object per line (protocol frames)

Option B (**length-prefixed**, more robust):
- `Content-Length: <n>\r\n\r\n<json>`

**Hard rule**:
- plugin **stdout** = protocol only
- plugin **stderr** = human logs

#### Mandatory handshake

First protocol frame from plugin:

```json
{"type":"handshake","protocol_version":"v1","plugin_name":"acme-custom","capabilities":{"phases":["config","validate"],"streaming":false},"declares":{"side_effects":"none","idempotent":true}}
```

Orchestrator rejects plugins without valid handshake.

#### Canonical request/response envelope

Requests:

```json
{"type":"request","request_id":"...","phase":"config","op":"run","ctx":{"repo_root":"...","cwd":"...","env_prefix":"MOMENTS","deadline_ms":30000},"input":{...}}
```

Responses:

```json
{"type":"response","request_id":"...","ok":true,"output":{...},"messages":[{"level":"info","message":"..."}]}
```

Errors:

```json
{"type":"response","request_id":"...","ok":false,"error":{"code":"EPLUGIN","message":"...","details":{...}}}
```

#### Cancellation

On Ctrl+C or timeout:
- orchestrator SIGTERMs the plugin process
- escalates to SIGKILL after a grace period
- (optional future) send a `shutdown` request if we keep plugins long-lived

#### Plugin lifecycle (MVP choice)

MVP: **one-shot execution per phase** (spawn plugin, run once, exit). This avoids needing streaming, multiplexing, or background lifetime management.

Later: long-lived daemon plugins are possible, but not MVP.

#### Deterministic ordering

- order is config file order, or lexical order when scanning directories
- orchestrator prints resolved order under `--verbose`

#### Merge rules (deterministic)

- **config**
  - plugins declare `writes: ["vite_env.VITE_FOO", "config.backend_port"]`
  - deep-merge; later plugin wins; collisions warn (or error under `--strict`)
- **vite_env**
  - union map; collisions warn/error depending on strictness
- **prepare/build**
  - additive steps; core steps remain unless user explicitly disables them
- **validate**
  - overall valid = AND; errors/warnings accumulate
- **launch**
  - plugins may add services by default
  - modifying core services requires `--allow-modify-core-services` + explicit allowlist
- **logs**
  - additive sources/filters

#### Side effects and `--dry-run`

Plugins declare:
- `side_effects`: `none|filesystem|network|db|process`
- `idempotent`: true/false

`moments-dev --dry-run` runs only `side_effects=none` plugins and emits a plan.

### Design decisions (what changed vs prior design)

- **Direct appconfig config** is core (not `moments-config` shelling).
- **Framing + stdout/stderr separation** is mandatory for plugins.
- **MVP boundaries** are explicit (avoid over-scope).
- **Core-vs-plugin responsibility** is defined; destructive plugin behavior is opt-in.

### Alternatives considered

- Keep bash + helpers: still leaves testability/extensibility problems.
- docker-compose/systemd: adds tooling burden; doesn’t align with current Makefile/bootstrap workflow.
- rely on `moments-config` CLI: tempting, but doesn’t supply the needed dot-notation keys and adds overhead; keep it for legacy scripts only.

### Implementation plan

#### Phase 0: lock contracts

- choose framing (NDJSON vs length-prefixed)
- write v1 protocol spec (handshake + envelope + stderr logs)
- define config→VITE derivation function and tests

#### Phase 1: MVP orchestration

- implement `start/stop/status/logs`
- direct appconfig init + VITE derivation
- wrap `make bootstrap` and pnpm flows
- readiness checks (`/rpc/v1/health`)
- timestamped log files

#### Phase 2: commands + minimal plugins

- implement `restart/rebuild`
- implement config + validate plugin hooks (single plugin per phase)

#### Phase 3: expand plugins + polish

- multi-plugin aggregation
- launch/log hooks
- optional persisted state

### Open questions

- framing choice: NDJSON vs length-prefixed
- do we need cross-shell status via persisted state?
- do we prefer VITE token injection or runtime `/config.js` for stytch in dev?
- strictness defaults: should collisions be errors by default?

### References

- `design-doc/01-go-based-startdev-replacement-architecture.md` (previous design)
- `analysis/02-moments-config-and-configuration-phase-analysis.md` (config findings)
- `analysis/03-review-go-based-startdev-replacement-architecture.md` (review)
- `scripts/startdev.sh` (current baseline behavior)
