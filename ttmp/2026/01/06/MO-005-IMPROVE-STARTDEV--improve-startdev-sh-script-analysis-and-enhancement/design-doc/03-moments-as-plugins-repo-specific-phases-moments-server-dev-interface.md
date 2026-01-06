---
Title: 'Moments as Plugins: Repo-Specific Phases + moments-server dev Interface'
Ticket: MO-005-IMPROVE-STARTDEV
Status: deprecated
Topics:
    - dev-tooling
    - scripts
    - bash
    - dev-environment
DocType: design-doc
Intent: deprecated
Owners:
    - team
RelatedFiles: []
ExternalSources: []
Summary: "How to make a generic dev-environment orchestrator and move Moments-specific logic into stdio plugins, plus a concrete moments-server 'dev' interface (manifest/vite-env/validate/plugin) to power those plugins."
LastUpdated: 2026-01-06T13:29:37.050528567-05:00
WhatFor: "Designing the split between a repo-agnostic dev tool and a Moments-specific plugin bundle, and defining the minimal moments-server enhancements needed to make that split correct and maintainable."
WhenToUse: "When implementing the plugin-first architecture or extending moments-server to provide the dev tooling contract."
---

## Executive summary (narrative)

The simplest way to make a **generic** dev-environment manager is to make the core tool *boring* on purpose: it should orchestrate phases, supervise processes, and present a great UX, but it should not know anything about “Moments”. All Moments knowledge—how to derive `VITE_*`, what health endpoints exist, how to bootstrap DB/migrations/keys, which services to run—should live outside the generic tool in a **Moments plugin bundle**.

That immediately raises a harder question: how do we prevent the plugin from turning into a second, drift-prone reimplementation of the backend’s configuration and conventions? The key is to augment `moments-server` with a small, stable, machine-readable **dev contract** (a manifest + vite env + validations). The plugin can then defer to `moments-server dev ...` as the authoritative source of truth rather than encoding assumptions that rot.

This document is an exhaustive guide for reviewers: it lays out the architecture split, includes diagrams and flows, gives API signatures and schema shapes, and provides pseudocode for `devctl`, the Moments plugin, and the `moments-server dev` subcommands.

## Table of contents

- **Context and goals**
- **Architecture overview (with diagrams)**
- **Phase contracts and data model**
- **Moments plugin bundle: shapes and pseudocode**
- **`moments-server dev` contract: commands, JSON schema, and Go API signatures**
- **Integration flows (sequence diagrams)**
- **Design decisions and trade-offs**
- **Implementation plan**
- **Open questions**

## Context and goals

We want teammates to extend dev startup without editing a central `startdev.sh` and without making the generic tool import Moments packages. But we still need correctness:

- configuration must match backend `appconfig` semantics (YAML merge + env overrides)
- `VITE_*` derivation must align with `web/vite.config.mts`
- readiness must be more than “port is listening” (we should check `/rpc/v1/health`)

## Architecture overview

### Big picture diagram

```
                ┌──────────────────────────────────────────────┐
                │                 devctl (generic)              │
                │----------------------------------------------│
                │ phase runner + plugin mgr + process supervisor│
                │ logs UX + status + restart/rebuild + dry-run  │
                └───────────────┬──────────────────────────────┘
                                │ stdio protocol (framed JSON)
                                │
                    ┌───────────▼───────────┐
                    │  moments plugin bundle │
                    │  (repo-specific)       │
                    │------------------------│
                    │ config/build/prepare/  │
                    │ validate/launch/logs   │
                    └───────────┬───────────┘
                                │ exec / call contract
                                │
                ┌───────────────▼──────────────────────────────┐
                │            moments-server dev (contract)       │
                │----------------------------------------------│
                │ dev manifest | dev vite-env | dev validate     │
                │ (optional) dev plugin (stdio)                  │
                └───────────────────────────────────────────────┘
```

The invariant is: **devctl stays generic**. Anything that requires repo knowledge goes into either:

- the Moments plugin bundle, or
- the `moments-server dev` contract.

### Phase pipeline diagram (control flow)

```
devctl start
  |
  +--> phase: config     (plugin)  -> PhaseConfigResult
  +--> phase: build      (plugin)  -> BuildPlan
  +--> phase: prepare    (plugin)  -> PreparePlan
  +--> phase: validate   (plugin)  -> ValidationPlan/Results
  +--> phase: launch     (plugin)  -> Services[]
  +--> devctl runs services + monitors readiness
  +--> phase: logs       (plugin)  -> LogPlan
  |
  '-> devctl prints status + log hints, optionally follows logs
```

## Phase contracts and data model

