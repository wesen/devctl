---
Title: 'Review: Go-Based startdev Replacement Architecture'
Ticket: MO-005-IMPROVE-STARTDEV
Status: active
Topics:
    - dev-tooling
    - scripts
    - bash
    - dev-environment
DocType: analysis
Intent: long-term
Owners:
    - team
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-01-06T13:16:33.707831285-05:00
WhatFor: ""
WhenToUse: ""
---

# Review: Go-Based startdev Replacement Architecture

## Executive summary (review verdict)

This design is directionally strong: it recognizes the real problem (a monolithic bash script) and proposes the right macro-shape (**phases**, **interfaces**, **commands**, **observability**, and **extensibility**). The main risks are **scope**, **protocol complexity**, and **unclear MVP boundaries**—especially around the stdio “plugin protocol” which, as written, is closer to “a distributed system in miniature” than a pragmatic extension mechanism.

If we constrain the scope and make a few protocol/contract changes (message framing, lifecycle, versioning, idempotency, deterministic merging, and a crisp “core vs plugin” boundary), this can become a genuinely reusable “generic tool” that’s easy for colleagues to extend.

## What’s good (keep these)

- **Clear separation by phase**: config/build/prepare/validate/launch/logs/commands matches how dev-env orchestration really works.
- **Explicit interfaces**: good for testing and incremental replacement (wrapping Makefile/bootstrap first, then moving into native Go).
- **Acknowledges the config story**: “use appconfig” is the right anchor so tooling matches backend runtime config behavior.
- **Health checks** as first-class: better than the current “port is listening” check.
- **Observability/log exposure**: treating logs as a product feature is correct for dev tooling.

## Major concerns / risks

### 1) MVP vs “platform” scope is not controlled

The doc includes:
- a full orchestrator,
- a process supervisor,
- log aggregation/search,
- a JSON-RPC-like plugin system,
- multi-phase plugin aggregation rules,
- plus cross-platform behavior.

That’s a lot of surface area to get “stable enough” to replace `startdev.sh`. Without a strict MVP, this will likely stall.

**Recommendation**: define an MVP that replaces `startdev.sh` with feature parity plus one “killer improvement” (e.g., real readiness checks + structured logs). Defer log search and “marketplace” features.

### 2) Configuration approach is inconsistent with what we learned

The design leans on “AppConfigProvider uses `moments-config` CLI” for compatibility. But the config analysis shows:
- `moments-config get` only supports a handful of bootstrap keys (repo_root/database_url/etc.).
- `startdev.sh` calls `moments-config get platform.mento-service-…` and `integrations.stytch…`, which are **not supported** by the CLI’s `get` command today.
- So current “config phase” largely falls back to defaults, meaning the real source of truth is *not actually used* for those keys.

**Recommendation**: in the Go replacement, integrate **directly** with `backend/pkg/appconfig` (blank-import registrations) and compute:
- the tool’s own config struct,
- plus the derived `VITE_*` env map.

Keep `moments-config` around for shell scripts, but don’t make the new tool dependent on it.

### 3) The plugin protocol is underspecified and too heavyweight in the wrong places

The protocol section is ambitious but has several sharp edges:

- **Framing**: “JSON messages over stdin/stdout” without framing rules is fragile.
  - If a plugin writes logs to stdout, it corrupts the protocol stream.
  - If responses span multiple lines, line-based reads break.
- **Lifecycle**: is a plugin a one-shot process-per-call, or a long-lived process that handles multiple requests?
- **Cancellation**: how does the orchestrator cancel a running plugin when the user hits Ctrl+C?
- **Streaming**: launch/log phases often need streaming updates; request/response alone is limiting.
- **Determinism**: the “merge results from multiple plugins” rules need to be explicit and reproducible.
- **Trust and safety**: “plugins run with same privileges” is fine, but you need guardrails:
  - timeouts and kill behavior,
  - working dir,
  - env injection policy,
  - and a strict separation of stdout (protocol) vs stderr (human logs).

**Recommendation**: simplify to a pragmatic “phase hook protocol”:
- Newline-delimited JSON (NDJSON) **or** length-prefixed frames.
- **stdout is reserved strictly for protocol frames**; **stderr is for human logs**.
- Mandatory **handshake** message (capabilities + protocol version).
- A small set of verbs: `handshake`, `run`, `stream` (optional), `shutdown`.
- One plugin binary may support multiple phases via advertised capabilities.