To keep the system reviewable and deterministic, we should define a small set of reusable data structures. The protocol can still pass JSON, but the shapes should correspond to versioned structs.

### Core JSON types (schema v1)

The generic tool understands these shapes regardless of repo.

#### `ServiceSpec`

```json
{
  "name": "backend",
  "cwd": "backend",
  "command": ["go", "run", "./cmd/moments-server", "serve"],
  "env": { "PORT": "8083" },
  "health": { "type": "http", "url": "http://localhost:8083/rpc/v1/health", "timeout_ms": 30000 }
}
```

#### `HealthCheckSpec`

```json
{ "type": "http", "url": "http://localhost:8083/rpc/v1/health", "timeout_ms": 30000, "interval_ms": 500 }
```

Types:
- `http`: GET to URL, success on 2xx and/or response body contract
- `tcp`: port-listen

#### `CommandPlan`

```json
{ "cwd": "backend", "command": ["make", "bootstrap"], "env": { "PORT": "8083" } }
```

#### `ViteEnv`

```json
{ "VITE_IDENTITY_BACKEND_URL": "http://localhost:8083", "VITE_IDENTITY_SERVICE_URL": "http://localhost:8083" }
```

### Go API signatures (generic tool)

These are the interfaces `devctl` would use internally.

```go
type PhaseName string

const (
    PhaseConfig   PhaseName = "config"
    PhaseBuild    PhaseName = "build"
    PhasePrepare  PhaseName = "prepare"
    PhaseValidate PhaseName = "validate"
    PhaseLaunch   PhaseName = "launch"
    PhaseLogs     PhaseName = "logs"
)

type ServiceSpec struct {
    Name    string            `json:"name"`
    Cwd     string            `json:"cwd"`
    Command []string          `json:"command"`
    Env     map[string]string `json:"env,omitempty"`
    Health  *HealthCheckSpec  `json:"health,omitempty"`
}

type HealthCheckSpec struct {
    Type       string `json:"type"` // "http" | "tcp"
    URL        string `json:"url,omitempty"`
    Port       int    `json:"port,omitempty"`
    TimeoutMS  int    `json:"timeout_ms,omitempty"`
    IntervalMS int    `json:"interval_ms,omitempty"`
}

type CommandPlan struct {
    Cwd     string            `json:"cwd"`
    Command []string          `json:"command"`
    Env     map[string]string `json:"env,omitempty"`
}
```

## Moments plugin bundle

### What does the plugin actually do?

In a plugin-first world, the Moments plugin’s job is to translate “generic phases” into “this repo’s plan”, using `moments-server dev` as the canonical source of truth. Concretely:

- For **config/launch/logs**, the plugin can often just call `moments-server dev manifest` and return the relevant parts.
- For **vite env**, it can call `moments-server dev vite-env` or use the `vite_env` field from the manifest.
- For **validate**, it can call `moments-server dev validate` for “repo-authored validations” and/or add checks (like “is pnpm installed”) that live outside the backend.
- For **build/prepare**, it can return the relevant command plans (which devctl executes), or execute them itself if you want “active” plugins (not recommended for MVP).

### Recommended implementation form: one Go plugin binary

Put it under `moments/backend/cmd/moments-dev-plugin/` so it can reuse appconfig logic and share dependencies with the backend. The plugin speaks the stdio protocol and returns JSON payloads defined above.

#### Plugin API surface (pseudocode)

```text
on start:
  emit handshake
  loop:
    read framed request
    switch phase/op:
      config/run:    return config result (via moments-server dev manifest/vite-env)
      build/run:     return build plan (commands)
      prepare/run:   return prepare plan (commands)
      validate/run:  return validate plan/results (via moments-server dev validate + extra checks)
      launch/run:    return services (from manifest)
      logs/run:      return log plan (from manifest or conventions)
```

#### Example: `config` phase output

```json
{
  "vite_env": { "VITE_IDENTITY_BACKEND_URL": "http://localhost:8083" },
  "services": [
    { "name": "backend", "cwd": "backend", "command": ["go","run","./cmd/moments-server","serve"], "env": { "PORT":"8083" } }
  ]
}
```

### Stdio protocol requirements (so teammates don’t get burned)

If colleagues extend via scripts, you want the protocol to be robust and boring:

- **stdout**: protocol frames only
- **stderr**: human logs
- framing: NDJSON or length-prefixed
- mandatory handshake
- deterministic ordering and merging (handled by devctl)

### Example protocol frames (NDJSON)

Handshake (plugin → orchestrator):

```json
{"type":"handshake","protocol_version":"v1","plugin_name":"moments","capabilities":{"phases":["config","build","prepare","validate","launch","logs"]}}
```

Request (orchestrator → plugin):

```json
{"type":"request","request_id":"req-1","phase":"config","op":"run","ctx":{"repo_root":"/abs/path","cwd":"/abs/path","deadline_ms":30000},"input":{}}
```

Response:

```json
{"type":"response","request_id":"req-1","ok":true,"output":{"vite_env":{},"services":[]},"messages":[{"level":"info","message":"derived vite env"}]}
```

## Augmenting `moments-server` (the dev contract)

### Why `moments-server dev` is the right “contract home”

`moments-server` already has:

- appconfig schemas/registrations (typed config is the real source of truth),
- readiness health endpoint (`/rpc/v1/health`) already registered,
- knowledge of “what the backend is” and which port it uses.

Putting the contract here means:

- the plugin doesn’t re-encode appconfig rules,
- changes to config structure have one canonical place to update,
- reviewers can see exactly what the repo promises to dev tooling.

### CLI surface: `moments-server dev ...`

Add a `dev` command group:

- `moments-server dev manifest`
- `moments-server dev vite-env`
- `moments-server dev validate`
- `moments-server dev plugin` (optional)

All commands share the same flags for config resolution:

```text
--repo-root <path>
--env-prefix <prefix>         (default "MOMENTS")
--config-env <env>            (e.g. development/staging/production)
--config-override <path>      (optional extra YAML merged last)
--format json|env             (where relevant)
--show-secrets                (where relevant)
```

### `moments-server dev manifest` (JSON schema v1)

#### What it contains (prose)

The manifest is a declarative “how to run Moments in dev” document. It should be versioned and intentionally small:

- service definitions (backend and optionally web)
- build and prepare command plans
- derived `vite_env`
- health check specs

It should *not* contain process IDs or runtime state; it is a contract, not a supervisor.

#### JSON example (with URL strings wrapped in backticks in prose, but raw in JSON)

```json
{
  "schema_version": "v1",
  "repo": { "name": "moments", "root": "/abs/path/to/moments" },
  "services": [
    {
      "name": "backend",
      "cwd": "backend",
      "command": ["go", "run", "./cmd/moments-server", "serve"],
      "env": { "PORT": "8083" },
      "health": { "type": "http", "url": "http://localhost:8083/rpc/v1/health", "timeout_ms": 30000 }
    },
    {
      "name": "web",
      "cwd": "web",
      "command": ["pnpm", "run", "dev", "--", "--port", "5173"],
      "env": { "VITE_IDENTITY_BACKEND_URL": "http://localhost:8083" },
      "health": { "type": "tcp", "port": 5173, "timeout_ms": 30000 }
    }
  ],
  "build": {
    "backend": [{ "cwd": "backend", "command": ["make", "build"] }],
    "web": [{ "cwd": "web", "command": ["pnpm", "install", "--prefer-offline"] }]
  },
  "prepare": {
    "bootstrap": [{ "cwd": "backend", "command": ["make", "bootstrap"] }]
  },
  "vite_env": {
    "VITE_IDENTITY_BACKEND_URL": "http://localhost:8083",
    "VITE_IDENTITY_SERVICE_URL": "http://localhost:8083",
    "VITE_BACKEND_URL": "http://localhost:8083"
  }
}
```

### Go API signatures for manifest generation (inside `moments-server`)

```go
type DevManifest struct {
    SchemaVersion string            `json:"schema_version"`
    Repo          RepoInfo          `json:"repo"`
    Services      []ServiceSpec     `json:"services"`
    Build         map[string][]CommandPlan `json:"build"`
    Prepare       map[string][]CommandPlan `json:"prepare"`
    ViteEnv       map[string]string `json:"vite_env"`
}

type RepoInfo struct {
    Name string `json:"name"`
    Root string `json:"root"`
}

func BuildDevManifest(ctx context.Context, repoRoot string, cfg DevConfig) (*DevManifest, error)
func DeriveViteEnv(cfg TypedConfig) map[string]string
```

Where `TypedConfig` is a small struct of the settings you actually use:

```go
type TypedConfig struct {
    Platform platform.Settings
    Server   servercfg.Settings
    Stytch   stytchcfg.Settings
}
```

### Pseudocode: implementing `dev manifest`

```text
cmd moments-server dev manifest:
  repoRoot = ResolveRepoRoot(flag)
  configPaths = appconfig.DefaultConfigPaths(repoRoot, env, includeLocal, overridePaths...)
  appconfig.InitializeFromConfigFiles(envPrefix, configPaths)

  typed.Platform = appconfig.Must[platform.Settings]()
  typed.Server   = appconfig.Must[servercfg.Settings]()
  typed.Stytch   = appconfig.Must[stytchcfg.Settings]()   // if needed

  viteEnv = DeriveViteEnv(typed)  // single canonical function

  manifest.Services = [
     backend spec (go run serve; env PORT from typed.Server.Port; health URL uses /rpc/v1/health),
     web spec (pnpm dev; env = viteEnv; port from flag or default 5173)
  ]
  manifest.Build = backend/web build plans
  manifest.Prepare = bootstrap plan

  print JSON to stdout
```