### 4) The design doesn’t state what is “core responsibility” vs “plugin responsibility”

If plugins can rewrite config, add services, add logs, etc., the orchestrator risks becoming non-deterministic and hard to support.

**Recommendation**: define a strict “core contract”:
- Core always does: repo-root resolution, config loading, baseline service definitions, process supervision, basic log capture, and user-facing CLI.
- Plugins may only:
  - augment config (not remove required keys),
  - add “prepare steps”,
  - add “validate checks”,
  - add services (but cannot delete core services unless explicitly allowed),
  - add log sources/filters.

Make “destructive” operations opt-in via flags or explicit allowlists.

## Specific, actionable improvements to the plugin protocol

### A) Message framing and channel separation

- **Protocol frames**: NDJSON on stdout (1 JSON object per line), or a length-prefixed format.
- **Human logs**: stderr only; orchestrator captures/stamps them with plugin name.

### B) Mandatory handshake

First message from plugin:

- `protocol_version`
- `plugin_name`
- `capabilities`: phases supported + optional features (streaming, side effects)
- `requires`: external tools (jq, psql, etc.) for validation UX

### C) Canonical request envelope

All requests include:
- `request_id`
- `phase`
- `operation` (e.g. `run`, `validate`, `augment`)
- `ctx`: repo_root, cwd, env_prefix, timestamps, user flags
- `inputs`: phase-specific payload

### D) Deterministic aggregation rules

Define per-phase merging semantics:
- **config**: deep-merge with “last plugin wins”, but require plugins to declare which keys they write.
- **vite_env**: union map; collisions must be explicit (either error or last-wins with warning).
- **validate**: overall valid = AND; errors accumulate; warnings accumulate.
- **launch**: allow only additive changes by default; modifications to core services require explicit opt-in.

### E) Side-effect declaration + idempotency

Plugins must declare whether they:
- are pure (read-only) vs side-effecting,
- are idempotent,
- require network/DB access.

This lets the orchestrator support:
- `--dry-run`,
- `--plan` output,
- “re-run safe” behavior.

## VITE configuration: what’s missing in the design

The design has “GetViteEnv”, but it should explicitly encode:
- which appconfig settings map to which `VITE_*` variables,
- and how Vite consumes them (`vite.config.mts` uses `VITE_IDENTITY_BACKEND_URL || VITE_IDENTITY_SERVICE_URL`).

**Recommendation**: make a single “ViteEnv derivation” function in core:
- takes typed `platform.Settings` + `stytchcfg.Settings` (and any others),
- returns a deterministic map,
- and prints a redacted view for the console.

Also: because the backend already serves `/config.js` via proxy, decide whether the dev tool should:
- rely on runtime config injection through `/config.js`, or
- keep using `VITE_STYTCH_PUBLIC_TOKEN`.

## Concrete MVP proposal (tight scope)

1. `moments-dev start`:
   - direct appconfig init (no moments-config shelling)
   - build: `make -C backend build` and `pnpm -C web install` (or keep current behavior)
   - prepare: `make -C backend bootstrap`
   - validate: check ports + `GET /rpc/v1/health`
   - launch: backend + frontend with robust process supervision
   - logs: tee to per-service files + optionally follow

2. `moments-dev status`, `moments-dev logs --follow`, `moments-dev stop`.

3. Plugin MVP:
   - config hooks + validate hooks only, one plugin per phase, NDJSON framing.

Once this is stable, expand to multi-plugin aggregation, launch hooks, and richer log features.

## Review checklist (what I’d want a second pair of eyes on)

- **Protocol framing**: ensure stdout protocol isn’t corrupted by plugin logs.
- **Process supervision semantics**: kill trees, handle Ctrl+C, cross-platform behavior.
- **Config correctness**: direct appconfig init, correct config paths and env prefix behavior.
- **Determinism**: stable plugin order + stable merging semantics.
- **Security posture**: explicit “trusted plugins only”, plus timeouts and env filtering.

## Conclusion

Keep the phase separation and the “dev tool as orchestrator” concept. Tighten scope for MVP, switch config to direct appconfig (not moments-config CLI), and simplify + harden the stdio plugin protocol (framing, handshake, deterministic merges, stderr for logs). With those changes, the design becomes both implementable and genuinely extensible for a team.