### `moments-server dev vite-env`

This command exists so the plugin (or humans) can get just the env needed for Vite in a stable way, without parsing a larger manifest.

API signature:

```go
func DeriveViteEnv(cfg TypedConfig) map[string]string
```

Output formats:

- `--format json` (default): `{"VITE_FOO":"..."}`
- `--format env`: lines like `export VITE_FOO='...'`
- token redaction by default; `--show-secrets` prints actual values.

### `moments-server dev validate`

This should be a structured check runner that returns machine-readable results. Think of it as “what does the repo consider a correct dev environment?”.

Suggested output:

```json
{
  "valid": false,
  "errors": [
    {"component":"config","message":"missing platform.mento-service-identity-base-url","fix":"set in config/app/local.yaml or MOMENTS_PLATFORM_MENTO_SERVICE_IDENTITY_BASE_URL"}
  ],
  "warnings": [],
  "checks": {
    "appconfig": {"ok": true},
    "db": {"ok": false, "skipped": true},
    "keys": {"ok": true},
    "health": {"ok": false, "skipped": true}
  }
}
```

### `moments-server dev plugin` (optional)

If we want maximum correctness and minimum moving parts, `moments-server` itself can speak the stdio protocol and answer phase requests. This makes `moments-server` the canonical Moments plugin executable.

Trade-off: heavier binary and longer startup for the plugin process; but correctness is excellent because it runs the same code as the backend.

## Integration flows (sequence diagrams)

### Flow 1: devctl → moments plugin (which calls moments-server dev manifest)

```
devctl                         moments-plugin                     moments-server
  |                                 |                                 |
  |-- config/run ------------------->|                                 |
  |                                 |-- exec: dev manifest ---------->|
  |                                 |<----------- JSON manifest -------|
  |<----------- config result -------|                                 |
  |-- launch/run ------------------->|                                 |
  |<----------- services[] ----------|                                 |
  |-- start processes (backend/web)  |                                 |
  |-- validate readiness (health)    |                                 |
```

### Flow 2: devctl → moments-server dev plugin (no separate plugin binary)

```
devctl                         moments-server (dev plugin mode)
  |                                 |
  |-- handshake/requests ----------->|
  |<-- responses (config/launch) ----|
  |-- start processes -------------->|
```

## Design decisions and trade-offs (full prose)

Putting a versioned dev manifest in `moments-server` is the key “anti-drift” move. It ensures that as appconfig evolves, the contract evolves with it, and plugins can remain thin translators rather than fragile reimplementations. The generic tool remains reusable because it only speaks phase protocols and runs commands; it never needs to import Moments code.

The plugin split does introduce one new discipline: you must treat the protocol and manifest schema as APIs. That means versioning, backward compatibility, and careful handling of secrets. But this is a good trade-off because it forces clarity around what “dev environment” means for the repo.

## Implementation plan (incremental)

### Step 1: Ship the contract first

Implement:
- `moments-server dev vite-env --format json`
- `moments-server dev manifest`

This is high leverage and unlocks everything else.

### Step 2: Implement a minimal Moments plugin

Implement a Go plugin binary that:
- answers `config` by calling `dev manifest`
- answers `launch` by returning manifest services

### Step 3: Expand phases

Add:
- `build` and `prepare` plans
- `validate` (calls `moments-server dev validate` + adds non-backend checks)
- `logs` plan (log file locations, follow hints)

### Step 4 (optional): consolidate into `moments-server dev plugin`

If we want fewer moving parts, move plugin logic into `moments-server`’s dev plugin mode and retire the separate plugin binary.

## Open questions (for reviewers)

- **Manifest scope**: should `dev manifest` include the web service definition, or should web remain purely a plugin responsibility?
- **Secrets**: should dev rely on `VITE_STYTCH_PUBLIC_TOKEN`, or should it rely purely on runtime `/config.js` and keep tokens out of env?
- **Protocol framing**: NDJSON vs length-prefixed—do we need the robustness of length-prefixed early?

## References

- `design-doc/02-updated-architecture-moments-dev-go-stdio-phase-plugins.md`
- `analysis/02-moments-config-and-configuration-phase-analysis.md`
- `analysis/03-review-go-based-startdev-replacement-architecture.md`
- `scripts/startdev.sh`
